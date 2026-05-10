# ADR-007: Gestione di Esc durante RPC in volo negli overlay

**Stato**: accettato
**Data**: 2026-04-24

## Contesto

Gli overlay modali introdotti nelle Fasi D–E (edit Step 19, delete confirm
Step 20, forward picker Step 21, in futuro search Step 26, command palette
Step 31) sguinzagliano RPC Telegram al momento della submit (`Enter` o `Y`).

Ci sono due stati distinti nel ciclo di vita dell'overlay:

1. **Input state** — l'utente sta ancora componendo/selezionando. Esc chiude
   l'overlay senza effetti.
2. **RPC in-flight state** — l'utente ha confermato; la goroutine
   `xxxMessageCmd` ha chiamato l'API e sta attendendo `ForwardResultMsg` /
   `EditResultMsg` / etc.

Domanda: cosa fa `Esc` nello stato (2)?

Questa decisione riguarda tutti gli overlay che innescano RPC; lo Step 21 ne
è il primo caso con un picker che può fallire e richiedere retry.

## Decisione

**Esc è ignorato durante `rpcInFlight`**. L'overlay rimane montato con
spinner visibile e input utente bloccato (no keystroke produce effetto) fino
a che arriva il messaggio di risultato (`ForwardResultMsg`, `EditResultMsg`,
etc.).

Motivazioni:

- **Stato consistente**: se chiudessimo l'overlay durante la RPC e questa poi
  avesse successo, l'effetto sarebbe "nascosto" (toast success per un'azione
  che sembra annullata). Se invece fallisse, non potremmo più mostrare il
  contesto (la chat selezionata, il testo editato) per il retry.
- **No cancelation al server**: gotd/td non espone un meccanismo pulito di
  cancellation per una `forwardMessages` già inviata; cancellare il context
  lato client non annulla la mutazione server-side.
- **Semplicità del modello**: il TLA+ (`forward_picker.tla`) verifica
  l'invariante `NO_ESC_DURING_RPC` e la liveness `NO_STUCK_RPC`. La finestra
  temporale di blocco è limitata dal timeout RPC (default gotd/td: 30s,
  tipico RTT: <500ms).

L'approccio è **uniforme** per tutti gli overlay che emettono RPC. Lo status
visivo della finestra "bloccata" è un **spinner + testo "Forwarding..."**
(edit: "Saving...", delete: "Deleting...").

In caso di RPC che non ritorna entro un timeout soft (5s), l'overlay mostra
un testo aggiuntivo "Taking longer than expected…" ma continua ad attendere
(no timeout hard). L'utente può sempre killare il processo con `Ctrl+Q` come
fallback.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **Esc ignorato durante RPC (scelta)** | Stato sempre consistente, modello TLA+ pulito | Finestra di ~500ms in cui l'utente non può annullare |
| Esc cancella il context e chiude overlay | Sembra reattivo | Effetto server non annullato; race su result msg |
| Esc marca l'overlay come "dismissed" ma aspetta result | Responsivo | Complessità: due stati di close, gestione toast silenzioso |
| Timeout hard a 3s → auto-close con errore | Utente mai bloccato | RPC potrebbe aver successo dopo il close → UI incoerente |
| Bottone "Cancel" visibile durante RPC | Discoverable | Identico al problema: no cancel server-side |

## Conseguenze

- **Positive**:
  - Invariante TLA+ `NO_ESC_DURING_RPC` banalmente soddisfatta.
  - Nessuna race condition tra `OverlayCloseMsg` e `ForwardResultMsg`.
  - UX coerente tra overlay (edit, delete, forward, e futuri).
  - Error path (`RPCFailure → Filtering`) mantiene il contesto per retry
    immediato.
- **Negative**:
  - Per RPC lente (flood wait) l'overlay può sembrare "freezato". Mitigato
    dal messaggio "Taking longer than expected…" dopo 5s.
  - Input keystrokes durante il blocco vengono scartati silenziosamente
    (nessun feedback). Mitigazione futura: flashare lo spinner.
- **Rischi**:
  - Se in futuro introduciamo RPC long-running (es. upload file), questa
    policy va rivista: per upload serve cancellazione vera (`Ctrl+C` → abort
    upload + rollback). Sarà oggetto di ADR separato.

## Scope

Questa ADR si applica a:

- Step 19 — Edit overlay (retroattivamente: il comportamento attuale già
  rispetta questa policy).
- Step 20 — Delete confirm (idem).
- **Step 21 — Forward picker** (prima applicazione esplicita).
- Step 22 — Multi-select forward/delete batch (eredita).
- Step 26 — Search overlay (la RPC di search ha semantica diversa: vedi ADR
  futuro).
- Step 31 — Command palette (idem).

## Cross-links

- [`phase-2-behavioral/forward-picker.md`](../phase-2-behavioral/forward-picker.md) §Invarianti
- [`phase-4-concurrency/forward_picker.tla`](../phase-4-concurrency/forward_picker.tla) — invariante formale
- Pipeline Step 21
