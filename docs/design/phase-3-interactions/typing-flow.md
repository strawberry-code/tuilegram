# Typing Flow — Sequence Diagrams (Step 23)

Flusso runtime del **typing indicator**. Complementare allo statechart in
[`../phase-2-behavioral/typing-indicator.md`](../phase-2-behavioral/typing-indicator.md).

## 1. Happy path — peer inizia a scrivere e si ferma

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine<br/>(gotd/td dispatcher)
    participant APP as App.Update<br/>(bubbletea main loop)
    participant CL as ChatList
    participant HDR as Conv Header
    participant TICK as tea.Tick (5s)

    Note over SRV: peer "John Doe" starts<br/>typing in private chat #42

    SRV->>TG: UpdateUserTyping{peer=42, action=typing}
    TG->>TG: parse → domain event
    TG->>APP: program.Send(UpdateUserTypingMsg{peer:42})

    APP->>APP: typing[42] = {lastTypingAt: t0}
    APP->>TICK: schedule TypingTimeoutMsg{peer:42, t:t0}<br/>fires at t0+5s
    APP->>CL: re-render row #42 → "typing..."
    APP->>HDR: re-render header → "John Doe — typing..."

    Note over SRV,APP: peer keeps typing,<br/>server emits another update at t1 (t1 < t0+5s)
    SRV->>TG: UpdateUserTyping{peer=42}
    TG->>APP: program.Send(UpdateUserTypingMsg{peer:42})
    APP->>APP: typing[42].lastTypingAt = t1
    Note over APP: NO new tea.Tick scheduled<br/>(see ADR-010)

    Note over TICK: at t0+5s, tick fires
    TICK->>APP: TypingTimeoutMsg{peer:42, t:t0}
    APP->>APP: now - typing[42].lastTypingAt = (t0+5s)-t1 < 5s
    Note over APP: STALE TICK — no-op<br/>(typing extended past t0+5s)

    Note over SRV: peer stops typing.<br/>No more updates after t1.

    Note over TICK: at... wait — no tick was<br/>scheduled for t1+5s yet
```

**Punto critico**: quando si rinfresca `lastTypingAt` non si schedula un nuovo
tick. Come si esce allora da `Typing.Active`? Vedi sezione "TTL refresh
strategy" più sotto e [ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md).

## 2. TTL refresh strategy — variante con re-arm

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine
    participant APP as App.Update
    participant TICK as tea.Tick

    SRV->>TG: UpdateUserTyping{peer=42}
    TG->>APP: UpdateUserTypingMsg{peer:42}
    APP->>APP: typing[42].lastTypingAt = t0
    APP->>TICK: schedule TypingTimeoutMsg{42, t:t0}

    Note over TICK: tick @ t0+5s

    SRV->>TG: UpdateUserTyping{peer=42}
    TG->>APP: UpdateUserTypingMsg{peer:42} at t1
    APP->>APP: typing[42].lastTypingAt = t1
    APP->>TICK: schedule TypingTimeoutMsg{42, t:t1}<br/>(NEW tick @ t1+5s)

    Note over TICK: now there are 2 ticks<br/>pending for peer 42

    TICK->>APP: TypingTimeoutMsg{42, t:t0}<br/>(first tick fires)
    APP->>APP: now - lastTypingAt = (t0+5s)-t1 < 5s
    Note over APP: STALE TICK — no-op

    TICK->>APP: TypingTimeoutMsg{42, t:t1}<br/>(second tick fires)
    APP->>APP: now - lastTypingAt = (t1+5s)-t1 = 5s
    APP->>APP: typing[42] deleted → state Idle
    APP->>CL: re-render row → name + last msg
    APP->>HDR: re-render header → "John Doe · last seen ..."
```

