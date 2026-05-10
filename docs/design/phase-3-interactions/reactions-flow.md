# Reactions & System Messages — Sequence Diagrams (Step 25)

Flusso runtime delle reazioni live e dei system messages.
Complementare allo statechart in
[`../phase-2-behavioral/reactions-and-system.md`](../phase-2-behavioral/reactions-and-system.md).

## 1. Live reaction update — happy path

Scenario di test n.3 dello Step 25 (verbatim italian):
*"Aggiungi una reazione da un altro client → appare in tempo reale".*

```mermaid
sequenceDiagram
    participant OTHER as Altro Client<br/>(Telegram Desktop)
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine<br/>(gotd/td dispatcher)
    participant CONV as convert/reactions.go
    participant APP as App.Update<br/>(bubbletea main loop)
    participant CV as Conversation viewport
    participant BUB as Bubble renderer

    OTHER->>SRV: messages.sendReaction(msgID=42, emoji="👍")
    SRV->>SRV: aggrega counters per messaggio 42
    SRV->>TG: UpdateMessageReactions{<br/>  peer, msgID=42,<br/>  reactions: [<br/>    {emoji:"👍", count:3, chosen:false},<br/>    {emoji:"❤️", count:2, chosen:false}<br/>  ]<br/>}
    TG->>CONV: convert(tg.MessageReactions)
    CONV->>CONV: 1) filter ReactionEmoji entries<br/>2) skip ReactionCustomEmoji (out of scope)<br/>3) sort by Count desc, Emoji asc tie-break
    CONV-->>TG: []domain.Reaction (ordered)
    TG->>APP: program.Send(<br/>  ReactionsUpdatedMsg{<br/>    chatID, msgID=42,<br/>    reactions: [...]<br/>  })

    APP->>CV: locate message #42 in viewport
    Note over CV: O(N) scan tra messaggi visibili<br/>(la chat aperta tipicamente <500 msg)
    CV->>CV: m.Reactions = reactions (replace)
    CV->>BUB: re-render bubble #42
    BUB->>BUB: render bubble + appendReactionsRow(m.Reactions)
    Note over BUB: layout invariato: bubble dimensions<br/>NON cambiano per reactions row<br/>(la riga è "extra" sotto)
```

**Punti notevoli**:

- Lo **snapshot** di reactions è completo: `m.Reactions := newReactions`
  (replace, non merge). Il server è autoritativo.
- Nessuna RPC verso Telegram: lo Step 25 è puramente consumer di update.
- Il messaggio è già nel viewport (è una chat aperta). Se non è visibile
  (es. arrivato per chat chiusa), il `ReactionsUpdatedMsg` viene
  comunque applicato al modello dati: il prossimo `MessagesLoadedMsg`
  porterà già il nuovo state.

## 2. Initial render — chat open con reactions storiche

Scenario di test n.1 dello Step 25 (verbatim italian):
*"Messaggi con reazioni → riga emoji sotto".*

```mermaid
sequenceDiagram
    participant U as User
    participant CL as ChatList
    participant APP as App.Update
    participant CMD as loadHistoryCmd
    participant TG as Telegram Goroutine
    participant SRV as Telegram Server
    participant CONV as convert/messages.go
    participant CV as Conversation viewport

    U->>CL: Enter su chat #100
    CL->>APP: ChatSelectedMsg{chatID:100}
    APP->>CMD: loadHistoryCmd(100)
    CMD->>TG: api.MessagesGetHistory
    TG->>SRV: MTProto request
    SRV-->>TG: messages.Messages{<br/>  messages: [<br/>    Message{id:42, msg:"hello", reactions: [...]},<br/>    Message{id:41, msg:"hi", reactions: nil},<br/>    MessageService{id:40, action: chatAddUser}<br/>  ]<br/>}
    TG->>CONV: convert each
    CONV->>CONV: per ogni Message:<br/>  - extract reactions (if any)<br/>  - sort by Count desc<br/>  - extract IsService + ServiceText (if action)
    CONV-->>TG: []domain.Message
    TG-->>APP: MessagesLoadedMsg{chatID:100, messages:[...]}
    APP->>CV: SetMessages(messages)
    CV->>CV: per ogni m:<br/>  kind = classify(m)<br/>  switch kind {<br/>    system → centered dim<br/>    media  → bubble + media<br/>    text   → bubble<br/>  }<br/>  if len(m.Reactions) > 0 && kind != system:<br/>    appendReactionsRow(m.Reactions)
    Note over CV: render iniziale completo,<br/>nessun re-render incrementale
```

