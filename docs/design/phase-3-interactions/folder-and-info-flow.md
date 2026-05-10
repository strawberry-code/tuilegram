# Folder Sidebar + Chat Info — Sequence Diagrams (Step 29)

Flussi runtime della **folder sidebar** (`F`) e dell'**overlay chat
info** (`i`) introdotti nello Step 29. Complementare agli statechart
in [`../phase-2-behavioral/folder-sidebar.md`](../phase-2-behavioral/folder-sidebar.md)
e [`../phase-2-behavioral/chat-info.md`](../phase-2-behavioral/chat-info.md).

Sette scenari coprono i path interessanti:

1. **Folder toggle on → select folder → toggle off** — happy path
   sidebar.
2. **Folder filter preserva la chat aperta** — l'utente apre Work,
   ma la chat di "Mom" (private, non in Work) resta visibile sulla
   destra.
3. **Chat info open con cache hit completo** — `i` su una private
   chat con bio gia fetched.
4. **Chat info open con cache miss** — `i` triggera lazy completion
   `users.getFullUser`.
5. **Chat info live update** — `UserStatusMsg` durante info open
   refresha il dot.
6. **Stale completion** — l'utente chiude e riapre su una chat
   diversa; il completion della prima arriva e viene droppato.
7. **F durante chat info open** — `F` consumato dall'overlay (UX
   guard, ADR-017 §D5).

## 1. Folder toggle on → select "Work" → toggle off

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App.Update
    participant FOLD as FolderModel
    participant CL as ChatListModel
    participant V as View (renderer)

    Note over APP: state: folderSidebarVisible=FALSE,<br/>selectedFolderID=0 ("All Chats"),<br/>folders=[All Chats, Personal, Work, Channels]

    U->>APP: keypress 'F' (focus on chat list)
    APP->>FOLD: FolderToggleMsg
    FOLD->>FOLD: folderSidebarVisible := TRUE<br/>folderCursor := index(selectedFolderID) = 0<br/>focus := folders (Visible.Browsing)
    APP->>V: re-layout: 3 panels (folders | chatlist | conv)
    V-->>U: sidebar appears at left,<br/>"All Chats" highlighted

    U->>APP: keypress 'j' (3 times)
    APP->>FOLD: FolderCursorMsg{+1} x3
    FOLD->>FOLD: folderCursor: 0 → 1 → 2
    V-->>U: cursor on "Work"

    U->>APP: keypress 'Enter'
    APP->>FOLD: FolderSelectMsg{folderID=2}
    FOLD->>FOLD: selectedFolderID := 2
    APP->>CL: re-filter chat list (sync, no RPC)<br/>filtered := dialogs.filter(c -> c.ID in folders[2].IncludedChats)
    CL->>CL: cursor reset to 0 if previous was filtered out
    APP->>V: re-render chat list panel
    V-->>U: chat list now shows only "Work" chats

    U->>APP: keypress 'F'
    APP->>FOLD: FolderToggleMsg
    FOLD->>FOLD: folderSidebarVisible := FALSE<br/>selectedFolderID PRESERVED (still 2)<br/>focus := chatList
    APP->>V: re-layout: 2 panels (chatlist | conv)
    V-->>U: sidebar disappears,<br/>chat list STILL filtered to Work

    Note over APP: filter persists; F again would reopen<br/>sidebar with cursor on "Work"
```

**Punto chiave**: `selectedFolderID` è **preservato** attraverso il
toggle off. Riaprire la sidebar mostra la stessa selezione. L'utente
deve esplicitamente selezionare "All Chats" per rimuovere il filtro.

## 2. Folder filter preserva la chat aperta

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App.Update
    participant CL as ChatListModel
    participant CONV as ConversationModel
    participant V as View

    Note over APP: state: activeChatID="Mom" (private),<br/>selectedFolderID=0 ("All Chats"),<br/>"Mom" NOT in folders[2] ("Work").IncludedChats

    U->>APP: keypress 'F' → 'j' x2 → 'Enter'
    APP->>CL: FolderSelectMsg{folderID=2}
    CL->>CL: filter: "Mom" filtered OUT<br/>activeChatID UNCHANGED ("Mom")
    APP->>V: render chat list (Work only)<br/>render conversation panel (Mom messages)
    V-->>U: left: Work chats (Mom NOT visible)<br/>right: Mom conversation STILL shown

    Note over CONV: Conversation panel UNAFFECTED by folder filter.<br/>activeChatID is independent of chatList visibility.<br/>(Invariant ACTIVE_CHAT_INVARIANT in folders_chatinfo.tla)

    U->>APP: keypress 'i' (chat info)
    Note over APP: works because activeChatID="Mom" != nil
    APP->>APP: ChatInfoOpenMsg<br/>guard activeChatID != nil OK<br/>guard activeOverlay = none OK
    Note over APP: → Scenario 3 below (cache hit)
```