Variante "re-arm" (raccomandata in [ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md)):
ogni `UpdateUserTypingMsg` schedula **un nuovo tick** (non sovrascrive il
precedente — non c'è cancellation). I tick precedenti, quando scadranno,
saranno benigni grazie al check su `lastTypingAt` (vedi `STALE_TICK_BENIGN`
in `typing.tla`).

Numero di tick pendenti per peer ≤ N (numero di update ricevuti negli
ultimi 5s). In pratica Telegram emette al massimo ~1 update / 5s per
azione utente, quindi il numero atteso è 1-2 tick pendenti per peer.

## 3. Multi-chat typing concorrente

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine
    participant APP as App.Update
    participant CL as ChatList

    SRV->>TG: UpdateUserTyping{peer=42}
    TG->>APP: UpdateUserTypingMsg{42}
    APP->>APP: typing[42] = {t0}
    APP->>CL: row #42 → "typing..."

    SRV->>TG: UpdateUserTyping{peer=99}
    TG->>APP: UpdateUserTypingMsg{99}
    APP->>APP: typing[99] = {t0+1ms}
    APP->>CL: row #99 → "typing..."

    Note over CL: due righe simultaneamente "typing..."<br/>(due peer diversi, indipendenti)
```

Ogni peer ha la sua entry in `typing[]`. Non c'è interazione tra peer.

## 4. Chat aperta vs chiusa — render scope

```mermaid
sequenceDiagram
    participant APP as App.Update
    participant CL as ChatList
    participant HDR as Conv Header
    participant CV as Conversation viewport

    Note over APP: typing[42] = {t0}<br/>chat attiva = 99 (NON 42)
    APP->>CL: row #42 → "typing..."
    Note over HDR: header mostra peer 99, non 42<br/>→ NESSUN cambiamento header

    Note over APP: typing[42] = {t0}<br/>chat attiva = 42
    APP->>CL: row #42 → "typing..."
    APP->>HDR: header → "John Doe — typing..."
    Note over CV: viewport non cambia<br/>(typing è metadata, non un msg)
```

Il viewport (lista messaggi) **non** è coinvolto: il typing non si aggiunge
come bubble né come riga di sistema. È solo nell'header.

## 5. Out-of-scope: gruppi (UpdateChatUserTyping)

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine
    participant APP as App.Update

    SRV->>TG: UpdateChatUserTyping{chat=100, user=7}
    Note over TG: Step 23: questo update è<br/>IGNORATO (no handler registrato).<br/>Sarà gestito in step futuro.
    TG-->>APP: (nessun messaggio inviato)
```

Lo Step 23 registra **solo** `OnTyping` per dialogs 1:1 (UpdateUserTyping).
`UpdateChatUserTyping` (typing in gruppi/canali) è gestito in step
successivi e richiede un modello UI diverso ("Alice and Bob are typing...").

## 6. Race con NewMessageMsg

```mermaid
sequenceDiagram
    participant SRV as Telegram Server
    participant TG as Telegram Goroutine
    participant APP as App.Update
    participant CL as ChatList
    participant CV as Conversation

    Note over APP: typing[42] = {t0}<br/>row #42 = "typing..."

    SRV->>TG: UpdateNewMessage{peer=42, msg="hello"}
    TG->>APP: NewMessageMsg{42, msg}

    APP->>CV: append "hello" to viewport
    APP->>APP: typing[42] = ?
    Note over APP: SCELTA DI DESIGN:<br/>su NewMessage, clear typing[peer] eagerly?<br/>oppure lascia decadere via TTL?

    APP->>CL: row #42 → preview "hello"<br/>(typing render is overwritten)
```

**Scelta**: lasciamo decadere via TTL (più semplice, no edge case extra).
La row si aggiorna comunque alla preview "hello" perché la chat list row
prioritizza il LastMessage rispetto al typing — ma l'header conversazione
può continuare a mostrare "typing..." per qualche secondo se Telegram
emette `UpdateUserTyping` mentre la persona finisce di scrivere/inviare.

Comportamento accettabile: rispecchia esattamente quello dell'app
ufficiale Telegram.

## Mapping tea.Cmd

Aggiornamento alla tabella "Mapping tea.Cmd" in
[`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md):

| Azione / evento | Cmd | Result Msg |
|------------------|-----|------------|
| `UpdateUserTypingMsg` ricevuto | `scheduleTypingTimeoutCmd` (= `tea.Tick(5s, ...)`) | `TypingTimeoutMsg` |

Nessuna nuova RPC verso Telegram in Step 23 (siamo solo consumer).

## Cross-links

- Statechart: [`../phase-2-behavioral/typing-indicator.md`](../phase-2-behavioral/typing-indicator.md)
- Concurrency invariants: [`../phase-4-concurrency/typing.tla`](../phase-4-concurrency/typing.tla)
- Pipeline: [`../development-pipeline.md` §Step 23](../development-pipeline.md)
- Decisione TTL: [ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md)
