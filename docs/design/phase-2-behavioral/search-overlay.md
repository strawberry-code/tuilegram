# Search Overlay — Statechart (Step 26)

Modello comportamentale dell'**overlay di ricerca globale** introdotto nello
Step 26 della pipeline. `/` apre un overlay modale con textinput + lista
risultati; la ricerca colpisce `api.MessagesSearchGlobal` con **debounce
300ms**; `Enter` su un risultato chiude l'overlay e fa "jump-to-message"
nella chat di destinazione.

**Scope Step 26**:
- Search **globale** (tutte le chat) via `MessagesSearchGlobal`.
- Debounce 300ms tra l'ultima keystroke e l'invio della RPC.
- Drop di risultati staled (RPC più vecchie del query attuale).
- Jump cross-chat: chiusura overlay + apertura chat target + centratura
  viewport sul `messageID` del hit.

**Fuori scope Step 26**:
- Search nella sola conversazione attiva (Step 27, `Ctrl+F`).
- Filtri server-side per peer/folder/data (`MessagesSearchGlobal` accetta
  filtri ma non li esponiamo in UI in Step 26).
- Highlight in-line del match nel viewport (Step 27).
- Paginazione: prendiamo i primi N hit del primo batch (no infinite
  scroll). Il numero è discusso in [ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md).

## Contesto nello statechart globale

L'overlay è un figlio di `Overlay.SearchOverlay` (vedi
[`ui-statechart.md`](ui-statechart.md), sezione "Overlay State Machine").
È raggiungibile da:

- `MainView` qualunque sia il pannello focused (è uno dei keybindings
  globali, vedi tabella "Regole di focus" in `ui-statechart.md`).

A differenza del **forward picker** (Step 21), l'overlay search:

1. **Non ha source snapshot**: non opera su un messaggio sotto cursore;
   apre vuoto.
2. **Emette RPC per-keystroke** (debounced), non sull'`Enter`. `Enter`
   triggers solo la **navigation**, non una RPC.
3. **Cancellation-safe**: la RPC in volo viene **scartata** (non
   cancellata server-side) tramite il pattern `queryID` monotono. La
   chiusura dell'overlay deve invalidare i risultati pendenti.

Per queste ragioni Step 26 introduce [ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md)
in deroga al pattern uniforme di [ADR-007](../phase-6-decisions/ADR-007-overlay-in-flight-rpc.md):
**`Esc` chiude l'overlay anche se una RPC è in volo** (i risultati
verranno scartati dal check `queryID`).

## Statechart dell'overlay

```mermaid
stateDiagram-v2
    [*] --> Closed

    Closed --> Opening : SearchOpenMsg ('/')
    Opening --> Idle : overlay mounted, textinput focused

    state "Open (modal)" as Open {
        [*] --> Idle

        Idle --> Typing : char typed → SearchQueryChangedMsg{q, qID++}<br/>schedule tea.Tick(300ms) → SearchDebounceFiredMsg{qID}
        Typing --> Typing : char/backspace → SearchQueryChangedMsg{q', qID++}<br/>schedule new tick (re-arm; old ticks stale)

        Typing --> Searching : SearchDebounceFiredMsg{qID}<br/>where qID == latestQueryID && q != ""<br/>spawn searchCmd(q, qID)
        Typing --> Idle : SearchDebounceFiredMsg{qID}<br/>where qID == latestQueryID && q == ""<br/>(empty query: clear results, no RPC)
        Typing --> Typing : SearchDebounceFiredMsg{qID}<br/>where qID < latestQueryID<br/>(stale tick: no-op)

        state Searching {
            [*] --> InFlight
            InFlight --> InFlight : char typed (re-types during RPC) → SearchQueryChangedMsg{q', qID++}<br/>(latestQueryID bumped; in-flight result will be dropped)
            InFlight --> Results : SearchResultMsg{qID, hits}<br/>where qID == latestQueryID
            InFlight --> InFlight : SearchResultMsg{qID, hits}<br/>where qID != latestQueryID<br/>(stale RPC: dropped silently)
            InFlight --> Error : SearchResultMsg{qID, err}<br/>where qID == latestQueryID
        }

        state Results {
            [*] --> ShowingHits
            ShowingHits --> ShowingHits : j/k → SearchCursorMsg{delta}
            ShowingHits --> Empty : (computed) hits == []
            Empty --> ShowingHits : new query yields hits
        }

        Results --> Typing : char/backspace (new query)
        Searching --> Typing : char/backspace (new query, qID bumped, see InFlight self-loop)
    }

    Open --> Closed : Esc → OverlayCloseMsg<br/>(allowed from any sub-state, including Searching;<br/>see ADR-013)
    Results --> Closed : Enter on hit → SearchSubmitMsg{hit}<br/>then JumpToMessageMsg{chatID, msgID}

    Closed --> [*]

    note right of Searching
      Esc è ACCETTATO durante InFlight (deroga ADR-007).
      Razionale in ADR-013: la RPC search è side-effect-free
      (read-only) e i risultati sono scartati via queryID check.
    end note
```

## Stati — descrizione

