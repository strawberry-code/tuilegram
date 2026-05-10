# ADR-013: Search overlay — debounce + stale-result drop + Esc-during-RPC

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 26 introduce l'overlay di ricerca globale: `/` apre un overlay
modale; ogni keystroke triggera (potenzialmente) una RPC
`api.MessagesSearchGlobal` su Telegram; i risultati popolano una lista
navigabile; `Enter` su un hit chiude l'overlay e fa jump-to-message.

Tre problemi di concorrenza/UX vanno decisi insieme perché si influenzano:

1. **Quando inviare la RPC?** Inviare ad ogni keystroke produrrebbe
   N RPC per "hello" — flood wait probabile. Un debounce è necessario.

2. **Come gestire risultati di RPC più vecchie del query attuale?**
   Senza cancellation primitive in gotd/td (e la search è solo
   read-only, non c'è effetto da rollback) i risultati di RPC più
   vecchie possono arrivare DOPO che l'utente ha già digitato
   ulteriormente. Mostrarli sovrascriverebbe risultati più nuovi → race
   visibile.

3. **Che fare di `Esc` durante una RPC in volo?** ADR-007 ha stabilito
   che gli overlay di mutazione (forward, edit, delete) IGNORANO `Esc`
   durante RPC in volo per evitare effetti server "nascosti". Vale lo
   stesso per la search?

Per (1), il bench-mark è chiaro:
- Telegram raccomanda 200-400ms di debounce per ricerca interattiva.
- Tdesktop usa 350ms; web.telegram.org ~250ms.
- 300ms è un compromesso utilizzato in molti TUI editor (telescope.nvim,
  fzf live preview).

Per (2), non esiste un meccanismo di cancellation pulito in gotd/td (la
RPC parte come `tg.MessagesSearchGlobalRequest` e completa via context
quando il server risponde; cancellare il context lato client non
informa il server, e la latency tipica `<500ms` rende inutile la
gymnastica). Servono due strumenti distinti:
- un **token di freschezza** (counter o timestamp) inviato con la RPC
  e verificato al return time;
- un meccanismo per **scartare i tick di debounce stale** (perché
  `tea.Tick` non ha cancellation, gli stessi tick precedenti scadono
  comunque).

Per (3), notiamo che la search differisce strutturalmente dagli
overlay coperti da ADR-007:
- è **read-only**: nessun effetto server-side da nascondere o rollback.
- l'utente che preme `Esc` **vuole** chiudere; bloccarlo per attendere
  un risultato che probabilmente non lo interessa più è UX-hostile.
- il pattern `latestQueryID` già rende benigni i risultati stale.
  Bumping `latestQueryID` su `Close` rende benigni anche i risultati
  che dovessero arrivare dopo la chiusura.

## Decisione

**Triplice decisione consolidata in una sola ADR perché interconnessa.**

### D1 — Debounce: 1s via `tea.Tick`

Ogni `SearchQueryChangedMsg` (cambiamento di query nell'overlay):

1. Bumpa `latestQueryID := latestQueryID + 1`.
2. Schedula `tea.Tick(1s, fn)` che produce
   `SearchDebounceFiredMsg{queryID: <appena bumpato>}`.

**Nota revisione**: la value originale era 300ms (allineata a tdesktop /
web.telegram). User feedback Step 26 ha richiesto pausa più lunga prima
del fire RPC: 1s lascia tempo all'utente di completare la query, riduce
RPC stale in volo, abbassa rischio flood-wait. UX accettata come "blur to
commit" piuttosto che "live preview".

Quando un `SearchDebounceFiredMsg{qID}` arriva al main loop:
- Se `qID != latestQueryID` → **no-op** (stale debounce; un'altra
  keystroke ha bumpato `latestQueryID` nel frattempo).
- Se `query == ""` → no-op + `state.Hits := []`.
- Altrimenti spawn `searchCmd(query, qID)` (goroutine RPC).

Tick precedenti **non** vengono cancellati (no primitive in `tea.Tick`).
Sono benigni allo scadere grazie al check `qID == latestQueryID`.

Il pattern è isomorfo a `typing.tla` ([ADR-010](ADR-010-typing-ttl-strategy.md)),
ma con `latestQueryID` (counter) al posto di `lastTypingAt` (timestamp).
La differenza è semantica: per typing ci interessa *il tempo* trascorso;
per search ci interessa *l'identità* della query corrente.

### D2 — Stale-result drop: token `latestQueryID` con i risultati RPC

`searchCmd(query, qID)` cattura `qID` nella closure. Al return:

- Goroutine: `return SearchResultMsg{QueryID: qID, Hits: ..., Err: ...}`.
- Main loop in `Update`:
  - Se `msg.QueryID != latestQueryID` → **drop silenzioso** (no
    `state.Hits` mutation, no toast, no log error).
  - Altrimenti applica: `state.Hits := msg.Hits`, `state.LastErr := msg.Err`,
    `state.Cursor := 0`.

Questo significa che potenzialmente più RPC sono in volo
contemporaneamente (se l'utente digita rapidamente). Lato server è uno
spreco di banda accettabile (search è leggera; il rate-limit di
Telegram interviene per RPC al secondo). Lato UX: il drop è invisibile
perché solo l'ultima qID applicata produce un render.

### D3 — Esc accettato durante RPC (deroga ADR-007)

`Esc` produce `OverlayCloseMsg` da qualunque sub-stato dell'overlay
search, **incluso** `Searching.InFlight`. La transizione `Open → Closed`
include:

- `latestQueryID := latestQueryID + 1` (invalidazione di tutte le RPC
  in volo).
- Reset di `query`, `Hits`, `Cursor`.
- Unmount dell'overlay.

Le RPC pendenti continuano lato client fino al return; al ritorno il
check `qID != latestQueryID` le scarta. **Nessun side-effect server da
mitigare** perché `messages.searchGlobal` è puramente read-only.

Razionali della deroga:

- **Read-only RPC**: non c'è "azione nascosta" da rivelare con un toast
  successivo. Diversamente da `forwardMessages` (che modifica chat di
  destinazione), `searchGlobal` non lascia traccia.
- **Pattern già covered**: il bumping di `latestQueryID` su `Close`
  fornisce la stessa "atomicità di chiusura" che ADR-007 dava bloccando
  `Esc`. L'effetto user-visible è identico (zero side-effects post-close).
- **UX**: bloccare `Esc` per 300-500ms su una ricerca testuale è
  sproporzionato. Lo user mental model di una search bar è "Esc = via
  subito".

ADR-007 elenca esplicitamente "Step 26 — Search overlay (la RPC di
search ha semantica diversa: vedi ADR futuro)" — questa è quell'ADR.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **D1+D2+D3 (scelta)**: debounce 1s + queryID-token + Esc accettato | Coerente con typing.tla / ADR-010; nessuna race visibile; meno RPC stale in volo; flood-wait risk basso | Sotto la soglia "live preview" tipica TUI moderne — accettato per scope MVP |
| Debounce on-blur (RPC solo quando l'utente smette di digitare per >1s) | 1 sola RPC per query | UX lenta; perde l'effetto "live preview" tipico delle ricerche TUI moderne |
| Cancellare il `context.Context` del searchCmd al keystroke successivo | Riduce banda | gotd/td non garantisce cancellation lato server; la RPC arriva comunque, scartare il result è equivalente al pattern queryID |
| Lock UI durante RPC (Esc bloccato come ADR-007) | Uniforme con altri overlay | UX hostile per read-only; nessun beneficio (search non muta nulla) |
| Usare timestamp invece di counter per `latestQueryID` | Allineato esattamente con ADR-010 | Ambiguità su clock skew / ridotti monotonici (vedi `Rischi` in ADR-010); counter è semplicemente più robusto per identificare l'evento |
| Debounce 150ms (più reattivo) | Più "live" | Triplo del traffico RPC su typing veloce; flood wait realistico |
| Debounce 500ms+ (più conservativo) | Meno traffico | UX percepita come laggy; sotto la soglia di "live search" attesa |

## Conseguenze

- **Positive**:
  - Pattern coerente (counter + re-arm + drop-stale) **riusabile** per
    futuri overlay che fanno RPC read-only frequenti: command palette
    server-side fuzzy (Step 28), suggest in-input mention (out of pipeline).
  - Modello TLA+ `search.tla` verifica le sei invarianti chiave
    (`MONOTONIC_QUERYID`, `STALE_DEBOUNCE_BENIGN`, `STALE_RESULT_DROP`,
    `CLOSE_INVALIDATES_INFLIGHT`, `AT_MOST_ONE_FRESH_RPC`,
    `EMPTY_QUERY_NO_RPC`) in ~10⁴ stati, esecuzione TLC <5s.
  - UX "blur to commit": 1s lascia all'utente tempo per completare la
    query prima del fire RPC.
  - `Esc` sempre responsive → niente "frozen overlay" complaint.
  - Implementazione Go minimale: counter `uint64` + `tea.Tick` + check.
- **Negative**:
  - RPC stale possibili ma rare con 1s debounce. Stima: pausa naturale
    fra parole > 300ms → tipicamente 0-1 RPC stale in volo. Sotto i
    limiti rate-limit Telegram (10 RPC/s aggregate per categoria).
  - Assunzione implicita: `latestQueryID` `uint64` non overflow-a in
    pratica (overflow a 2⁶⁴ = ~10¹⁹ keystroke, irrilevante).
  - L'utente vede risultati ~1s dopo l'ultima keystroke. Atteso per
    pattern "blur to commit".
- **Rischi**:
  - **Re-open rapidamente dopo close**: se l'utente preme `Esc` poi `/`
    entro 1s, il vecchio `tea.Tick` può scadere e produrre
    `SearchDebounceFiredMsg{qID=k}` mentre `latestQueryID` è già `k+2`
    (dopo bump da Close + bump da nuovo Open). Il check `qID !=
    latestQueryID` lo scarta. **Verificato in `search.tla` con
    `MaxKeystrokes=3` e doppia open/close: nessuna violazione.**
  - **Flood wait** (`FLOOD_WAIT_xxx`): può arrivare al return RPC se
    l'utente digita molto in modo prolungato. Gestito dal path
    `Open.Error` (toast + keep typing per retry). Out-of-scope retry
    automatico (richiederebbe backoff strategy → ADR futuro se diventa
    un problema reale).
  - **Modal primitive**: l'overlay deve usare la primitive Charm-style
    unificata (`feedback_modal_charm.md`). Se la primitive non esiste,
    va estratta da edit/forward/delete overlay in via opportunistica
    PRIMA di scrivere il codice di `search.go`. Questo non è oggetto
    primario dell'ADR ma è scope-relevant.

## Scope

Questa ADR si applica a:

- **Step 26 — Search globale** (prima applicazione).
- Step 28 — Command palette: il fuzzy match commands è in-memory (no
  RPC), quindi non eredita la deroga ADR-007. Eredita però il pattern
  `Esc` accettato (overlay non-mutativo).
- Step futuri che introducono overlay con RPC read-only frequenti
  (es. server-side mention/sticker suggest): pattern counter + drop-stale
  riusabile direttamente.

**Non si applica a** (rimane sotto ADR-007):

- Forward picker (Step 21, 22) — RPC mutativa.
- Edit overlay (Step 19) — RPC mutativa.
- Delete confirm (Step 20) — RPC mutativa.

## Cross-links

- [`phase-2-behavioral/search-overlay.md`](../phase-2-behavioral/search-overlay.md) §Statechart, §Invarianti
- [`phase-3-interactions/search-flow.md`](../phase-3-interactions/search-flow.md) §Race + Cancel
- [`phase-4-concurrency/search.tla`](../phase-4-concurrency/search.tla) — invarianti `MONOTONIC_QUERYID`, `STALE_RESULT_DROP`, `CLOSE_INVALIDATES_INFLIGHT`
- [`phase-5-data/domain-types.md`](../phase-5-data/domain-types.md) §SearchOverlayState
- [ADR-007](ADR-007-overlay-in-flight-rpc.md) — pattern uniforme, qui derogato per read-only
- [ADR-010](ADR-010-typing-ttl-strategy.md) — pattern timestamp + re-arm da cui deriva il counter + re-arm
- Pipeline Step 26
- Memory: `feedback_modal_charm.md` — primitive modale unificata