## 3. System message ingest — un utente entra nel gruppo

Scenario di test n.2 dello Step 25 (verbatim italian):
*"Un utente entra nel gruppo → system message centrato".*

```mermaid
sequenceDiagram
    participant ALICE as Alice<br/>(altro client)
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine
    participant CONV as convert/messages.go
    participant APP as App.Update
    participant CV as Conversation viewport

    ALICE->>SRV: joined group #100<br/>(via invite link / add)
    SRV->>SRV: emit MessageService<br/>action: MessageActionChatAddUser{users:[Alice.id]}
    SRV->>TG: UpdateNewMessage{<br/>  msg: tg.MessageService{<br/>    id: 99,<br/>    peer: 100,<br/>    action: MessageActionChatAddUser{...}<br/>  }<br/>}
    TG->>CONV: convert(tg.MessageService)
    CONV->>CONV: discriminate on tg type:<br/>  case *tg.Message → IsService=false<br/>  case *tg.MessageService → IsService=true<br/>  formatActionText(action) → "Alice joined"
    CONV-->>TG: domain.Message{<br/>  ID:99, IsService:true,<br/>  ServiceText:"Alice joined",<br/>  Text:"", Media:nil, Reactions:nil<br/>}
    TG->>APP: program.Send(NewMessageMsg{m})

    APP->>CV: append message
    CV->>CV: kind(m) = "system"<br/>(IsService check has priority 1)
    CV->>CV: render via SystemBranch:<br/>  centered + dim<br/>  "── Alice joined ──"
    Note over CV: NO reactions row<br/>NO timestamp inline<br/>NO reply bar
```

## 4. Reactions update race — message edit + reactions concorrenti

Edge case: due update arrivano molto vicini (server può ordinare
arbitrariamente).

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine
    participant APP as App.Update
    participant CV as Conversation

    Note over SRV: state msg#42:<br/>text="hello", reactions=[👍 1]

    par concurrent server-side
        SRV->>TG: UpdateEditMessage{id:42, text:"hello edited"}
    and
        SRV->>TG: UpdateMessageReactions{id:42, reactions:[👍 2]}
    end

    Note over TG: gotd dispatcher invoca handlers in ordine<br/>arbitrario MA single-threaded per peer<br/>(no concurrent dispatch sullo stesso peer)

    TG->>APP: MessageEditedMsg{id:42, text:"hello edited"}
    APP->>CV: m.Text = "hello edited"<br/>m.Reactions invariato (= [👍 1])

    TG->>APP: ReactionsUpdatedMsg{id:42, [👍 2]}
    APP->>CV: m.Reactions = [👍 2]<br/>m.Text invariato (= "hello edited")

    Note over CV: render finale corretto:<br/>text "hello edited" + 👍 2<br/>indipendentemente dall'ordine
```

**Invariante chiave** (vedi
[`reactions.tla`](../phase-4-concurrency/reactions.tla) `INDEPENDENT_FIELDS`):
text e reactions sono campi indipendenti, ognuno aggiornato dal proprio
update type. L'ordine di arrivo dei due update non cambia lo stato
finale (commutatività).

## 5. Reactions su system message (sanity)

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine
    participant APP as App.Update
    participant CV as Conversation

    Note over SRV: edge case: Telegram normalmente NON<br/>permette reazioni su MessageService.<br/>Modelliamo la difesa per robustezza.

    SRV->>TG: UpdateMessageReactions{id:99, reactions:[...]}<br/>(id 99 è un MessageService nel viewport)
    TG->>APP: ReactionsUpdatedMsg{id:99, reactions}
    APP->>CV: locate m#99
    Note over CV: m.IsService = true
    CV->>CV: m.Reactions = reactions (data layer)
    CV->>CV: render bubble:<br/>  kind(m) = "system" → SystemBranch<br/>  CheckReactions: kind=system → skip
    Note over CV: NO reactions row rendered<br/>(rendering layer ignora<br/>reactions su system, vedi<br/>SYSTEM_NO_REACT in reactions.tla)
```

