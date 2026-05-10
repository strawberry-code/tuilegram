# ADR-009: Confirm dialog del batch delete (singolo confirm N-aware)

**Stato**: accettato
**Data**: 2026-04-24

## Contesto

Lo Step 20 ha introdotto il **confirm dialog** modale per il delete di un
singolo messaggio: overlay centrato con testo *"Delete this message?
[Y] [N]"*, dove `Y` invia la `messages.deleteMessages` e `N` chiude
l'overlay senza azione.

Lo Step 22 introduce il delete batch su `N` messaggi selezionati. La
domanda è:

> Come confermiamo l'azione distruttiva su N messaggi?

Le opzioni sono:

1. **Un confirm per messaggio** (`N` overlay sequenziali).
2. **Un singolo confirm "Delete N messages?"** con messaggio N-aware.
3. **Confirm + lista anteprima** (overlay più grande con elenco messaggi).
4. **No confirm in batch** (delete immediato, basandosi sulla scelta
   esplicita di multi-selezione).

In parallelo c'è la questione di `revoke` (delete per tutti vs delete locale):
nel single-msg lo Step 20 default è `revoke=true` per propri messaggi e
`revoke=false` per altrui. Per il batch, la scelta dev'essere uniforme.

## Decisione

**Singolo confirm dialog N-aware, single-RPC, revoke uniforme**:

- **Un solo overlay confirm**, identico al pattern Step 20, con testo
  parametrico:
  - `N == 1` → *"Delete this message? [Y] [N]"*
  - `N >  1` → *"Delete N messages? [Y] [N]"*
- **Una sola RPC** `api.MessagesDeleteMessages` con `id: []int`.
  MTProto garantisce atomicità sul subset di ID validi; gli ID già
  cancellati (race con `UpdateDeleteMessages`) vengono ignorati lato
  server.
- **Revoke uniforme**: `revoke = true` per il batch se **tutti** i msg
  selezionati sono propri (mittente = self), altrimenti `revoke = false`
  con warning visibile nell'overlay: *"⚠ Some messages can only be
  deleted locally"*. Questo riflette il behavior dei client ufficiali.
- **Cancel preserva S**: `N` o `Esc` chiudono l'overlay ma non puliscono
  `S`. L'utente può aggiornare la selezione e riprovare, oppure premere
  `Esc` di nuovo per uscire da MultiSelect.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **Singolo confirm N-aware (scelta)** | UX semplice; un solo gesto Y; consistente con Step 20 | Nessuna granularità per-msg; se utente cambia idea su 1 msg tra N deve cancellare e ri-selezionare |
| `N` confirm sequenziali | Granularità totale | UX inaccettabile per N>3; click fatigue; bypass facile |
| Confirm con anteprima dei msg | Trasparenza piena | Overlay grande e complesso; reading load alto; out of scope Step 22 |
| No confirm | Minimal friction | Azione distruttiva irreversibile senza guardia → non accettabile |
| Confirm con select per-msg "include this?" | Granularità on-the-fly | UX troppo complessa; replica del MultiSelect dentro l'overlay |
| Two-step confirm (Y twice) per N>10 | Protezione extra | Inconsistente con single-msg; arbitrario |

## Conseguenze

- **Positive**:
  - Riuso totale del confirm overlay Step 20: solo testo parametrico.
  - UX coerente: stesso gesto (`D` → `Y` → done) per single e batch.
  - Una sola RPC riduce flood-wait risk e tempo di completamento.
  - `S` preservato su cancel: l'utente può iterare sulla selezione senza
    perdere lo stato.
  - Invariante `SOURCE_SNAPSHOT` (vedi
    [`multi_select.tla`](../phase-4-concurrency/multi_select.tla))
    garantisce che il payload del delete sia frozen al submit, anche se
    `S` muta concorrentemente per `UpdateDeleteMessages` remoti.
- **Negative**:
  - L'utente che cambia idea su 1 msg dei N selezionati deve premere `N`,
    aggiornare `S`, e ripremere `D`. Mitigazione: l'info bar
    *"N selected"* è sempre visibile, riducendo l'errore.
  - Il messaggio di warning *"⚠ Some messages can only be deleted
    locally"* può confondere utenti non tecnici. Mitigazione: testo
    chiaro, link a help (`?` keybinding).
- **Rischi**:
  - Per `N` molto grande (es. >20) Telegram potrebbe rispondere
    `MESSAGE_DELETE_FORBIDDEN` per alcuni messaggi (limite admin in
    gruppi). La RPC fallisce parzialmente. Mitigazione: status bar mostra
    *"Deleted X of N messages: <reason>"*. La gestione dell'errore
    parziale è prevista come tech-debt da rifinire allo Step futuro
    "polish" (Step 33) o in un ADR successivo.

## Scope

- Step 22 — Multi-select + batch delete (prima applicazione).
- Eredità Step 20: il confirm overlay invariato a parte testo dinamico.

## Cross-links

- Statechart batch: [`phase-2-behavioral/multi-select.md`](../phase-2-behavioral/multi-select.md)
- Sequence diagram: [`phase-3-interactions/multi-select-flow.md`](../phase-3-interactions/multi-select-flow.md)
- TLA+: [`phase-4-concurrency/multi_select.tla`](../phase-4-concurrency/multi_select.tla)
- ADR predecessore: [ADR-007](ADR-007-overlay-in-flight-rpc.md) (Esc bloccato durante RPC, eredita)
- Step 20 confirm pattern (riusato)
- Pipeline: Step 22
