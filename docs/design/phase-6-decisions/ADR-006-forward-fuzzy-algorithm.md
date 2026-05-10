# ADR-006: Algoritmo fuzzy-search del forward picker

**Stato**: accettato
**Data**: 2026-04-24

## Contesto

Lo Step 21 introduce il **forward picker overlay** (vedi
[`phase-2-behavioral/forward-picker.md`](../phase-2-behavioral/forward-picker.md)).
L'utente digita una query e la lista delle chat candidate viene filtrata in
tempo reale.

Vincoli:

- La ricerca deve essere **istantanea** (re-rank ad ogni keystroke, nessun
  debounce-RPC): la sorgente è la lista dialogs già in cache locale.
- La dimensione tipica dialogs è O(10²–10³); il costo dev'essere dominato
  dalla CPU locale, non da allocazioni o regex compile.
- Il match deve essere **forgiving** ("team" → "Team Dev", "tdv" → "Team Dev").
- L'implementazione dev'essere semplice e testabile; no dipendenze nuove se
  possibile.

Lo Step 22 (multi-select batch forward) riuserà lo stesso picker senza
cambiare l'algoritmo di ranking.

## Decisione

Implementare un **fuzzy matcher in-house**, pure-Go, con questa scoring
function:

1. **Prefix match** su `Chat.Title` (case-insensitive) → score 1000 + bonus
   per lunghezza query.
2. **Contiguous substring match** su `Title` o `@username` → score 500 +
   bonus posizione (prima = migliore).
3. **Subsequence match** (caratteri della query appaiono nell'ordine ma non
   contigui) → score 100 + penalità per "gap" tra caratteri matched.
4. **No match** → chat esclusa dalla lista.

Tie-break per score uguale: ordinamento originale (pinned first, poi recency).

Target di riferimento: algoritmo ispirato a
[fzf](https://github.com/junegunn/fzf)'s v1 scoring, semplificato. Nessuna
dipendenza esterna; ~60 LOC in `internal/ui/components/fuzzy.go`.

Query vuota → nessun filtro, lista intera nell'ordine originale.
Matching è **Unicode-safe** (normalize NFC, case-fold via `unicode`).

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **Fuzzy in-house (scelta)** | Nessuna dep, pieno controllo, <100 LOC | Richiede test di ranking |
| `github.com/sahilm/fuzzy` | Libreria provata, API semplice | Nuova dipendenza per feature singola |
| `github.com/junegunn/fzf` lib | Algoritmo gold-standard | Heavyweight per un semplice picker |
| Sostring contains (no fuzzy) | Banale | Esperienza povera: "tdv" non matcha "Team Dev" |
| Regex user-supplied | Potente | Overkill, errori di sintassi, lento |
| Levenshtein distance | Buono per typo | Lento su liste grandi, tuning soglia difficile |

## Conseguenze

- **Positive**:
  - Zero dipendenze nuove (coerente con policy `go mod tidy` minimale).
  - Algoritmo riusabile da altri picker futuri (command palette Step 31,
    search globale Step 26).
  - Testabile in isolamento (unit test su scoring function).
- **Negative**:
  - Richiede un piccolo body di test per validare il ranking ("team" deve
    precedere "Outstream Tech" quando si digita "team").
  - Non supporta ancora highlight visivo dei caratteri matched (TODO Step 22
    o successivo).
- **Rischi**:
  - Su dialogs list molto grandi (10⁴+) il re-rank ad ogni keystroke potrebbe
    laggare. Mitigazione: se misurato >16ms, aggiungere debounce 50ms. Per
    ora non necessario (target utente tipico <500 dialogs).

## Cross-links

- [`phase-2-behavioral/forward-picker.md`](../phase-2-behavioral/forward-picker.md) §Regole di ranking
- [`phase-3-interactions/forward-flow.md`](../phase-3-interactions/forward-flow.md) §Fuzzy filter — detail
- Pipeline Step 21
