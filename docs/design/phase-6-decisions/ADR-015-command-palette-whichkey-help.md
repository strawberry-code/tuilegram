# ADR-015: Command palette, which-key, help — primitive Modal, registry statico, mutex overlays, fuzzy subsequence, debounce 300ms

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 28 introduce **tre overlay UI-only** in un colpo solo
(`Ctrl+P` palette, prefix-keys + 300ms which-key, `?` help). Il fatto
che siano tre feature separate ma simultaneamente introdotte e
strutturalmente correlate (stessa primitive, stesso lock di mutua
esclusione, stesse semantiche di `Esc`) richiede una decisione
architetturale unica e coerente.

Cinque sottodecisioni vanno prese insieme perché si influenzano:

1. **Primitive UI**: i tre overlay riusano la `Modal` Crush-style
   introdotta nello Step 26 oppure ognuno ha il suo? La memoria utente
   `feedback_modal_charm.md` impone l'uso della primitive unificata
   per gli overlay; ma which-key è un overlay piccolo, ancorato in
   basso a destra, non full-screen — vale ancora la regola?

2. **Registry source dei comandi (palette)**: i comandi sono dichiarati
   compile-time in un registry statico, oppure scoperti runtime via
   reflection / file di config? L'uso di un registry statico è il
   contratto più semplice ma rigido.

3. **Mutua esclusione fra overlay**: solo uno alla volta, oppure stack
   (per esempio: command palette → da palette si esegue "Mute" → si
   apre confirm dialog sopra la palette)? Lo stack è più potente ma
   complica significativamente lo state model e introduce ambiguità
   su `Esc` (chiude top o tutto?).

4. **Algoritmo di matching della palette**: substring `Contains`,
   subsequence (fuzzy minimale), o libreria esterna (sahilm/fuzzy,
   junegunn/fzf-style)? Le ricerche tipiche di palette ("mute",
   "fwd", "info") sono brevi e benefit di subsequence + ranking
   intelligente. Substring è troppo restrittivo. Una libreria esterna
   è una nuova dipendenza per un problema piccolo.

5. **Window di debounce which-key**: 300ms è il default proposto dalla
   pipeline. Vs. 200ms (più snappy ma rischio di reveal accidentali
   per typing rapido) vs. 500ms (più tolerante ma laggy percepito).
   Cosa fa l'industria?

Per (1), benchmark: file managers (ranger, lf), text editors (vim,
helix), TUI Telegram clients (telegram-tui, tg) usano consistentemente
una primitive overlay riusabile per palette/help, e una primitive
**piccola overlay ancorata** per which-key (es. helix usa overlay in
basso che mostra le continuations dopo timeout).

Per (2), bench:
- VS Code → registry statico generato da extension manifest + commands
  contributi runtime. Per un TUI client mono-binary, la dimensione del
  registry è limitata (~30-50 comandi), il valore aggiunto della
  scoperta runtime è ~zero.
- Helix / vim → registry statico (in Helix, da `commands.rs` compilato).
- Telescope.nvim → dynamic ma è plugin-friendly per natura.

Per (3), bench:
- Helix / vim → mutex (un solo "popup" visibile, Esc chiude top).
  Soluzione semplice; quando si esegue un comando dalla palette che
  richiede un altro popup (es. ":quit!" → confirm), la palette si
  chiude PRIMA, poi si apre il confirm. Mai stack.
- VS Code (UI grafica) → stack (una notifica può sovrapporsi a un
  modal). Ma è UI grafica, non TUI; il TUI ha real estate limitato.
- File managers (ranger) → mutex.