**Punto chiave**: il filtro folder agisce sulla **chat list** (left
panel), non sulla **conversazione attiva** (right panel). L'utente può
quindi continuare a leggere/scrivere alla chat aperta anche se non
appare nella lista filtrata, e può aprire la chat info via `i`.

## 3. Chat info open — cache hit completo

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App.Update
    participant INFO as ChatInfoModel
    participant CACHE as Domain Cache
    participant V as View

    Note over APP: state: activeChatID="John",<br/>activeOverlay=none,<br/>cache has User{John}.{Bio,Phone,Status} all populated

    U->>APP: keypress 'i'
    APP->>INFO: ChatInfoOpenMsg
    INFO->>INFO: chatInfoTarget := "John"<br/>activeOverlay := chatInfo
    INFO->>CACHE: read User{John}.{Bio, Phone, Username, Status}
    CACHE-->>INFO: full snapshot
    INFO->>INFO: chatInfoCard := build from cache<br/>(no fetchFullUserCmd needed)
    APP->>V: render Modal (compact, right-anchored)
    V-->>U: overlay shown, card populated:<br/>"John Doe / @john / ● Online<br/>Phone: +1 555-...<br/>Bio: Developer<br/>Shared Media [24] / Files [8]"

    U->>APP: keypress 'Esc'
    APP->>INFO: ChatInfoCloseMsg
    INFO->>INFO: activeOverlay := none<br/>chatInfoTarget := nil
    APP->>V: unmount Modal
    V-->>U: overlay disappears
```

**Punto chiave**: nel path felice, l'overlay è istantaneo (no spinner,
no RPC). Tutti i campi vengono dal `Domain Cache` materializzato da
`DialogsLoadedMsg`.

## 4. Chat info open — cache miss (lazy completion)

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App.Update
    participant INFO as ChatInfoModel
    participant CACHE as Domain Cache
    participant CMD as fetchFullUserCmd
    participant TG as Telegram (gotd/td)
    participant V as View

    Note over APP: cache has User{Mom}.{Username,Phone,Status},<br/>but Bio == "" (not yet fetched)

    U->>APP: keypress 'i'
    APP->>INFO: ChatInfoOpenMsg
    INFO->>INFO: chatInfoTarget := "Mom"<br/>activeOverlay := chatInfo
    INFO->>CACHE: read User{Mom}
    CACHE-->>INFO: snapshot (Bio="")
    INFO->>INFO: chatInfoCard := build with Bio="—" placeholder
    APP->>V: render Modal (overlay shown with "—" Bio placeholder)
    V-->>U: overlay shown, Bio "⠧ loading..."

    APP->>CMD: spawn fetchFullUserCmd("Mom")
    CMD->>TG: api.UsersGetFullUser({Mom})
    TG-->>CMD: tg.UserFull{Bio, About, ...}
    CMD->>APP: ChatInfoCompletionMsg{chatID="Mom", fields={Bio:"Loves jazz."}}
    APP->>INFO: handle ChatInfoCompletionMsg
    INFO->>INFO: guard chatID == chatInfoTarget ✓<br/>merge: chatInfoCard.Bio := "Loves jazz."<br/>also write-through to CACHE for next open
    APP->>V: re-render Modal body
    V-->>U: Bio updated to "Loves jazz."

    U->>APP: keypress 'Esc'
    APP->>INFO: ChatInfoCloseMsg → close
```

