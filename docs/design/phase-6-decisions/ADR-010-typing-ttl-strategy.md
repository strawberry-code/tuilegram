# ADR-010: Strategia TTL per il typing indicator

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 23 introduce il typing indicator: alla ricezione di un
`UpdateUserTyping` (MTProto update emesso dal server quando un peer sta
scrivendo) la UI mostra "typing..." per **5 secondi** dall'ultimo evento.
Trascorsi 5s senza nuovi update, lo stato torna al rendering normale
(nome chat + ultimo messaggio).

Telegram **non** emette un evento esplicito di "stop typing": il client
deve dedurre la fine via TTL. La specifica MTProto raccomanda un timeout
di 5-6 secondi, in linea con quanto fanno il client ufficiale e tdesktop.

In bubbletea, l'unico meccanismo nativo per "fare qualcosa più tardi" è
`tea.Tick(d, fn)` che produce un `tea.Cmd`. Il messaggio risultante
arriva al `Update()` esattamente come un evento normale. **Non c'è
cancellation**: una volta schedulato, il tick scade.

Si pongono due decisioni progettuali interconnesse:

1. **Modello dati**: come rappresentare lo stato typing per-peer?
   - **Opzione A**: `map[ChatID]time.Time` (= `lastTypingAt`). Lo stato
     "typing" è derivato runtime da `now - lastTypingAt < 5s`.
   - **Opzione B**: `map[ChatID]TimerID` con tracking esplicito del
     timer attivo. Cancellation simulata invalidando l'ID.

2. **Re-arm strategy**: cosa fare quando arriva un nuovo
   `UpdateUserTyping` mentre un tick precedente è ancora pendente?
   - **Strategia 1 (re-arm)**: schedula un **nuovo** `tea.Tick(5s)` ad
     ogni update. I tick precedenti scadranno e saranno benigni (no-op
     se lo stato non si è "raffreddato").
   - **Strategia 2 (single-tick)**: schedula il tick solo al primo
     update; aggiornamenti successivi rinfrescano `lastTypingAt` ma non
     creano nuovi tick. Problema: se l'utente continua a scrivere oltre
     i 5s del primo tick, il primo tick scade e clear-a uno stato che
     dovrebbe rimanere attivo.

Questi due assi sono in parte ortogonali ma si influenzano:

- (A, 1): timestamp-based + re-arm. Stato derivato da timestamp;
  multipli tick pendenti sono OK perché sono check-and-clear idempotenti.