Lo stato dati può contenere reactions su un service message (per
robustezza), ma il render le ignora.

## 6. Chat chiusa — reactions update non visibile

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine
    participant APP as App.Update
    participant MODEL as App Model<br/>(non viewport)

    Note over APP: chat #100 chiusa,<br/>chat #50 attiva

    SRV->>TG: UpdateMessageReactions{chat:100, msg:42, [...]}
    TG->>APP: ReactionsUpdatedMsg{chat:100, msg:42, reactions}

    APP->>APP: chatID != activeChat
    Note over APP: SCELTA DI DESIGN:<br/>Step 25 NON mantiene per-chat message<br/>cache fuori dal viewport attivo.<br/>L'update viene SCARTATO.
    APP-->>MODEL: no-op

    Note over APP: Quando l'utente apre chat #100:<br/>loadHistoryCmd → fetch fresco<br/>→ reactions correnti dal server
```

**Razionale**: lo Step 25 mantiene il modello "viewport-scoped reactions
cache". Il server è autoritativo: una chat riaperta ricarica history,
con reazioni aggiornate. Niente complessità di cache cross-chat.

## Mapping tea.Cmd / tea.Msg

Aggiornamento alla tabella in
[`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md):

| Evento / azione | Cmd | Result Msg | Step di origine |
|------------------|-----|------------|-----------------|
| `UpdateMessageReactions` ricevuto | (nessuno: solo dispatch) | `ReactionsUpdatedMsg` | Step 25 |
| Ricezione messaggio service | (nessuno: convert layer) | `NewMessageMsg{IsService:true}` | Step 25 (esteso) |
| Caricamento history con service msg | (nessuno: convert layer) | `MessagesLoadedMsg` | Step 11 (esteso in Step 25) |

Nessuna nuova RPC verso Telegram in Step 25. Siamo solo consumer.

## Convert layer — dispatch

```
internal/telegram/convert/messages.go (esistente, esteso):
    convert(tg.MessageClass) → domain.Message
        case *tg.Message:        IsService=false, classify media (Step 24)
        case *tg.MessageService: IsService=true, ServiceText=formatAction(...)
        case *tg.MessageEmpty:   skip (don't produce domain msg)

internal/telegram/convert/reactions.go (NEW):
    convertReactions(tg.MessageReactions) → []domain.Reaction
        for each ReactionCount in Results:
            if Reaction is *ReactionEmoji:
                emit Reaction{Emoji: r.Emoticon, Count: rc.Count, Chosen: rc.Chosen}
            if Reaction is *ReactionCustomEmoji:
                skip (out of scope Step 25)
        sort by Count desc, Emoji asc

internal/telegram/convert/service.go (NEW):
    formatAction(tg.MessageActionClass, fromUserName) → string
        big switch over ~30 action variants (see reactions-and-system.md
        §System Message Classification). Returns "Service message" for
        unknown variants.
```

I tre file sono entry-point puri (nessuno stato, nessun side-effect),
testabili con unit-test deterministici.

## Cross-links

- Statechart + classification: [`../phase-2-behavioral/reactions-and-system.md`](../phase-2-behavioral/reactions-and-system.md)
- TLA+ formal spec: [`../phase-4-concurrency/reactions.tla`](../phase-4-concurrency/reactions.tla)
- Decisione storage + detection: [ADR-012](../phase-6-decisions/ADR-012-reactions-storage-and-system-detection.md)
- Pipeline: [`../development-pipeline.md` §Step 25](../development-pipeline.md)
- Domain types: [`../phase-5-data/domain-types.md`](../phase-5-data/domain-types.md) §Reaction §Message
- Entity mapping: [`../phase-5-data/entity-mapping.md`](../phase-5-data/entity-mapping.md) §Reactions §System Message