| Stato | Descrizione | Input accettati | Componenti attivi |
|-------|-------------|-----------------|-------------------|
| `Closed` | Overlay non montato | — | — |
| `Opening` | Overlay appena triggrato; mounting in corso (frame singolo) | — | textinput (vuoto) |
| `Open.Idle` | Overlay aperto, query vuota o lista risultati appena cleared | char, `Esc` | textinput |
| `Open.Typing` | Utente sta digitando; debounce in attesa di scadere | char, backspace, `Esc` | textinput, "Searching…" placeholder se cursor su `Searching` |
| `Open.Searching.InFlight` | RPC `MessagesSearchGlobal` in volo per `latestQueryID` | char, backspace, `Esc` | textinput + spinner "Searching…" |
| `Open.Results.ShowingHits` | RPC tornata con `hits != []` | char, backspace, `j/k`, `Enter`, `Esc` | textinput + lista hits + scroll cursor |
| `Open.Results.Empty` | RPC tornata con `hits == []` | char, backspace, `Esc` | textinput + placeholder "No results" |
| `Open.Error` | RPC fallita (e.g. flood wait, network) | char, backspace, `Esc` | textinput + toast errore + "Retry typing to search again" |

## Eventi / Messaggi (tipizzati `tea.Msg`)

Estendono [`message-taxonomy.md`](../phase-1-context/message-taxonomy.md).

| Msg | Origine | Payload | Effetto |
|-----|---------|---------|---------|
| `SearchOpenMsg` | Keystroke `/` (App livello root) | — | `Closed → Opening`; reset `latestQueryID := 0`, `query := ""`, `hits := []` |
| `SearchQueryChangedMsg` | textinput change handler nell'overlay | `query string, queryID uint64, scheduledAt time.Time` | Bumping `latestQueryID := queryID`; schedule `tea.Tick(300ms) → SearchDebounceFiredMsg{queryID}` |
| `SearchDebounceFiredMsg` | `tea.Tick(300ms)` | `queryID uint64` | Se `queryID == latestQueryID` e `query != ""` → spawn `searchCmd(query, queryID)`; se `queryID < latestQueryID` → **no-op** (stale debounce); se `query == ""` → clear hits, no RPC |
| `SearchResultMsg` | `searchCmd` (goroutine) | `queryID uint64, hits []SearchHit, err error` | Se `queryID != latestQueryID` → **drop** (stale RPC, ADR-013); altrimenti popola `hits`, transition a `Results.ShowingHits` (o `Empty` o `Error`) |
| `SearchCursorMsg` | `j/k` nell'overlay | `delta int` | Sposta cursore lista (no RPC) |
| `SearchSubmitMsg` | `Enter` su un hit | `SearchHit` | Emette `OverlayCloseMsg` + `JumpToMessageMsg{chatID, msgID}` come `tea.Batch` |
| `JumpToMessageMsg` | App.Update post `SearchSubmitMsg` | `chatID ChatID, messageID int` | Se `chatID != activeChatID` → `loadChatCmd(chatID)` con flag `centerOn=msgID`; altrimenti scroll viewport a `msgID` |
| `OverlayCloseMsg` | `Esc` (qualunque sub-stato) | — | `Open.* → Closed`; non cancella RPC pendenti server-side, ma drop result via `queryID` |

## Keybindings (Open state)

| Tasto | Azione |
|-------|--------|
| char printable | Append a query → `SearchQueryChangedMsg` |
| `Backspace` | Rimuove ultimo char → `SearchQueryChangedMsg` |
| `j` / `↓` | Cursore lista hits ++ → `SearchCursorMsg{+1}` |
| `k` / `↑` | Cursore lista hits -- → `SearchCursorMsg{-1}` |
| `Enter` | `SearchSubmitMsg{hits[cursor]}` (no-op se `len(hits) == 0`) |
| `Esc` | `OverlayCloseMsg` (sempre, anche durante `Searching.InFlight`) |
| `Tab` | **Ignorato** (overlay modale) |
| `/` | **Ignorato** dentro l'overlay (gia` aperto) |

## Modello dati associato

L'overlay tiene state locale:

```
search : SearchOverlayState

SearchOverlayState ::= {
    query           : string
    latestQueryID   : uint64        // monotonic counter (incrementato ad ogni keystroke)
    hits            : []SearchHit
    cursor          : int           // indice in hits, 0..len(hits)-1
    inFlight        : bool          // true se searchCmd è in volo per latestQueryID
    lastErr         : error         // ultimo errore RPC, nil se ok
}
```

Il `latestQueryID` è il **token di freschezza** che permette di scartare
sia i tick di debounce stale sia i `SearchResultMsg` da RPC stale.

## Regola di freshness — `latestQueryID`

Pattern preso da [ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md)
("timestamp + re-arm") ed esteso a "monotonic counter + drop-stale".
Formalizzato in [ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md).