- (A, 2): timestamp-based + single-tick. Richiederebbe ri-schedulare al
  fire-time se ancora "fresco" (loop di tick, complica l'analisi).
- (B, 1): timer-id + re-arm. Equivalente ad (A, 1) ma con stato in più.
- (B, 2): timer-id + cancellazione "logica". Bookkeeping aggiuntivo;
  l'unico modo per cancellare un `tea.Tick` è ignorarlo al fire-time.

## Decisione

**Adottiamo (A, 1): modello timestamp-based + strategia re-arm.**

In dettaglio:

- Lo stato per-peer è `typing : map[ChatID]TypingState` con
  `TypingState ::= { lastTypingAt time.Time; userID int64 }`.
- L'assenza dalla mappa equivale a stato `Idle`.
- Alla ricezione di `UpdateUserTypingMsg{peer}`:
  - Aggiorna `typing[peer].lastTypingAt = now`.
  - Schedula sempre `tea.Tick(5*time.Second, ...)` che produrrà
    `TypingTimeoutMsg{peer, scheduledAt: now}`.
- Alla ricezione di `TypingTimeoutMsg{peer, scheduledAt}`:
  - Se `peer` non è in `typing` → no-op (idempotenza).
  - Se `now - typing[peer].lastTypingAt >= 5s` → `delete(typing, peer)`,
    re-render.
  - Altrimenti → no-op (stale tick: un update successivo ha esteso il
    TTL).

Vantaggi:

- **Modello formale semplice**: il TLA+ (`typing.tla`) verifica
  l'invariante `TYPING_TTL_BOUND` in pochi stati. La benignità degli
  stale tick è codificata direttamente nell'azione `TickFire`.
- **No race**: l'ordine `Update → Tick` arriva sempre nel main loop
  bubbletea (single-threaded), quindi non c'è race fisica. Il check
  `now - lastTypingAt >= TTL` esegue la decisione corretta atomicamente.
- **Complessità di stato bounded**: il numero di tick pendenti per peer
  è limitato dal rate degli update (Telegram emette ~1 update/5s →
  attesi 1-2 tick pendenti per peer attivo).
- **Memory leak escluso**: i tick scadono comunque dopo 5s e si
  rimuovono dal pool runtime di bubbletea. La mappa `typing` viene
  pulita al primo TickFire valido.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **(A, 1) timestamp + re-arm** [scelta] | Modello semplice, no race, TLA+ pulito, idempotenza naturale | Più tick pendenti contemporaneamente per peer (≤ N updates in 5s); spreco minimale |
| (A, 2) timestamp + single-tick | Solo 1 tick per peer | Loop "self-rescheduling" se tick fire mentre fresco → action ricorsiva, più difficile da modellare |
| (B, 1) timer-id + re-arm | Tracciamento esplicito | Stato extra senza beneficio (la cancellation logica c'è già nel check timestamp) |
| (B, 2) timer-id + ignore | Cancellation "esplicita" | Stato extra + bookkeeping; equivalente a (A, 1) per la UI |
| Polling globale (1 tick periodico ogni 1s) | 1 solo tick attivo, semplice in apparenza | Render thrashing (re-render ogni 1s anche senza cambi); costo CPU costante; difficile sincronizzare con TTL preciso |
| Goroutine dedicata + channel | Cancellation reale possibile | Anti-pattern bubbletea: stato fuori dal `Model`, sync issues, viola `feedback_design_approach` |

## Conseguenze

- **Positive**:
  - Invariante `TYPING_TTL_BOUND` (typing⟹freschezza<5s) banalmente
    verificata dal TLA+.
  - Stale-tick safety codificata strutturalmente, non come patch.
  - Implementazione Go minimale: `tea.Tick` + `time.Now()` + `map`. No
    dipendenze nuove, no goroutine custom.
  - Pattern riusabile: lo stesso schema potrà servire per altri
    indicatori a TTL (es. "online" effimero, draft saving toast).
- **Negative**:
  - Tick "spreco" per peer attivi: se Telegram emette 5 update/5s, ci
    saranno fino a 5 tick pendenti per quel peer. Costo: ~80 byte per
    tick. Trascurabile per il numero di peer realistico (<100 contatti
    attivi simultaneamente).
  - Lo stale check implica un confronto `time.Now()` ad ogni tick:
    cost O(1), non rilevante.
- **Rischi**:
  - **Drift dell'orologio**: se il sistema sospende (laptop closed) il
    `time.Now()` può saltare avanti. Allo wake, tutti i tick pendenti
    scattano "in ritardo" ma il check `>= 5s` farà comunque clear
    correttamente. Nessun rischio di stato bloccato.
  - **Re-arm con orologio non monotono**: usiamo `time.Now()` (non
    monotonic). Se l'utente cambia manualmente l'ora avanti durante un
    typing attivo, lo stato passa a Idle prematuramente. Mitigazione:
    in futuro usare `time.Since` su un valore monotonic captured
    all'avvio. Fuori scope Step 23 (effetto solo cosmetico, no
    correttezza).

## Scope

Questa ADR si applica a:

- **Step 23 — Typing indicator** (prima applicazione).
- Step futuri che useranno lo stesso pattern timestamp+TTL: rendering
  "online" effimero, status draft saving, debounce search overlay
  (Step 26), which-key timeout (Step 28).

Per lo Step 26 (search debounce) il TTL è molto più breve (300ms) ma il
pattern resta identico: timestamp di ultimo input, `tea.Tick`, check
allo scadere.

## Cross-links

- [`phase-2-behavioral/typing-indicator.md`](../phase-2-behavioral/typing-indicator.md) §Invarianti
- [`phase-3-interactions/typing-flow.md`](../phase-3-interactions/typing-flow.md) §TTL refresh strategy
- [`phase-4-concurrency/typing.tla`](../phase-4-concurrency/typing.tla) — invariante `TYPING_TTL_BOUND`, `STALE_TICK_BENIGN`
- Pipeline Step 23
