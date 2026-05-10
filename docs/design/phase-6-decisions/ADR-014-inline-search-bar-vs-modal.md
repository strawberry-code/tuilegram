# ADR-014: Search in chat — inline bar (no Modal) + sync local search + re-index strategy

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 27 introduce la **ricerca locale nella conversazione attiva**:
`Ctrl+F` apre una barra di ricerca; la query è eseguita sui messaggi
già in memoria; le occorrenze sono **highlight-ate inline** nel
viewport; `Enter`/`n` naviga al prossimo match, `Esc` chiude.

Tre decisioni vanno prese insieme perché si influenzano:

1. **Quale primitive UI?** Lo Step 26 ha appena introdotto la
   primitive `Modal` Crush-style (full-screen overlay con bordo, title,
   hint, body) per la search globale. Riusarla anche qui è tentante
   per coerenza visuale, ma stravolge la UX (la search in chat è un
   "augment del viewport", non una modalità separata).

2. **Quale algoritmo di ricerca?** Substring vs fuzzy vs regex; sync
   vs async; case-sensitive o no; quali campi (Text, captions,
   reactions, sender names).

3. **Come gestire la concorrenza con `NewMessageMsg` e `LoadMoreMsg`
   che mutano la lista `messages` mentre la barra è aperta?** Un nuovo
   messaggio può match-are la query corrente; un `LoadMoreMsg`
   pre-pende un blocco di history che potrebbe contenere N nuovi
   match. Cosa fa `currentIdx`?

Per (1), notiamo che la search globale (Step 26) e la search in chat
(Step 27) sono **strutturalmente diverse**:

| Aspetto | Step 26 (globale) | Step 27 (in chat) |
|---------|--------------------|-------------------|
| Sorgente dati | RPC `messages.searchGlobal` | slice `[]Message` in memoria |
| Latency | 200-2000ms (rete) | <1ms (CPU local scan) |
| Cardinalità risultati | 0..50 hit | 0..N (con N = `len(messages)`) |
| Visualizzazione | lista risultati separata | highlight inline nel viewport |
| Modalità | full-screen overlay (cattura tutto l'input) | augment del viewport (background visibile) |
| Scope | tutte le chat | conversazione attiva |
| Esc-during-async | sotto pattern ADR-013 | irrilevante (no async) |

Forzare il `Modal` primitive su Step 27 produrrebbe:
- Un overlay full-screen che **nasconde** il viewport (mentre l'utente
  vuole VEDERE il highlight inline).
- Un body separato per la lista risultati (mentre la lista risultati
  È il viewport stesso).
- Un cambio di mental model violento ("ho aperto una finestra" vs "ho
  aperto una barra dentro la chat").

Per (2), bench-mark di TUI editor con search in-buffer:
- Vim `/` → substring case-sensitive (con `\c` per insensitive),
  navigazione `n`/`N`, sync.
- VS Code `Ctrl+F` → substring case-insensitive di default, opzioni
  toggleable (regex, case, word), sync (per buffer in-memory).
- Telescope.nvim → fuzzy matcher, async (debounced) — ma è per
  liste arbitrarie, non per buffer di testo.

Per la nostra UX (search in conversazione, dimensione tipica
`<1000 msgs`), substring case-insensitive sync è il punto sweet.

Per (3), tre opzioni:
- **Snapshot frozen**: `index` e `matches` calcolati all'apertura
  della barra; `NewMessageMsg` ignorato fino a `Esc`. Semplice ma
  l'utente vede il counter `1/3` mentre arrivano nuovi msg che
  match-ano e non vengono contati. Disorientante.
- **Re-index incrementale**: append/prepend a `index` e `matches` ad
  ogni mutazione; `currentIdx` shift-ato per preservare l'identità
  del match corrente. Più complesso ma UX corretta.
- **Re-search completo**: ogni mutazione triggera un re-scan completo
  dell'intero `index`. Inutilmente costoso (`O(N * |q|)`) per il
  beneficio (`O(1)` di lavoro per messaggio nuovo).

## Decisione

**Triplice decisione consolidata in una sola ADR perché interconnessa.**

### D1 — Inline search bar, NO `Modal` primitive

Step 27 usa una **barra inline** tipo `bubbles/textinput` agganciata
al footer della `ConversationModel` (sotto la `viewport` dei messaggi
e sopra la status bar):

```
┌─ ChatList ─┬─ Conversation: chatA ───────────────────┐
│ ...        │  [msg history with highlight ...]       │
│ chatA  *   │  bg=accent on substring matches         │
│ chatB      │  bg=accent.bold on currentIdx match     │
│            ├─────────────────────────────────────────┤
│            │ [Search: hello]  2/5  ↵ next ⇧↹ prev ⎋ │ <- inline bar
│            ├─────────────────────────────────────────┤
│            │ status bar                              │
└────────────┴─────────────────────────────────────────┘
```

NON usa la `Modal` primitive Crush-style introdotta nello Step 26
(`internal/ui/components/modal.go`). Razionale:

- **Background visibility**: la search in chat È un augment del
  viewport, non una modalità separata. L'utente DEVE vedere i match
  inline; un overlay full-screen li nasconderebbe.
- **Mental model**: "ho aperto una barra di ricerca dentro la chat"
  vs "ho aperto un overlay di ricerca sopra l'app". Il primo è
  vim/VS Code-style; il secondo è file-explorer-style. Per ricerca
  in-buffer, il primo è universalmente atteso.
- **No focus capture totale**: la barra cattura solo i tasti di
  navigation/edit (char, backspace, Enter, Shift+Tab, Esc). Le
  scorciatoie globali (`Ctrl+P` palette, `?` help, `/` search
  globale) restano accessibili. La `Modal` primitive invece è
  "modale dura": cattura tutto.
- **Composizione lipgloss**: il footer-bar è uno `lipgloss.JoinVertical`
  in più nella `ConversationModel.View()`. Triviale da implementare,
  zero new component.

La `Modal` primitive resta usata da search globale (Step 26), command
palette (Step 28), help overlay (Step 28), confirm dialog (Step 20),
edit overlay (Step 19), forward picker (Step 21). La regola euristica:
**`Modal` per overlay che CAMBIANO il pannello attivo**; **inline bar
per augment in-place dello stesso pannello**.

Memoria utente `feedback_modal_charm.md` rispettata: gli overlay che
sono effettivamente overlay continuano a usare la primitive
unificata; questa NON è un overlay, è un sub-stato del viewport
stesso.

### D2 — Algoritmo: substring case-insensitive, sync, solo `Text`

`SearchInChatQueryChangedMsg` triggera un **re-compute sincrono**
nello stesso `Update` cycle:

```
qLC := strings.ToLower(query)
matches := []
for im in state.index:
    if im.textLC contains qLC (strings.Index/Contains):
        matches.append({msgID: im.msgID, spans: ...})
```

- **Substring**, no fuzzy / regex / glob (semplice e prevedibile).
- **Case-insensitive** (lowercased pre-build dell'index +
  lowercased query una volta sola).
- **Solo `Message.Text`**: media captions, reactions emoji, sender
  names sono **esclusi** (out-of-scope MVP).
- **Service messages esclusi** (`m.IsService == TRUE`): non contengono
  contenuto utente, non sono semanticamente cercabili. Coerente con
  `SYSTEM_NO_REACT` di `reactions.tla`.
- **Sync nel main loop**: per `len(messages) <= 1000` (tipico),
  `O(N * |q|)` è <1ms. NESSUN `tea.Cmd` async; NESSUN debounce; ogni
  keystroke produce un re-render immediato.

#### Re-index incrementale su mutazione

Mentre la barra è aperta, mutazioni della lista `messages` triggrano
un **re-index incrementale** (NON re-search completo):

| Mutazione | Effetto |
|-----------|---------|
| `NewMessageMsg{m}` (m.ChatID == active, !m.IsService, m.Text != "") | append a `index`; se query non vuota e `m` matcha → append a `matches`; `currentIdx` invariato |
| `NewMessageMsg{m}` con isService o Text="" | no-op su `index` |
| `LoadMoreMsg{newMsgs}` (history pre-pend) | prepend filtered a `index`; scan `newMsgs` vs query → prepend dei nuovi `matches`; `currentIdx += len(newMatches)` per **preservare l'identità** del match corrente |
| `MessageDeletedMsg{ids}` | rimuove da `index` e `matches`; clamp `currentIdx` |
| `MessageEditedMsg{id, newText}` | aggiorna `index[id].textLC`; ri-scan solo quel msg vs query; aggiorna o rimuove entry in `matches`; clamp `currentIdx` |
| `ReactionsUpdatedMsg` | no-op (solo `Text` è cercato) |

**Match identity preservation principle**: per ogni mutazione che
NON tocca il messaggio corrispondente a `matches[currentIdx]`,
l'identità (msgID) del match corrente è preservata. L'utente non
vede "saltare" il highlight per effetto di un `NewMessageMsg`
arrivato in background. Verificato in
[`../phase-4-concurrency/search_in_chat.tla`](../phase-4-concurrency/search_in_chat.tla)
invariante `MATCH_IDENTITY_PRESERVED_*`.

**No cascading messages**: il re-index NON emette nuovi `tea.Msg` di
tipo `SearchInChat*`. È side-effect del gestore `App.Update` per
`NewMessageMsg`/`LoadMoreMsg`/`MessageDeletedMsg`/`MessageEditedMsg`
quando `searchInChat.active == true`. Questo evita race tra eventi
di re-index e eventi di navigation utente nello stesso frame.

### D3 — Esc semantics + global keys passthrough

- `Esc` chiude la barra e ripristina lo stato precedente
  (`returnTo` = `BrowsingMessages` o `MultiSelect`). Il
  multi-select set `S` è preservato.
- `Ctrl+P`, `?`, `/` (scorciatoie globali a livello root) passano al
  root model anche con la barra aperta. Razionale: l'utente può
  voler aprire la search globale (`/`) o la palette (`Ctrl+P`)
  direttamente dalla search in chat, senza prima `Esc`. Quando
  l'overlay globale viene chiuso, la search in chat riemerge.
- `Tab`, `Ctrl+F` sono **ignorati** dentro la barra (non c'è focus
  traversal mentre la barra cattura input; la barra è già aperta).
- `j`/`k`, `Space`, lettere sono catturati come char dal textinput
  (sono parte della query).

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **D1+D2+D3 (scelta)**: inline bar + sync substring + re-index incrementale + global passthrough | UX vim/VS Code-aligned; zero RPC; identità match preservata; coerente con `feedback_modal_charm.md` (Modal usato solo dove serve) | Asimmetrica vs Step 26 — richiede docs chiari per disambiguare |
| Riusare `Modal` Crush-style anche per Step 27 | Coerenza visuale 100% | Nasconde viewport (highlight inline impossibile); UX violentemente diversa da vim/VS Code; viola "Modal per overlay che cambiano pannello" |
| Snapshot frozen (no re-index su NewMessageMsg) | Implementazione più semplice | Counter `1/N` mente quando arrivano nuovi msg; UX disorientante in chat attive |
| Re-search completo ad ogni mutazione | Più semplice del re-index incrementale | `O(N * |q|)` per ogni nuovo msg; spreco evitabile |
| Async search via `tea.Cmd` (come Step 26) | Pattern uniforme | Inutile per data in-memory <1ms; aggiunge complessità senza beneficio |
| Fuzzy matcher (sahilm/fuzzy) | Match più tollerante a typo | Out-of-scope MVP; può essere aggiunto come opzione futura senza breaking |
| Regex search di default | Power user-friendly | Errori di regex visibili; complessità sintattica; out-of-scope MVP |
| Search anche su captions / sender / reactions | Più completa | Aumenta complessità di indexing; UX confusa (highlight su zone non-text); out-of-scope MVP |
| `Esc` chiude barra E lascia highlight (toggle) | "Persistent search" come VS Code | UX più complessa (state-keeping post-close); divergenza da vim model; out-of-scope MVP |
| `Ctrl+P` / `/` bloccati con barra aperta | Modale puro | Forza `Esc` redundante per accedere a feature globali; UX-hostile |

## Conseguenze

- **Positive**:
  - **Pattern chiaro per future inline-bar**: input bar agganciata a
    un pannello specifico (vs overlay full-screen) è riusabile per
    futuri feature in-pannello (es. quick-jump nel chat list, filter
    folder).
  - **Modello TLA+ `search_in_chat.tla` verifica le invarianti chiave**
    (`MATCH_IDENTITY_PRESERVED_*`, `NO_PHANTOM_MATCH`,
    `SYSTEM_NOT_INDEXED`, `CURSOR_BOUNDED`,
    `INDEX_CONSISTENT_WITH_MESSAGES`, `INACTIVE_CLEAN`,
    `QUERY_EMPTY_NO_MATCHES`) in ~10⁴ stati, esecuzione TLC <5s.
  - **UX vim/VS Code-aligned**: `Ctrl+F` → barra → `n`/`N` o
    `Enter`/`Shift+Tab` per next/prev → `Esc` chiude. Mental model
    universale.
  - **Identità match preservata**: `NewMessageMsg` o `LoadMoreMsg` in
    background non disturbano la navigation. UX stabile in chat
    attive.
  - **Zero RPC, zero debounce, zero overhead**: search sincrona,
    `tea.Msg`-only, side-effect-free.
  - **`Modal` primitive non viene "abusata"**: rimane usata SOLO dove
    serve davvero (overlay che cambiano pannello attivo). Coerente
    con `feedback_modal_charm.md`.
- **Negative**:
  - **Asimmetria con Step 26**: due approcci diversi alla "search"
    nello stesso prodotto. Mitigato da docs chiari (statechart,
    questa ADR, sequence diagram); l'utente percepisce due feature
    distinte (`/` globale vs `Ctrl+F` locale) quindi l'asimmetria è
    naturale.
  - **`SearchInChat*Msg` types nuovi** (5: Open, QueryChanged,
    ResultsComputed, Next/Prev, Close): aggiungono nomi alla
    taxonomy. Mitigato dal prefisso `SearchInChat` che evita
    collisione semantica con `Search*Msg` di Step 26.
  - **Re-index incrementale ha invarianti delicate**
    (`currentIdx` shift su prepend): bug-prone se non testato.
    Mitigato dal modello TLA+ che esplicita gli invarianti e dal
    fatto che il pattern è isolato in `ConversationModel` (no
    cross-component complexity).
- **Rischi**:
  - **Performance su chat molto lunghe**: per `N > 10⁴` messaggi, ogni
    keystroke costa `O(N * |q|)`. Stima: a `N=10⁴`, `|q|=5`, ~1-5ms
    per CPU moderna. Sotto la soglia di percezione ma vicino. Se
    utenti reali si lamentano, valutare un trie o un index inverso
    (out-of-scope Step 27 → ADR futuro se necessario).
  - **Highlight rendering cost**: ricreare gli span lipgloss ad ogni
    keystroke per tutti i match visibili può aggravare il render.
    Mitigato da: re-render solo dei messaggi nel viewport range
    (non l'intera history). Se diventa un problema, valutare un
    `viewport.SetContent` differenziale (out-of-scope Step 27).
  - **Race tra re-index e navigation utente**: l'utente preme
    `Enter` (next) nello stesso frame in cui arriva
    `NewMessageMsg`. bubbletea serializza i `tea.Msg` in single
    channel quindi l'ordine è deterministico ma non noto a priori.
    L'invariante `MATCH_IDENTITY_PRESERVED_NEW` garantisce che il
    risultato è corretto in entrambi gli ordini (verificato in TLA+).
  - **`Ctrl+F` può collidere con scorciatoie utenti**: alcuni terminali
    legacy intercettano `Ctrl+F` per page-down. Mitigato da
    documentazione (`?` help) e da fallback futuro a `/`-in-chat se
    necessario (out-of-scope decisione finale del binding).

## Scope

Questa ADR si applica a:

- **Step 27 — Search in conversazione + Ctrl+F** (prima e unica
  applicazione attesa).
- Step futuri che introducono **inline-bar input** in-pannello
  (es. quick-jump in chat list, filter folder, inline command in
  input pane): ereditano D1 (inline-bar pattern) e D3 (global keys
  passthrough). D2 (algoritmo substring) è specifico della search e
  non eredita.

**Non si applica a**:

- Overlay che cambiano il pannello attivo (Step 26 search globale,
  Step 28 command palette, Step 28 help, Step 19 edit, Step 20
  confirm, Step 21 forward picker): rimangono sotto la primitive
  `Modal` di `feedback_modal_charm.md`.
- Future search server-side in chat (es. `messages.search` su
  history non interamente caricata): se introdotta, sarà un
  raffinamento di Step 27 con un `tea.Cmd` async + pattern di stale
  drop tipo ADR-013. Non rompe questa ADR (resta valido per la
  componente client-side).

## Cross-links

- [`phase-2-behavioral/search-in-chat.md`](../phase-2-behavioral/search-in-chat.md) §Statechart, §Invarianti
- [`phase-3-interactions/search-in-chat-flow.md`](../phase-3-interactions/search-in-chat-flow.md) §3 (NewMessageMsg), §4 (LoadMoreMsg)
- [`phase-4-concurrency/search_in_chat.tla`](../phase-4-concurrency/search_in_chat.tla) — invarianti `MATCH_IDENTITY_PRESERVED_*`, `NO_PHANTOM_MATCH`
- [`phase-5-data/domain-types.md`](../phase-5-data/domain-types.md) §SearchInChatState (in aggiornamento)
- [`phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md) §Internal UI Messages (in aggiornamento)
- [ADR-013](ADR-013-search-debounce-and-stale-results.md) — search globale Step 26 (NON applicabile qui per assenza di RPC)
- Pipeline Step 27
- Memoria utente: `feedback_modal_charm.md` (Modal usato selettivamente, qui giustificato non-uso)