```
on SearchQueryChangedMsg{q}:
    latestQueryID++
    schedule tea.Tick(300ms) → SearchDebounceFiredMsg{latestQueryID}

on SearchDebounceFiredMsg{qID}:
    if qID != latestQueryID: return                       // stale tick, no-op
    if query == "":            hits := []; return         // empty, no RPC
    spawn searchCmd(query, qID)

on SearchResultMsg{qID, hits, err}:
    if qID != latestQueryID: return                       // stale RPC, drop
    state.hits := hits; state.lastErr := err

on OverlayCloseMsg:
    latestQueryID++                                       // invalidate any in-flight RPC
    state := initial
```

L'incremento di `latestQueryID` su `OverlayCloseMsg` garantisce che
qualunque `SearchResultMsg` arrivi dopo la chiusura sia scartato (anche
se dovesse riaprirsi un overlay nel frattempo, cosa improbabile in 300ms).

## Invarianti comportamentali

1. **Modal**: in `Open.*`, input non raggiunge mai pannelli sottostanti
   (ChatList/Conversation/Input).
2. **Esc-anywhere**: `Esc` è sempre accettato (deroga a ADR-007); la
   chiusura non lascia state inconsistente perché i risultati pendenti
   sono droppati via `queryID` check (vedi `STALE_RESULT_DROP` in
   `search.tla`).
3. **Monotonic queryID**: `latestQueryID` è strettamente crescente
   nell'arco di vita del processo (mai decremento, mai reset a 0 se non
   alla `SearchOpenMsg` iniziale che parte da `0`).
4. **No race su risultati**: per ogni `SearchResultMsg` ricevuta vale
   l'invariante `qID <= latestQueryID` (la RPC non può essere stata
   spawned con un qID > di quello noto al main loop). Se `qID <
   latestQueryID` → drop. Se `qID == latestQueryID` → applica.
5. **Cursor reset on results**: ogni nuovo `SearchResultMsg` valido
   resetta `cursor := 0`.
6. **Empty query no RPC**: una query vuota non emette mai
   `MessagesSearchGlobal` (filtro client-side); `hits := []`,
   transition a `Idle`.
7. **Jump idempotente**: due `SearchSubmitMsg` consecutivi sullo stesso
   hit producono lo stesso effetto del primo (overlay già chiuso →
   secondo `OverlayCloseMsg` no-op; `JumpToMessageMsg` ricarica la
   stessa chat allo stesso offset).
8. **Cross-chat handoff**: il jump emette eventualmente `loadChatCmd`
   se la chat target è diversa da quella attiva — la centratura del
   viewport sul `messageID` avviene **dopo** che `MessagesLoadedMsg`
   è arrivato, non immediatamente (vedi
   [`../phase-3-interactions/search-flow.md`](../phase-3-interactions/search-flow.md)
   §"Jump cross-chat").

## Loading / Empty / Error states — render

| Stato dati | Render lista hits |
|-----------|-------------------|
| `query == ""`, `hits == []` | placeholder dim: `Type to search messages…` |
| `query != ""`, `inFlight == true`, `hits == []` | spinner + `Searching…` |
| `query != ""`, `inFlight == true`, `hits != []` (digit. mentre RPC in volo) | hits precedenti + spinner discreto in header overlay |
| `query != ""`, `inFlight == false`, `hits == []`, `lastErr == nil` | placeholder: `No results for "<query>"` |
| `query != ""`, `inFlight == false`, `hits != []` | lista hits scrollabile, cursor highlight |
| `query != ""`, `inFlight == false`, `lastErr != nil` | toast errore in-overlay: `Search failed: <reason> · keep typing to retry` |

## Modal primitive — riuso

Per il requisito ricordato in `feedback_modal_charm.md` (memory utente),
l'overlay deve usare la **primitive modale unificata** ispirata allo
stile Crush (un `internal/ui/components/modal.go` o equivalente) e
**non** ri-implementare bordi/title/hint da zero.

L'API attesa è:
- `title: "Search"`
- `body: tea.Model` (textinput + lista hits)
- `hint: "↵ jump · ↑↓ navigate · esc close"`
- `state: { input | searching | results | empty | error }` per drive del
  rendering loading/error.

Se la primitive non esiste ancora al momento dell'implementazione, va
**estratta opportunisticamente** dall'overlay edit/forward esistenti
prima di scrivere il codice di `search.go` (refactor → step Step 26
diventa anche occasione di consolidamento, allineato a
`feedback_modal_charm.md`).

Decisione formale (estrazione vs. one-off): segnalata in
[ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md)
§"Conseguenze" — non è oggetto principale dell'ADR ma è scope-relevant.

## Cross-links

- Pipeline step: [`development-pipeline.md` §Step 26](../development-pipeline.md)
- Sequence diagrams: [`../phase-3-interactions/search-flow.md`](../phase-3-interactions/search-flow.md)
- Concurrency invariants: [`../phase-4-concurrency/search.tla`](../phase-4-concurrency/search.tla)
- Decisione debounce + stale-drop: [ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md)
- Pattern correlato (TTL + re-arm): [ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md)
- Pattern correlato (overlay in-flight RPC, **deroga**): [ADR-007](../phase-6-decisions/ADR-007-overlay-in-flight-rpc.md)
- Modal primitive (memory): `feedback_modal_charm.md`
- Forward picker (overlay precursore): [`forward-picker.md`](forward-picker.md)