Per il TUI di tuilegram, lo stack porterebbe complessità senza beneficio
chiaro (l'utente raramente vuole vedere DUE overlay simultanei in 80×24).

Per (4), bench:
- Helix command palette → fuzzy subsequence con scoring (matchit/skim).
- Telescope → fzf-native binding o proprio fuzzy matcher.
- fzf cli → algoritmo proprietario subsequence + ranking.

Per la nostra UX (palette di ~30 comandi, query <10 char), un subsequence
matcher in-house con ranking semplice (consecutive bonus, word-start
bonus, length penalty) è sufficiente e zero-deps.

Per (5), bench:
- Helix `which-key` → 200ms default (configurabile).
- Spacemacs / DOOM Emacs → 400ms-500ms default.
- vim-which-key (popular Vim plugin) → 500ms default, raccomandato
  abbassare a 300ms per power user.
- Studi UX di chord input: 250-300ms è la sweet spot per discriminare
  tra "intentional pause" (utente sta pensando) e "rapid chord"
  (utente sa il chord). Sotto 200ms: rivelazioni accidentali quando
  l'utente è veloce ma esita. Sopra 500ms: laggy, l'utente non sa
  perché l'overlay non appare.

**Confronto sintetico timing**:

| Window | Pro | Contro |
|--------|-----|--------|
| 200ms | snappy reveal per esitazioni brevi | reveal accidentale per chord eseguiti normalmente (es. `gg` 250ms apart per typing un po' lento) |
| **300ms** | sweet spot UX (discriminazione robusta) | nessuno significativo |
| 500ms | molto tolerante, no false reveals | laggy percepito, utente non capisce perché l'overlay non appare |

## Decisione

**Quintuplice decisione consolidata in una sola ADR perché interconnessa.**

### D1 — Primitive `Modal` Crush-style per tutti e tre, con flag `compact`

Tutti e tre gli overlay (palette, which-key, help) usano la primitive
**`Modal`** introdotta in `internal/ui/components/modal.go` allo Step 26.

| Overlay | Mode `Modal` | Layout | Title / Hint |
|---------|--------------|--------|--------------|
| Palette | `compact: false` (full-screen centered) | textinput + lista filtered | "Commands" / `↵ run · ↑↓ navigate · esc close` |
| Which-key | `compact: true` (small, anchored bottom-right) | tabella `key → action` | (no title) / `esc cancel` |
| Help | `compact: false` (full-screen centered) | viewport scrollable con sezioni | "Keybindings" / `↑↓ scroll · esc close` |

L'estensione `compact` è una proprietà di layout della primitive
(positioning + dimensioni), NON un nuovo componente. Coerente con
`feedback_modal_charm.md`: la primitive resta una sola, il rendering
si adatta. Decisione di estendere il `Modal` con flag `compact` (vs.
creare `MiniOverlay` separato) in: rispetta DRY, evita component
sprawl, e preserva l'invariante "tutti gli overlay usano `Modal`".

### D2 — Command registry statico compile-time

Il registry dei comandi della palette è una **mappa Go statica**
dichiarata compile-time (es. `var DefaultRegistry = CommandRegistry{...}`
in `internal/ui/commands/registry.go`):

```
// Pseudo-struttura, non codice — lo Step 28 modeller non scrive Go
{
  "chat.mute": {
    Title:   "Mute current chat",
    Section: "Chat",
    Keys:    ["m"],
    Handler: func() tea.Cmd { return muteChatCmd(currentChatID) },
  },
  "chat.archive": { ... },
  "view.toggle_folders": { ... },
  ...
}
```

**Allo stesso modo**, il registry delle continuations which-key è
statico:

```
{
  "g": {
    "g": ScrollTopHandler,
    "G": ScrollBottomHandler,
    "u": JumpUnreadHandler,
    "i": ChatInfoHandler,
  },
  "z": {
    "z": ScrollCenterHandler,
    "t": ScrollToTopHandler,
    "b": ScrollToBottomHandler,
  },
}
```

Razionale:

- **Tooling**: gli IDE auto-completano i commandID e i prefix, il
  refactor (rinomina) è safe.
- **Type safety**: ogni handler è una closure tipata `func() tea.Cmd`,
  no reflection, no string-based dispatch a runtime con possibili
  panic.
- **Performance**: `O(1)` lookup, nessun parsing di file di config a
  startup.
- **Documentazione automatica**: il registry È la fonte di verità per
  la generazione del help overlay (D1). Sezione `Section`, ordering,
  e descrizione vengono lette dal registry in compile-time.

**No persistence di "recently used" / "favorites" in Step 28**: la
palette parte sempre dall'ordine canonico del registry. Una future
ADR potrà introdurre un MRU cache (out-of-scope).

**No registrazione runtime** di nuovi comandi (es. plugin system): la
pipeline tuilegram non prevede plugin. Se in futuro emergesse il
bisogno, il registry può essere esteso a `RegisterCommand(id, entry)`
mantenendo il contratto current.

### D3 — Mutua esclusione: un solo overlay attivo (no stack)

Il root model mantiene `App.activeOverlay : OverlayKind` (single-value).
Aprire un overlay quando `activeOverlay != none` è **no-op silenzioso**:
la keystroke `Ctrl+P` / `?` / `/` etc. è consumata dall'overlay attivo
(o ignorata, se non rilevante per esso).

Per la palette in particolare:

- Eseguire un comando dalla palette che apre un altro overlay (es.
  "Mute chat" → confirm dialog) viene gestito così:
  1. `CmdPaletteSubmitMsg` setta `activeOverlay := none` (palette
     unmount).
  2. Lo stesso `Update` ritorna il `tea.Cmd` del comando.
  3. Il `tea.Cmd` (es. `requestConfirmCmd`) ritorna un nuovo
     `tea.Msg` che apre il confirm dialog nel **next** `Update`
     cycle, su `activeOverlay = none`.
  4. La transizione `none → confirm` rispetta il guard di mutex.

L'invariante TLA+ `MUTEX_OVERLAYS` (in `whichkey.tla`) garantisce che
**al più uno** degli overlay {palette, whichKey, help, "other"} è
attivo in ogni stato raggiungibile.

**Eccezione documentata** (già discussa in
[ADR-014](ADR-014-inline-search-bar-vs-modal.md) §D3): la barra inline
`SearchInChat` di Step 27 NON è un overlay (è sub-state della
`ConversationModel`). Le keystroke globali `Ctrl+P` / `?` continuano a
passare al root anche con la barra aperta. Quando si apre un overlay
da quella situazione, il lock `activeOverlay` è acquisito normalmente;
la barra resta in background ed è ripristinata alla chiusura overlay.

#### Perché no stack?

| Aspetto | Mutex (scelto) | Stack |
|---------|----------------|-------|
| Mental model | semplice ("un overlay alla volta") | confusing (Esc chiude top o tutto?) |
| Real estate TUI | adeguato a 80×24 | pessimo (overlay sopra overlay nasconde dati) |
| State complexity | `OverlayKind` enum + guard | `[]OverlayKind` + ordering rules + per-overlay focus |
| Esc semantics | sempre uniforme: chiudi e basta | ambigua: chiudi top? clear stack? |
| Pattern di altri TUI | universale (helix, vim, ranger, lf) | raro (mai visto in TUI Telegram clients) |
| Caso d'uso reale | palette → comando → overlay è gestito atomicamente via D3 step 1-4 | scenari di "stack" non emergono nell'UX di Step 28 |

### D4 — Fuzzy subsequence in-house, no dipendenze esterne

Il filtering della palette usa un **subsequence matcher
case-insensitive** scritto in Go (~30 righe), con scoring:

- +1 per ogni char matched
- +5 + streak per match consecutivo (bonus crescente)
- +10 per match a inizio parola (boundary detection)
- -1/4 di `(len(title) - len(query))` (penalità lunghezza)

Sort: `score DESC, then title ASC`. Algoritmo dettagliato e pseudocodice
in [`../phase-2-behavioral/command-palette-help-whichkey.md`](../phase-2-behavioral/command-palette-help-whichkey.md)
§"Algoritmo fuzzy match (subsequence)".

**Confronto con alternative**:

| Algoritmo | Pro | Contro |
|-----------|-----|--------|
| Substring `Contains` | trivialissimo | non tollerante a typo, query "fwd" non matcha "Forward" |
| **Subsequence in-house (scelto)** | ~30 LOC, zero deps, sufficiente per ~30 comandi, allineato a Step 21 forward picker (ADR-006) | scoring meno raffinato di fzf |
| `sahilm/fuzzy` (libreria) | algoritmo battle-tested | nuova dependency, +500KB binario, overkill |
| `junegunn/fzf` style | best-in-class ranking | richiede porting C-Go, complessità sproporzionata |
| Bleve / Tantivy index | full-text index | enorme overkill, latency di build index inutile |

**Nota di consistency**: il forward picker (Step 21,
[ADR-006](ADR-006-forward-fuzzy-algorithm.md)) usa già un fuzzy
subsequence simile. Step 28 può **riusare lo stesso matcher**
(estraendolo in `internal/ui/components/fuzzy.go` se non già fatto)
per evitare duplicazione e mantenere consistenza percepita
("fuzzy si comporta allo stesso modo in palette e in forward picker").

### D5 — Window di debounce which-key: **300ms**

Decidiamo **300ms**. Razionale:

- È la sweet spot empirica documentata in studi UX di chord input
  (vedi §Contesto).
- Allineato con [ADR-013](ADR-013-search-debounce-and-stale-results.md)
  (search debounce 300ms): consistenza con altre tempistiche del
  prodotto. La regola euristica per tuilegram: **300ms è la "human
  perceptual debounce" standard** — search, which-key, e (futuro)
  altre tempistiche di pre-action ne ereditano.
- Implementato via `tea.Tick(300 * time.Millisecond)` con freshness
  scheme `latestPrefixID` (monotonic counter + drop-stale) — pattern
  identico a search, vedi
  [`../phase-4-concurrency/whichkey.tla`](../phase-4-concurrency/whichkey.tla).

**Configurabilità**: il valore è una `const time.Duration` in
`internal/ui/components/whichkey.go` (compile-time). Se in futuro
emerge la richiesta di renderla configurabile via
`~/.config/tuilegram/config.toml` (sezione `[ui.whichkey]`), il
refactor è triviale: la const diventa un campo di `App.config`. Out
of scope Step 28.

### D6 (sub-decisione) — Best-effort re-dispatch su unknown key

Quando un utente preme un prefix (es. `g`) e poi una key non
registrata in `continuations[g]` (es. `x`), il behaviour è:

1. **Cancel del prefix**: bump `latestPrefixID`, `state := Idle`,
   `activeOverlay := none` (se era visible).
2. **Re-dispatch best-effort**: la key `x` viene ri-route-ata al root
   handler come se fosse stata premuta standalone.
3. Se `x` ha un'azione globale binding standalone, l'azione viene
   eseguita. Altrimenti no-op silenzioso.

Razionale:

- **UX vim-aligned**: vim cancella il prefix su key invalida e
  re-dispatcha (pattern di "operator + motion").
- **Tolleranza al typo**: utente vuole `gG` (bottom) ma scivola su `x`
  (vicino sulla qwerty) — meglio cancellare il prefix che restare in
  hung state.
- **Best-effort, NON garantito**: il contratto formale (in
  `command-palette-help-whichkey.md` §Invarianti) dice solo che il
  prefix viene cancellato. Il re-dispatch è advisory; se la key
  unknown è anche unbound globale, è no-op. Documentato in
  [`../phase-3-interactions/whichkey-timing-flow.md`](../phase-3-interactions/whichkey-timing-flow.md)
  §5.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **D1+D2+D3+D4+D5+D6 (scelta)**: Modal+compact / registry statico / mutex / fuzzy subsequence in-house / 300ms debounce / best-effort re-dispatch | Coerente con feedback_modal_charm.md, zero deps, semplice, sweet-spot UX, allineato a vim/helix; pattern formale verificato in TLA+ | Tre overlay introdotti insieme = più cose da rivedere; `compact` flag su Modal estende l'API |
| Componente `MiniOverlay` separato per which-key | Semantica "this is a different kind of overlay" più esplicita | Viola feedback_modal_charm.md ("non duplicare primitive"); component sprawl |
| Registry runtime (file TOML / extension API) | Configurabile da utente, plugin-friendly | Overkill per binary mono-statico; richiede parser/validator; rischi di runtime panic; out-of-scope tuilegram |
| Stack di overlay | Più potente, può supportare flow complessi | Complessità ingiustificata; UX confusing in 80×24; nessun caso d'uso reale in Step 28 |
| Substring matching (palette) | Triviale, no dependency | Troppo restrittivo: query "fwd" non matcha "Forward chat" |
| `sahilm/fuzzy` library | Algoritmo state-of-the-art | Overkill per ~30 comandi; nuova dependency |
| Debounce 200ms | Più snappy | False reveals per chord eseguiti a velocità normale (utente medio) |
| Debounce 500ms | Più tolerante | Laggy percepito; utente non capisce perché l'overlay tarda |
| Configurable per-user debounce | Massima flessibilità | Out-of-scope MVP; aggiunge superficie di config; nessun pain point reale |
| Strict cancel (no re-dispatch) su unknown key | Più "puro" formalmente | Worse UX: utente preme `gx` per typo → niente succede invece di esegure `x` standalone |
| Auto-execute al timeout (es. `g` da solo → top dopo 300ms) | "Smart default" | Dangerous: utente che esita può triggerare azione inattesa; principio least-surprise violato |

## Conseguenze

- **Positive**:
  - **Coerenza visuale**: tutti gli overlay (palette, which-key, help, +
    overlay esistenti edit/forward/confirm/search/chatInfo) usano la
    `Modal` primitive. `feedback_modal_charm.md` rispettata.
  - **Zero nuove dipendenze**: fuzzy matcher in-house ~30 LOC, riusabile
    da forward picker (ADR-006) → consolidamento.
  - **Modello formale verificato in TLA+**: `whichkey.tla` modella le
    interazioni a tre produttori (keystroke, tick, altri overlay) e
    verifica `MUTEX_OVERLAYS`, `STALE_TICK_BENIGN_WHICHKEY`,
    `FAST_CHORD_NO_OVERLAY`, `EVENTUAL_RESOLUTION` in ~10⁴ stati,
    esecuzione TLC <5s.
  - **Sweet-spot UX (300ms)** allineato a studi empirici e a
    [ADR-013](ADR-013-search-debounce-and-stale-results.md): consistenza
    "perceptual debounce 300ms" come pattern del prodotto.
  - **Type-safe registry**: refactor / rinomina / add command sono
    safe via tooling Go (gopls, gorename).
  - **Pattern riusabile**: il monotonic-counter + drop-stale
    (`latestPrefixID`) è ora applicato in **tre punti** del codebase
    (typing TTL via timestamp, search debounce, which-key debounce),
    consolidando il pattern come canonical per "human-perceptual
    debounce + race-free freshness".
- **Negative**:
  - **Nuovo flag `compact: bool` sulla `Modal` primitive**: un
    parametro in più, ma giustificato dal layout requirements. Non
    rompe Step 26 (default `compact: false` = behaviour attuale).
  - **`activeOverlay` enum cresce di 3 valori** (palette, whichKey,
    help) + 1 implicit "other" placeholder per overlay esistenti.
    Aggiunge superficie minima al root model.
  - **Mutex impedisce flow stack-like**: se in futuro emerge un caso
    d'uso reale ("dalla palette voglio aprire un sub-menu"), va
    rifattorizzato in stack o in cascading-modals. Mitigato dal fatto
    che D3 step 1-4 (palette → tea.Cmd → next overlay) gestisce il
    caso più comune senza stack.
  - **Tre nuovi overlay introdotti in un unico Step 28**: reviewer
    dovrà valutarli tutti insieme. Mitigato dal fatto che condividono
    primitive (D1) e pattern (D5), quindi review è
    decompositiva-per-decisione.
- **Rischi**:
  - **300ms può essere percepito troppo lungo** da power user advanced.
    Mitigazione: opzione futura di config (out-of-scope Step 28). Se
    user feedback dopo Step 28 è negativo, refactor è triviale.
  - **Best-effort re-dispatch (D6) può confondere**: utente preme `gx`
    e vede una azione random eseguita. Mitigazione: documentazione nel
    help overlay (sezione "Prefix keys" elenca prefixe + continuations
    valide, riducendo la prob di typo); behaviour conforme a vim
    (utenti vim non saranno sorpresi).
  - **Race tick + chord (TLA+ T4)**: in caso di timing perfetto, può
    apparire un "flash" dell'overlay per un singolo frame (~16ms).
    Mitigato dalla serializzazione bubbletea + frame timing; user
    impact ~zero. Documentato in
    [`../phase-3-interactions/whichkey-timing-flow.md`](../phase-3-interactions/whichkey-timing-flow.md)
    §4.
  - **Fuzzy subsequence può rankare male in edge case**: query "form"
    rispetto a comandi "Forward...", "Format...", "Information" può
    dare ordering controintuitivo per utenti che si aspettano
    ranking-by-relevance fzf-like. Mitigato dal numero limitato di
    comandi (~30); se diventa pain point, switch a libreria fzf-style
    è isolato (ADR futuro).
  - **Comandi handler che dipendono da contesto (es. "Mute current
    chat" richiede `currentChatID`)**: se il contesto è invalido (no
    chat selected), l'handler deve fallire gracefully (no-op +
    status-bar message, no panic). Decisione operativa, non ADR;
    tracciata come responsabilità dell'implementer di Step 28
    (`tui-architect` con review `code-reviewer`).

## Scope

Questa ADR si applica a:

- **Step 28 — Command palette + which-key + help** (prima e
  contemporanea applicazione delle cinque sottodecisioni).
- Step futuri che introducono **nuovi overlay UI-only**: ereditano
  D1 (Modal primitive con flag layout), D3 (mutex `activeOverlay`).
- Step futuri che introducono **nuovi prefix-key chord** (es. `,`
  per leader key custom): ereditano D5 (300ms debounce) e D6
  (best-effort re-dispatch). Estendono il `Continuations` registry
  (D2) compile-time.
- Step futuri che introducono **registries di azioni o configurazioni
  enumerate** (es. theme picker, account switcher): ereditano D4
  (fuzzy subsequence in-house riusabile) e D2 (registry statico).

**Non si applica a**:

- Overlay che richiedono RPC (search globale Step 26): rimangono
  sotto [ADR-013](ADR-013-search-debounce-and-stale-results.md) per
  semantica `Esc-during-RPC`. Si collocano comunque sotto la mutex D3
  (un solo overlay alla volta).
- La barra inline `SearchInChat` (Step 27,
  [ADR-014](ADR-014-inline-search-bar-vs-modal.md)): NON è un overlay,
  NON usa la primitive `Modal`, NON partecipa al lock D3. Continua
  a coesistere con i tre overlay nuovi senza interferenza.

## Cross-links

- [`phase-2-behavioral/command-palette-help-whichkey.md`](../phase-2-behavioral/command-palette-help-whichkey.md) §Statechart, §Invarianti
- [`phase-3-interactions/whichkey-timing-flow.md`](../phase-3-interactions/whichkey-timing-flow.md) §1 (fast chord), §2 (slow chord), §4 (race), §5 (unknown key)
- [`phase-4-concurrency/whichkey.tla`](../phase-4-concurrency/whichkey.tla) — invarianti `MUTEX_OVERLAYS`, `STALE_TICK_BENIGN_WHICHKEY`, `FAST_CHORD_NO_OVERLAY`, proprietà `EVENTUAL_RESOLUTION`
- [`phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md) §Internal UI Messages (in aggiornamento per `CmdPalette*`/`WhichKey*`/`Help*`)
- [`phase-2-behavioral/ui-statechart.md`](../phase-2-behavioral/ui-statechart.md) §Overlay State Machine (in aggiornamento)
- Pipeline Step 28
- [ADR-006](ADR-006-forward-fuzzy-algorithm.md) — fuzzy subsequence per forward picker (riusato qui)
- [ADR-013](ADR-013-search-debounce-and-stale-results.md) — pattern monotonic counter + drop-stale (riusato qui per which-key timer)
- [ADR-010](ADR-010-typing-ttl-strategy.md) — pattern timestamp + re-arm (precursore concettuale)
- [ADR-014](ADR-014-inline-search-bar-vs-modal.md) §D3 — eccezione SearchInChat al lock di mutua esclusione
- Memoria utente: `feedback_modal_charm.md` (Modal primitive riusata; estesa con flag `compact`)