**Punto chiave**: lazy completion è **best-effort**. L'overlay si apre
sempre subito; il bio si materializza in un secondo frame. Se la RPC
fallisce, status-bar mostra dim message (non blocca l'overlay).

## 5. Live update durante chat info open

```mermaid
sequenceDiagram
    participant TG as Telegram (goroutine)
    participant APP as App.Update
    participant INFO as ChatInfoModel
    participant V as View

    Note over APP: state: activeOverlay=chatInfo,<br/>chatInfoTarget="John",<br/>chatInfoCard.OnlineStatus={online: TRUE}

    Note over TG: peer "John" goes offline (UpdateUserStatus)
    TG->>APP: p.Send(UserStatusMsg{userID=John, status=offline, lastSeen=now})
    APP->>INFO: UserStatusMsg
    INFO->>INFO: guard chatInfoTarget == John (private chat with John) ✓<br/>chatInfoCard.OnlineStatus := offline
    APP->>V: re-render Modal section "Identity"
    V-->>V: dot ● (green) → ○ (dim)<br/>"● Online" → "Last seen 2 min ago"
```

Stesso pattern per `ChatUpdateMsg` (es. titolo gruppo cambiato,
member count incrementato).

## 6. Stale completion — utente chiude e riapre su chat diversa

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App.Update
    participant INFO as ChatInfoModel
    participant CMD as fetchFullUserCmd

    Note over APP: state: activeChatID="Mom"

    U->>APP: keypress 'i'
    APP->>INFO: ChatInfoOpenMsg
    INFO->>INFO: chatInfoTarget := "Mom"<br/>activeOverlay := chatInfo
    APP->>CMD: spawn fetchFullUserCmd("Mom")
    Note over CMD: RPC in flight (slow network, ~2s)

    U->>APP: keypress 'Esc' (immediate, before RPC returns)
    APP->>INFO: ChatInfoCloseMsg
    INFO->>INFO: chatInfoTarget := nil<br/>activeOverlay := none

    U->>APP: keypress 'j' x5 → Enter (open different chat "Bob")
    APP->>APP: activeChatID := "Bob"

    U->>APP: keypress 'i'
    APP->>INFO: ChatInfoOpenMsg
    INFO->>INFO: chatInfoTarget := "Bob"<br/>activeOverlay := chatInfo
    APP->>CMD: spawn fetchFullUserCmd("Bob") (NEW Cmd)

    Note over CMD: RPC for "Mom" returns (~2s elapsed)
    CMD->>APP: ChatInfoCompletionMsg{chatID="Mom", fields={Bio:"..."}}
    APP->>INFO: handle ChatInfoCompletionMsg
    INFO->>INFO: guard chatID="Mom" == chatInfoTarget="Bob" → STALE → no-op
    Note over INFO: chatInfoCard for Bob UNCHANGED.<br/>(However, the cache for Mom CAN still be updated<br/>as a side effect — it's a benign write-through.)

    Note over CMD: RPC for "Bob" returns
    CMD->>APP: ChatInfoCompletionMsg{chatID="Bob", fields={Bio:"..."}}
    APP->>INFO: handle ChatInfoCompletionMsg
    INFO->>INFO: guard chatID="Bob" == chatInfoTarget="Bob" ✓<br/>merge into chatInfoCard
```

**Punto chiave**: il completion stale è **droppato a render-time**
(non muta `chatInfoCard`), ma la cache per "Mom" può comunque essere
aggiornata (write-through benigno, accelera il prossimo open su Mom).
Pattern formalmente verificato in `folders_chatinfo.tla` invariante
`STALE_COMPLETION_DROP`.

## 7. F durante chat info open — overlay consuma

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App.Update
    participant INFO as ChatInfoModel

    Note over APP: state: activeOverlay=chatInfo, folderSidebarVisible=FALSE

    U->>APP: keypress 'F' (intends to toggle folders)
    APP->>APP: dispatch key to active overlay (chat info)
    APP->>INFO: KeyPressMsg('F')
    INFO->>INFO: 'F' not in keybinding set for chat info → CONSUME (no-op)
    Note over INFO: Decision rationale (ADR-017 §D5):<br/>opening sidebar while overlay open<br/>would destabilize layout perception.
    APP-->>U: nothing happens visually
    Note over U: must Esc first, then F
```

**Punto chiave**: la sidebar **non** è un overlay e tecnicamente non
viola il mutex `activeOverlay`. Tuttavia per coerenza UX, l'overlay
chat info **consuma** `F` (no-op). L'utente deve chiudere l'overlay
(`Esc`) prima di aprire la sidebar. Decisione in
[ADR-017 §D5](../phase-6-decisions/ADR-017-chat-info-data-source.md).

## Riepilogo invarianti runtime

| Invariante | Esempio scenario |
|------------|------------------|
| `selectedFolderID` preservato attraverso toggle | Scenario 1 |
| `activeChatID` indipendente dalla chat list filtrata | Scenario 2 |
| Open chat info è sincrono, completion best-effort | Scenari 3, 4 |
| Live updates re-render solo se `chatID == chatInfoTarget` | Scenario 5 |
| Stale completion droppato a return-time | Scenario 6 |
| Sidebar coesiste con chat info, ma `F` durante info è UX-consumed | Scenario 7 |

## Cross-links

- Statechart sidebar: [`../phase-2-behavioral/folder-sidebar.md`](../phase-2-behavioral/folder-sidebar.md)
- Statechart chat info: [`../phase-2-behavioral/chat-info.md`](../phase-2-behavioral/chat-info.md)
- TLA+: [`../phase-4-concurrency/folders_chatinfo.tla`](../phase-4-concurrency/folders_chatinfo.tla)
- ADR folder source: [ADR-016](../phase-6-decisions/ADR-016-folder-source-and-filtering.md)
- ADR chat info data source + Modal reuse: [ADR-017](../phase-6-decisions/ADR-017-chat-info-data-source.md)
- Mutex pattern: [ADR-015 §D3](../phase-6-decisions/ADR-015-command-palette-whichkey-help.md)
- Drop-stale pattern: [ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md), [ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md)
- Pipeline step: [`../development-pipeline.md` §Step 29](../development-pipeline.md)
