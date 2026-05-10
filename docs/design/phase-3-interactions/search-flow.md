# Search Flow — Sequence Diagrams (Step 26)

Flusso runtime della **ricerca globale** introdotta nello Step 26.
Complementare allo statechart in
[`../phase-2-behavioral/search-overlay.md`](../phase-2-behavioral/search-overlay.md).

Tre scenari principali coprono i path interessanti:

1. Happy path con risultati e jump in-chat (stessa chat).
2. Happy path con jump cross-chat.
3. Race debounce + stale RPC drop.
4. Cancel via `Esc` durante RPC in volo (deroga ADR-007).
5. Errore RPC.

## 1. Happy path — typing → debounce → results → jump (stessa chat)

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App.Update
    participant OVL as SearchOverlay
    participant TICK as tea.Tick (300ms)
    participant CMD as searchCmd (goroutine)
    participant TG as Telegram Client
    participant SRV as Telegram Server
    participant CV as Conversation viewport

    U->>APP: '/'
    APP->>OVL: SearchOpenMsg → mount, latestQueryID=0
    Note over OVL: state = Open.Idle

    U->>OVL: type 'h'
    OVL->>OVL: query="h", latestQueryID=1
    OVL->>TICK: schedule SearchDebounceFiredMsg{qID=1} @ now+300ms
    Note over OVL: state = Open.Typing

    U->>OVL: type 'e' (within 300ms)
    OVL->>OVL: query="he", latestQueryID=2
    OVL->>TICK: schedule SearchDebounceFiredMsg{qID=2} @ now+300ms
    Note over OVL: tick qID=1 still pending,<br/>will be benign (qID < latestQueryID)

    Note over TICK: tick qID=1 fires
    TICK->>OVL: SearchDebounceFiredMsg{qID=1}
    OVL->>OVL: qID=1 < latestQueryID=2 → no-op (stale debounce)

    Note over TICK: tick qID=2 fires (300ms after 'e')
    TICK->>OVL: SearchDebounceFiredMsg{qID=2}
    OVL->>OVL: qID=2 == latestQueryID, query="he" != ""
    OVL->>CMD: spawn searchCmd("he", qID=2)
    Note over OVL: state = Open.Searching.InFlight

    CMD->>TG: api.MessagesSearchGlobal(q="he", limit=N)
    TG->>SRV: messages.searchGlobal
    SRV-->>TG: messages.MessagesSlice{[hit1, hit2, ...]}
    TG-->>CMD: []SearchHit
    CMD-->>OVL: SearchResultMsg{qID=2, hits=[...], err=nil}

    OVL->>OVL: qID=2 == latestQueryID → apply
    OVL->>OVL: state.hits = [...], cursor = 0
    Note over OVL: state = Open.Results.ShowingHits
    OVL-->>U: render hits list

    U->>OVL: j (cursor down)
    OVL->>OVL: SearchCursorMsg{+1} → cursor=1
    OVL-->>U: highlight shifted

    U->>OVL: Enter
    OVL->>APP: SearchSubmitMsg{hit=hits[1]}
    APP->>OVL: OverlayCloseMsg → unmount, latestQueryID++
    APP->>APP: JumpToMessageMsg{chatID=hit.ChatID, msgID=hit.MessageID}
    Note over APP: hit.ChatID == activeChatID:<br/>no chat reload needed
    APP->>CV: scroll to msgID, center viewport, flash highlight
    APP-->>U: re-render (overlay gone, viewport at message)
```

## 2. Happy path — jump cross-chat

```mermaid
sequenceDiagram
    participant U as User
    participant APP as App.Update
    participant OVL as SearchOverlay
    participant CMD as loadChatCmd
    participant TG as Telegram Client
    participant CV as Conversation viewport
    participant CL as ChatList

    U->>OVL: Enter on hit (hit.ChatID != activeChatID)
    OVL->>APP: SearchSubmitMsg{hit}
    APP->>OVL: OverlayCloseMsg → unmount
    APP->>APP: JumpToMessageMsg{chatID, msgID}
    APP->>APP: activeChatID != chatID → spawn loadChatCmd(chatID, centerOn=msgID)
    APP->>CL: select chat row(chatID), scroll list if off-screen
    APP-->>U: re-render (chatlist updated, viewport "Loading…")

    CMD->>TG: api.MessagesGetHistory(peer=chatID, offset_id=msgID, ±N)
    TG-->>CMD: []Message (window around msgID)
    CMD-->>APP: MessagesLoadedMsg{chatID, messages, centerOn=msgID}

    APP->>CV: SetContent(messages), scroll cursor to msgID
    APP->>CV: flash highlight (1s) on msgID
    APP-->>U: re-render (target message centered)
```

`loadChatCmd` carries an extra `centerOn` field (or App stores a
`pendingJump` map) so that, when `MessagesLoadedMsg` arrives, the
viewport knows to center on `msgID` rather than scroll-to-bottom.
Out-of-scope of Step 26 design: the exact wire-up of `centerOn` —
behavioral expectation is enough; implementation chooses the cleanest
path.

## 3. Race — RPC stale dropped via queryID

```mermaid
sequenceDiagram
    participant U as User
    participant OVL as SearchOverlay
    participant TICK as tea.Tick
    participant CMD1 as searchCmd qID=1
    participant CMD2 as searchCmd qID=2
    participant TG as Telegram Client

    U->>OVL: type 'a' → qID=1, schedule tick
    Note over OVL: 300ms pass
    TICK->>OVL: SearchDebounceFiredMsg{qID=1}
    OVL->>CMD1: spawn searchCmd("a", qID=1)
    CMD1->>TG: api.MessagesSearchGlobal("a")
    Note over OVL: state = Searching.InFlight (qID=1)

    U->>OVL: type 'b' (RPC ancora in volo) → qID=2, schedule tick
    Note over OVL: latestQueryID = 2,<br/>RPC qID=1 ancora pending

    Note over TG: server slow su qID=1
    TICK->>OVL: SearchDebounceFiredMsg{qID=2} (300ms dopo 'b')
    OVL->>CMD2: spawn searchCmd("ab", qID=2)
    CMD2->>TG: api.MessagesSearchGlobal("ab")

    TG-->>CMD1: hits per "a" (return finalmente)
    CMD1-->>OVL: SearchResultMsg{qID=1, hits, nil}
    OVL->>OVL: qID=1 < latestQueryID=2 → DROP (stale RPC)
    Note over OVL: hits non applicati, render invariato

    TG-->>CMD2: hits per "ab"
    CMD2-->>OVL: SearchResultMsg{qID=2, hits, nil}
    OVL->>OVL: qID=2 == latestQueryID → APPLY
    OVL-->>U: render hits per "ab"
