# Forward Flow — Sequence Diagrams (Step 21)

Flusso runtime del **forward** di un singolo messaggio. Complementare allo
statechart in [`../phase-2-behavioral/forward-picker.md`](../phase-2-behavioral/forward-picker.md).

Scenario 9 in [`scenarios.md`](scenarios.md) era già un abbozzo; questo
documento lo estende con il dettaglio del fuzzy filter, del concurrency model
e dei path di errore introdotti dallo Step 21.

## Happy path — single message forward

```mermaid
sequenceDiagram
    participant U as User
    participant CV as Conversation
    participant APP as App.Update
    participant OVL as ForwardPicker
    participant CMD as forwardMessageCmd (goroutine)
    participant TG as Telegram Client
    participant SRV as Telegram Server
    participant SB as Status Bar

    U->>CV: cursor on msg, press 'f'
    CV->>APP: ForwardRequestMsg{[]Message{m}}
    APP->>OVL: mount picker, state=Opening
    APP->>APP: snapshot dialogs from local cache
    APP->>OVL: ForwardPickerReadyMsg{[]Chat}
    Note over OVL: state = Filtering.Idle
    OVL-->>U: render overlay (full chat list)

    U->>OVL: types "te"
    OVL->>OVL: ForwardFilterMsg{"te"} (in-memory re-rank)
    OVL-->>U: list narrows ("Team Dev", "Tester Bot")

    U->>OVL: types "am"
    OVL->>OVL: ForwardFilterMsg{"team"} (re-rank)
    OVL-->>U: list = ["Team Dev"]

    U->>OVL: Enter
    OVL->>APP: ForwardSubmitMsg{targetChatID=42, []Message{m}}
    APP->>CMD: spawn forwardMessageCmd
    Note over OVL: state = Forwarding.RPCInFlight<br/>spinner shown, input blocked

    CMD->>TG: api.MessagesForwardMessages(peer=42, fromPeer=m.ChatID, id=[m.ID], randomID=[rand])
    TG->>SRV: messages.forwardMessages
    SRV-->>TG: Updates (UpdateNewMessage su chat 42)
    TG-->>CMD: return nil error
    CMD-->>APP: ForwardResultMsg{targetChatID=42, err=nil}

    APP->>OVL: unmount (state=Closed)
    APP->>SB: "Forwarded to Team Dev"
    APP-->>U: re-render (overlay gone)

    Note over TG,SRV: separately, the dispatcher delivers<br/>UpdateNewMessage for chat 42 to APP<br/>(normal ReceiveMessage flow, out of scope here)
```

## Error path — RPC fallisce

```mermaid
sequenceDiagram
    participant U as User
    participant OVL as ForwardPicker
    participant APP as App.Update
    participant CMD as forwardMessageCmd
    participant TG as Telegram Client
    participant SRV as Telegram Server
    participant SB as Status Bar

    U->>OVL: Enter on "Team Dev"
    OVL->>APP: ForwardSubmitMsg{42, [m]}
    APP->>CMD: spawn
    Note over OVL: state = Forwarding.RPCInFlight

    CMD->>TG: api.MessagesForwardMessages(...)
    TG->>SRV: messages.forwardMessages
    SRV-->>TG: RPC_ERROR (e.g. CHAT_WRITE_FORBIDDEN / FLOOD_WAIT)
    TG-->>CMD: error
    CMD-->>APP: ForwardResultMsg{42, err}

    APP->>OVL: state = Filtering.Idle (picker stays open)
    APP->>SB: "Forward failed: <reason>"
    OVL-->>U: toast visible, list re-shown (user may pick another chat)
```

## Cancel path — Esc prima di submit

```mermaid
sequenceDiagram
    participant U as User
    participant OVL as ForwardPicker
    participant APP as App.Update

    U->>OVL: Esc
    Note over OVL: state in {Filtering.Idle, Filtering.Typing}
    OVL->>APP: OverlayCloseMsg
    APP->>OVL: unmount (state=Closed)
    APP-->>U: re-render (no RPC emitted)
```

## Cancel durante RPC — bloccato (ADR-007)

```mermaid
sequenceDiagram
    participant U as User
    participant OVL as ForwardPicker
    participant APP as App.Update

    Note over OVL: state = Forwarding.RPCInFlight
    U->>OVL: Esc
    OVL->>OVL: input ignored (see ADR-007)
    Note over OVL: picker attende ForwardResultMsg prima di<br/>accettare nuovi input da utente
```

## Fuzzy filter — detail

```mermaid
sequenceDiagram
    participant U as User
    participant TI as textinput (bubble)
    participant OVL as ForwardPicker
    participant IDX as in-memory index ([]Chat)

    U->>TI: keystroke 'a'
    TI->>OVL: updated query "tea"
    OVL->>IDX: rank(query, chats)
    Note over IDX: for each chat: score = f(title, username, query)<br/>sort by score desc, drop zero-score
    IDX-->>OVL: []ChatRanked
    OVL->>OVL: reset cursor=0
    OVL-->>U: render filtered list
```

Ranking è **sincrono e in-memory** (nessuna RPC). La lista sorgente è uno
snapshot dei dialogs già in cache al momento dell'apertura; eventuali update
concorrenti (nuovi dialogs, rename chat) non impattano l'overlay attivo — vedi
invariante "Source = snapshot" in `forward-picker.md`.

## Mapping tea.Cmd

Aggiornamento alla tabella "Mapping tea.Cmd" in
[`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md):

| Azione utente | Cmd | API gotd/td | Result Msg |
|---------------|-----|-------------|------------|
| `f` → select chat → Enter | `forwardMessageCmd` | `api.MessagesForwardMessages` | `ForwardResultMsg` |

## Cross-links

- Statechart: [`../phase-2-behavioral/forward-picker.md`](../phase-2-behavioral/forward-picker.md)
- Concurrency invariants: [`../phase-4-concurrency/README.md`](../phase-4-concurrency/README.md) §Forward RPC
- Pipeline: [`../development-pipeline.md` §Step 21](../development-pipeline.md)
- Decisioni: [ADR-006](../phase-6-decisions/ADR-006-forward-fuzzy-algorithm.md),
  [ADR-007](../phase-6-decisions/ADR-007-overlay-in-flight-rpc.md)
