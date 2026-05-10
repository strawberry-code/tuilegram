# ADR-008: Semantica del batch forward (single-target picker reuse)

**Stato**: accettato
**Data**: 2026-04-24

## Contesto

Lo Step 22 introduce la **multi-selezione** di messaggi nella conversazione
e l'azione batch `f` (forward) sui messaggi selezionati. Lo Step 21 ha già
introdotto un forward picker overlay (vedi
[`phase-2-behavioral/forward-picker.md`](../phase-2-behavioral/forward-picker.md))
che gestisce il forward di **un singolo messaggio verso una singola chat**.

La domanda è:

> Come gestiamo il forward batch dei `N` messaggi selezionati?

Tre dimensioni di scelta sono ortogonali:

1. **Numero di destinazioni**: una sola chat target (single-target) vs
   multiple chat target (multi-target con check-list).
2. **Granularità RPC**: una sola `messages.forwardMessages` con `id: []int`
   vs `N` chiamate separate (una per messaggio).
3. **Riuso UI**: stesso picker dello Step 21 oppure nuovo overlay
   dedicato al batch.

## Decisione

**Single-target, single-RPC, riuso del picker Step 21**:

- **Single-target**: il batch forward inoltra tutti gli `N` messaggi
  selezionati verso **una sola** chat di destinazione. Il picker mostra il
  conteggio nell'header (es. *"Forward 3 messages to…"*).
- **Single-RPC**: una sola chiamata `api.MessagesForwardMessages` con
  `id: []int = [m1, m2, …, mN]`. Il metodo MTProto accetta nativamente
  array di ID, e Telegram garantisce ordering e atomicità lato server.
- **Picker reuse**: stesso componente dello Step 21. Differenza UX minima:
  l'header dell'overlay riflette `len(source)` se > 1.

Questa decisione è coerente con gli altri client Telegram ufficiali (mobile
e desktop), dove il forward multi-messaggio porta a un picker chat unico.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **Single-target + single-RPC + picker reuse (scelta)** | Riuso totale UI Step 21; UX coerente con client ufficiali; una sola RPC riduce flood-wait risk | Per inoltrare a 2 chat l'utente deve ripetere `f` (S preservato dopo retry sì, ma non dopo success: ADR-008 §Consequences) |
| Multi-target picker (check-list di chat) | Un solo gesto per fan-out | UX divergente dal single-msg flow; doppia complessità nel picker; richiede overlay nuovo |
| `N` RPC separate (una per messaggio) | Granularità di errore per-message | `N` chiamate flood-wait risk; ordering non garantito; inutile complessità |
| Nuovo overlay "batch forward" dedicato | Header e UX dedicati | Duplica codice; due pattern overlay da mantenere; viola DRY |
| Pipeline verso Step futuri (multi-target in step 28) | Step 22 minimal | Funzionalità incompleta a Step 22, scope creep |

## Conseguenze

- **Positive**:
  - Riuso totale del picker Step 21: zero nuovo codice UI per l'overlay.
  - Una sola RPC `messages.forwardMessages` per N msg → flood-wait risk
    invariante rispetto a Step 21.
  - Invariante `SOURCE_SNAPSHOT` di
    [`forward_picker.tla`](../phase-4-concurrency/forward_picker.tla)
    rimane valida: la lista `id` è snapshot al submit.
  - Il modello TLA+ Step 22
    ([`multi_select.tla`](../phase-4-concurrency/multi_select.tla)) eredita
    `BATCH_ATOMICITY` e `NO_ESC_DURING_RPC` da ADR-007.
  - UX consistente con client ufficiali Telegram (riconoscibilità).
- **Negative**:
  - L'utente che vuole inoltrare a 2 chat deve fare 2 round (selezione →
    `f` → chat A → success → riselezionare → `f` → chat B). Mitigazione:
    dopo success, `S` è cleared (`BatchActionDoneMsg`) ma il cursore resta
    sui messaggi originali → `Space` per ri-selezionare è economico.
  - Header del picker varia da "Forward to…" (N=1) a "Forward N
    messages to…" (N>1): minimo costo di rendering condizionale.
- **Rischi**:
  - Se Telegram in futuro introducesse limiti `id: []int ≤ 100` (oggi
    documentati a 100), per `|S| > 100` dovremmo paginare la RPC. Per ora
    la UX scoraggia tale uso (la lista checkbox diventa ingestibile prima).
    Sarà oggetto di ADR successivo se necessario.

## Scope

- Step 22 — Multi-select + batch forward (prima applicazione).
- Eredità Step 21: il picker invariato a parte l'header dinamico.

## Cross-links

- Statechart batch: [`phase-2-behavioral/multi-select.md`](../phase-2-behavioral/multi-select.md)
- Statechart picker (riuso): [`phase-2-behavioral/forward-picker.md`](../phase-2-behavioral/forward-picker.md)
- Sequence diagram: [`phase-3-interactions/multi-select-flow.md`](../phase-3-interactions/multi-select-flow.md)
- TLA+: [`phase-4-concurrency/multi_select.tla`](../phase-4-concurrency/multi_select.tla)
- ADR predecessore: [ADR-006](ADR-006-forward-fuzzy-algorithm.md), [ADR-007](ADR-007-overlay-in-flight-rpc.md)
- Pipeline: Step 22