```

Punto chiave: la RPC stale (`qID=1`) **non viene cancellata server-side**
(non c'è API per farlo dopo il send), ma il suo risultato è scartato
silenziosamente dal check `qID == latestQueryID` nel main loop. Il costo
è una RPC sprecata; il beneficio è che non serve cancellation primitive.

Vedi [ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md)
per la giustificazione formale e il modello TLA+
[`../phase-4-concurrency/search.tla`](../phase-4-concurrency/search.tla)
per le invarianti `STALE_RESULT_DROP` e `MONOTONIC_QUERYID`.

## 4. Cancel via Esc durante RPC in volo (deroga ADR-007)

```mermaid
sequenceDiagram
    participant U as User
    participant OVL as SearchOverlay
    participant APP as App.Update
    participant CMD as searchCmd qID=k
    participant TG as Telegram Client

    Note over OVL: state = Searching.InFlight (qID=k)
    U->>OVL: Esc
    OVL->>APP: OverlayCloseMsg
    APP->>OVL: unmount, latestQueryID++ (now k+1)
    APP-->>U: re-render (overlay gone)

    Note over CMD: RPC ancora in volo lato client
    TG-->>CMD: hits (eventualmente)
    CMD-->>APP: SearchResultMsg{qID=k, hits, nil}
    APP->>APP: overlay è Closed, OR qID=k < latestQueryID=k+1 → DROP
    Note over APP: nessun side-effect visibile
```

A differenza di forward picker (`ADR-007`: `Esc` ignorato durante RPC),
l'overlay search **accetta** `Esc`:

- La RPC è **read-only** (`messages.searchGlobal` non muta nulla
  server-side). Cancellare il context lato client è pulito; non lasciar
  attendere l'utente è UX-friendly.
- L'effetto del risultato pendente è già **annullato** dal pattern
  `latestQueryID` (incrementato in `OverlayCloseMsg`). Niente toast
  silenzioso, niente state inconsistente.
- Modellato in `search.tla` invariante `CLOSE_INVALIDATES_INFLIGHT`.

Decisione formale: [ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md).

## 5. Errore RPC

```mermaid
sequenceDiagram
    participant U as User
    participant OVL as SearchOverlay
    participant CMD as searchCmd
    participant TG as Telegram Client
    participant SRV as Telegram Server

    Note over OVL: query="hello", qID=k, in flight
    CMD->>TG: api.MessagesSearchGlobal("hello")
    TG->>SRV: messages.searchGlobal
    SRV-->>TG: RPC_ERROR (e.g. FLOOD_WAIT_42)
    TG-->>CMD: error
    CMD-->>OVL: SearchResultMsg{qID=k, hits=nil, err=FLOOD_WAIT_42}

    OVL->>OVL: qID=k == latestQueryID → apply error path
    OVL-->>U: in-overlay toast "Search failed: FLOOD_WAIT 42s · keep typing to retry"
    Note over OVL: state = Open.Error<br/>overlay stays open, user can keep typing
```

L'utente può continuare a digitare; ogni nuova keystroke schedula un
nuovo debounce, eventualmente un nuovo `searchCmd`. Non c'è retry
automatico (lo Step 26 evita complessità: l'utente decide).

## Mapping tea.Cmd

Aggiornamento alla tabella "Mapping tea.Cmd" in
[`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md):

| Azione utente / evento | Cmd | API gotd/td | Result Msg |
|------------------------|-----|-------------|------------|
| `/` | (no Cmd, immediato) | — | `SearchOpenMsg` |
| char/backspace nell'overlay | `searchDebounceCmd(qID)` (= `tea.Tick(300ms)`) | — | `SearchDebounceFiredMsg{qID}` |
| `SearchDebounceFiredMsg` con qID fresh + non-empty query | `searchCmd(query, qID)` | `api.MessagesSearchGlobal` | `SearchResultMsg{qID, hits, err}` |
| `Enter` su hit | `tea.Batch(OverlayCloseMsg, JumpToMessageMsg)` | — | (synchronous chain) |
| `JumpToMessageMsg` con chat diversa | `loadChatCmd(chatID, centerOn=msgID)` | `api.MessagesGetHistory` | `MessagesLoadedMsg` |

## Cross-links

- Statechart: [`../phase-2-behavioral/search-overlay.md`](../phase-2-behavioral/search-overlay.md)
- Concurrency invariants: [`../phase-4-concurrency/search.tla`](../phase-4-concurrency/search.tla)
- Pipeline: [`../development-pipeline.md` §Step 26](../development-pipeline.md)
- Decisione debounce + stale: [ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md)
- Pattern ascendente (overlay RPC): [`forward-flow.md`](forward-flow.md)
- Pattern ascendente (TTL + tick): [`typing-flow.md`](typing-flow.md)
