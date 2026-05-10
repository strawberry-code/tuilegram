# ADR-018: Responsive layout — threshold (100 cols), hysteresis, Tab semantics in Wide mode, layout side-effects

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 30 introduce il **layout responsive**. Il root model acquisisce
una nuova dimensione di stato `layoutMode ∈ {Wide, Compact}` derivata
deterministicamente da `WindowSizeMsg.Width`:

- **Wide**: ≥ 100 colonne — layout multi-pannello (chat list + conversation; con sidebar opzionale ADR-016).
- **Compact**: < 100 colonne — un solo pannello visibile per volta, switch con `Tab`.

Quattro sotto-decisioni vanno prese insieme perché si influenzano:

1. **Threshold value (100)**: dove sta la soglia? Coerente con la docs
   canonical [`tui-design.md`](../tui-design.md) §"Compact Mode (<100 cols)"
   e con [ADR-016 §D5](ADR-016-folder-source-and-filtering.md) (sidebar
   skip in compact). Cambiare il valore qui ha effetto a cascata.

2. **Hysteresis (dead-band)**: con threshold singolo, un terminale che
   oscilla esattamente attorno a 100 cols (es. tmux pane resize a 99
   ↔ 100) può flippare `layoutMode` ad ogni `WindowSizeMsg`. Serve
   una banda morta? E quanto larga?

3. **`Tab` semantics in Wide mode**: in Compact, `Tab` toglie/aggiunge
   pannello visibile. In Wide, `Tab` ha già una semantica preesistente
   (focus cycle ChatList → Messages → Input, vedi [`ui-statechart.md` §"Focus State Machine"](../phase-2-behavioral/ui-statechart.md#focus-state-machine)).
   Cosa fa `Tab` in Wide? Tre opzioni:
   - **(a) Riusa la semantica esistente di Wide** (focus cycle), nessun
     cambio rispetto agli step precedenti.
   - **(b)** `Tab` mostra/nasconde un pannello anche in Wide (no-op
     visivo perché entrambi sempre visibili).
   - **(c)** Stessa semantica in Wide e Compact, ma significato diverso.

4. **Side-effect su stato co-dipendente** al cross-threshold (resize
   da Wide → Compact o viceversa):
   - `activePanel` cosa diventa al collapse?
   - `folderSidebarVisible` (ADR-016): se `TRUE` in Wide, cosa succede
     scendendo sotto 100? La sidebar è auto-chiusa?
   - Overlay attivo (palette, info): si chiude o si rilayouta?

Reference design:

- **Tmux** / **i3wm**: nessun threshold concept; il pannello si
  ridimensiona "fluido", il contenuto si adatta o si tronca.
- **Telegram Desktop**: threshold ~600 px (proporzionale, non exact);
  sotto soglia, single-pane con back navigation.
- **Slack TUI / weechat**: nessun "compact" mode esplicito; pannelli
  sempre visibili, con scroll orizzontale o troncamento.
- **Helix / lazygit**: layout responsive con threshold hard-coded
  (Helix ha "soft-wrap" + minimal panes); lazygit usa "main view +
  side view" e in narrow window collassa il side view.

## Decisione

**Quadruplice decisione consolidata in una sola ADR.**

### D1 — Threshold = 100 colonne, hard-coded, riferimento canonical

`layoutMode = Wide` ⟺ `width >= 100`. `layoutMode = Compact` ⟺ `width < 100`.

Razionale:

- **Coerenza con canon esistente**: [`tui-design.md`](../tui-design.md)
  §"Compact Mode (<100 cols)" e [ADR-016 §D5](ADR-016-folder-source-and-filtering.md)
  già usano 100 come soglia. Cambiarla qui creerebbe drift con
  artefatti precedenti.
- **Empirico**: a 100 cols il layout 2-pannelli con
  `chatlist_w ≈ 25 cols` + `conv_w ≈ 70 cols` (separator + bordi
  inclusi) è leggibile. A 99 cols, `conv_w ≈ 69` provoca wrap
  aggressivo dei messaggi outgoing (allineati a destra, larghezza
  visiva minore) → degrado UX percepito.
- **Step 31 (theming + config)** può rendere il threshold parametrico
  via `config.toml [display] compact_threshold = 100`. ADR-018
  vincolante per Step 30; futuro override via config è additive.
- **Non si applica al campo Height**: il threshold è solo orizzontale.
  L'altezza minima per usabilità (≥ 20 righe) è gestita altrove
  (rendering "terminal too small" se `< 20`, out-of-scope Step 30).

**Limitazioni accettate**:

- 100 non considera la presenza della sidebar. Quando
  `folderSidebarVisible = TRUE` la chat list è già più stretta;
  scendere sotto 100 con sidebar aperta degrada peggio. Mitigato
  da D4 sotto.
- 100 è arbitrario per terminali esotici (es. terminale con caratteri
  CJK width=2). Modello di width = "colonne reportate da
  `WindowSizeMsg.Width`", che bubbletea misura in celle del terminale,
  non in code points. Coerente con tutto il resto della codebase.

### D2 — Hysteresis: nessuna dead-band (single threshold)

Il flip Wide↔Compact avviene **al primo `WindowSizeMsg` che attraversa
la soglia 100**. Nessuna dead-band, nessun timeout, nessun debounce.

Razionale:

- **Sorgenti di flicker improbabili in pratica**: l'utente non
  ridimensiona il terminale "tremante" attorno a 100; l'oscillazione
  reale viene da resize manuali (passi discreti di 1+ colonne). Il
  rate di `WindowSizeMsg` da resize manuale è ≤ 10/s tipico.
- **Bubbletea già debouncia rendering**: il re-render avviene una
  volta per `Update` cycle, non per ogni `WindowSizeMsg`. Anche con
  due flip consecutivi rapidi, l'utente vede al massimo un frame
  intermedio.
- **Hysteresis aggiunge complessità senza valore**: una dead-band
  (es. Wide se `width >= 102`, Compact se `width < 98`) introduce
  uno stato "indeciso" tra 98 e 102 dove `layoutMode` dipende dalla
  storia. Più difficile da modellare in TLA+, più difficile da
  testare manualmente, più difficile da documentare per l'utente.
- **YAGNI**: se in pratica si osserva flicker (raro), l'aggiunta di
  hysteresis è ~3 righe in `App.Update`. Non c'è motivo di pre-pagare
  la complessità.
- **Coerenza con il pattern "atomic transition"** già usato in:
  `searchInChat` (Step 27, transizione sync), `cmdPalette` (Step 28,
  no debounce sull'apertura), `folderSelect` (Step 29, sync re-filter).

**Conseguenza modellata**: la transizione `LayoutModeChanged` è
**funzione totale** di `width`. `f(width) = if width < 100 then Compact else Wide`.
Idempotente: ricevere due `WindowSizeMsg{width=80}` in fila non muta
nulla (no-op sul second).

### D3 — `Tab` semantics: identica in Wide e Compact, dispatch context-aware

Decisione: **opzione (a)**, `Tab` mantiene la sua semantica preesistente
in Wide (focus cycle ChatList → Messages → Input → ChatList). In
Compact, `Tab` aggiunge una semantica **aggiuntiva** non sostitutiva:
toggle del pannello visibile (ChatList ↔ Conversation).

Razionale:

- **Compatibilità retroattiva**: l'utente abituato a Wide non vede
  cambi in Wide. Tutta la "Tab navigation smart" documentata in
  [`tui-design.md`](../tui-design.md) §"Focus Navigation (Tab)" e
  [`ui-statechart.md` §"Focus State Machine"](../phase-2-behavioral/ui-statechart.md)
  resta valida.
- **In Compact, `Tab` ha un nuovo significato preminente**: switchare
  pannello visibile è l'unica navigazione cross-panel possibile (in
  Wide entrambi sono visibili → `Tab` cambia solo focus). Coerente
  con il pattern Telegram Desktop "back/forward navigation" su mobile.
- **Niente modalità separate**: il dispatcher `App.Update` esegue il
  branch su `layoutMode`:
  - Wide: dispatch `Tab` come `FocusNextMsg` (pre-esistente).
  - Compact: dispatch `Tab` come `LayoutPanelSwitchMsg` (nuovo Step 30).
- **Tab in Compact equivale a Enter+h in sequenza**: aprire una chat
  in Compact + Tab = vedere conversazione, Tab di nuovo = tornare alla
  lista. Niente nuovi gesti da imparare.
- **Esc semantica preservata**: in Compact, dentro
  `Showing Conversation`, `Esc` torna a `ShowingChatList` (stessa
  cosa di `Tab`, parallelo a Wide dove `Esc` esce dal focus
  Messages → ChatList).

**Cosa NON fa `Tab`**:

- **Non cambia `layoutMode`**: `Tab` opera dentro Compact, non flippa
  a Wide. L'unico trigger di flip è `WindowSizeMsg`.
- **Non chiude overlay**: in entrambi i modes, `Tab` è no-op se
  `activeOverlay != none` (l'overlay consuma `Tab` o l'ignora).

### D4 — Side-effect del cross-threshold: collapse-aware ma minimally invasive

Quando `WindowSizeMsg` attraversa la soglia, l'app esegue:

**Wide → Compact (collapse)**:

- `layoutMode := Compact`.
- `compactVisible := PreferConversationOver(activePanel, activeChatID)`:
  - Se `activeChatID != nil` e `activePanel ∈ {Messages, Input}` →
    `compactVisible := Conversation` (preserva l'attenzione corrente
    dell'utente).
  - Altrimenti → `compactVisible := ChatList` (default sicuro).
- `folderSidebarVisible := FALSE` (auto-chiusura forzata, vedi
  [ADR-016 §D5](ADR-016-folder-source-and-filtering.md)). `selectedFolderID`
  è preservato.
- `activeOverlay`: invariato. Gli overlay sono floating e si
  ridisegnano alle nuove dimensioni. `chatInfo` overlay con
  `placement: right` può non avere abbastanza spazio: in compact si
  rilayouta a `placement: full` (override automatico, gestito da
  `Modal` primitive — non è una decisione di questo ADR ma una
  conseguenza del rendering Modal).
- `activePanel` (focus): preservato dove possibile, altrimenti reset
  a `ChatList`.

**Compact → Wide (expand)**:

- `layoutMode := Wide`.
- `compactVisible`: scartato (non più rilevante).
- `folderSidebarVisible`: resta `FALSE` (l'utente lo riapre manualmente
  con `F` se vuole). Decisione: NON auto-restore della sidebar.
- `activePanel`: preservato. Se era `ChatList` o `Conversation` resta
  così. Se era `nil` (mai toccato) → `ChatList` di default.
- `activeOverlay`: invariato, rilayout automatico.

Razionale:

- **Wide → Compact**: l'utente sta restringendo (resize down). Il
  comportamento "preserva conversazione se aperta" fa coincidere
  l'aspettativa "non perdere il contesto", coerente con
  `ACTIVE_CHAT_INVARIANT` di ADR-016.
- **Auto-chiusura sidebar in Compact**: necessaria per real estate.
  Equivalente a `F` con `selectedFolderID` preservato — riapre
  alla stessa folder al expand+manual `F`. Coerente con
  [ADR-016 §D5](ADR-016-folder-source-and-filtering.md) "sidebar
  not available in compact mode".
- **No auto-restore della sidebar al expand**: l'utente che ha
  esperito "compact mode" può non volere la sidebar di nuovo aperta.
  `F` esplicito è il trigger.
- **Overlay invariato**: gli overlay (palette, info, search, edit,
  forward, confirm, help, whichKey) sono floating e indipendenti dal
  layout di base. Forzarne la chiusura su resize sarebbe sorprendente
  ("perché si è chiusa la palette quando ho stretto la finestra?").
  La primitive `Modal` ricalcola dimensioni e placement da
  `WindowSizeMsg` — questa è una capability esistente, non una nuova
  decisione.

**Edge case**: resize così rapido che `compactVisible` non viene mai
osservato (Wide → Compact → Wide in due `WindowSizeMsg` consecutivi
senza render intermedio). Risultato: nessun glitch utente perché tra
i due `Update` cycles non c'è mai stato un render. La transizione
`Compact → Wide` ripristina `activePanel` da preservato.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **D1+D2+D3+D4 (scelta)** | Coerente con canon, no flicker concerns reali, retro-compatibilità Tab Wide, side-effects minimali | Auto-chiusura sidebar al collapse può sorprendere utenti sidebar-heavy (mitigato: `selectedFolderID` preservato) |
| Threshold 80 cols | Standard "VT100"-era; molti terminali partono da 80×24 | Sotto 100 il layout 2-pannelli è già degradato (vedi rationale D1); l'utente userebbe quasi sempre compact. 100 è il "cutoff utile" canonical |
| Threshold 120 cols | Leggibilità ottima in Wide (più whitespace) | Impatta utenti con tmux split (tipica metà-schermo è 80-100 cols). Forzare compact in metà-schermo è UX hostile |
| Hysteresis 5-col dead-band (Wide ≥ 102, Compact < 98) | Elimina flicker teorico al boundary | Stato indeciso tra 98-102 dipende dalla storia; più complesso da modellare/testare; YAGNI (vedi D2) |
| Hysteresis temporale (debounce 200ms su `WindowSizeMsg`) | Filtra resize events rapidi | Latency percettibile durante drag-resize; il re-render non avviene per 200ms = freeze visivo. Peggio che flicker |
| `Tab` in Wide = stesso comportamento di Compact (toggle pannello visibile) | Uniformità Wide/Compact | Rompe la "Tab navigation smart" (Step 6+); regressione UX per utenti Wide |
| `Tab` in Wide = no-op (riservato a Compact) | Semantica chiara "Tab = layout switch" | Rompe completamente la Tab navigation pre-esistente; richiede rebinding di tutta la focus cycle (Shift+Tab? rilegato a F1?) |
| `Tab` in Compact = no-op, switch via Enter/Esc | Coerente con "Esc = back" | `Tab` è il gesto naturale per "next panel" in TUI; non usarlo è anti-pattern |
| Wide → Compact: forza chiusura overlay attivo | UI consistency (overlay sized for Wide può essere broken in Compact) | Sorprendente per l'utente; primitive Modal già rilayouta |
| Wide → Compact: auto-show sidebar in compact-mode (icon-only) | Featureful in compact | Out-of-scope Step 30 (vedi ADR-016); aggiunge stato non necessario |
| Compact → Wide: auto-restore sidebar se era visible prima del collapse | "Memoria" del layout precedente | Richiede una variabile `sidebarPreCollapse: bool` extra; YAGNI; user può `F` |
| `compactVisible` initial = sempre ChatList | Default semplice | Perde l'attenzione utente: chi era in conversazione e ridimensiona vede saltar via la chat. UX hostile |
| `compactVisible` initial = sempre Conversation se `activeChatID != nil` | Coerente "preserva conversazione" | Ma se utente era focus ChatList per scegliere chat, lo butta nella conversation panel sbagliata |
| `compactVisible` = preservato attraverso multipli collapse-expand cycle | "Memoria" UI | Richiede variable persistente anche in Wide mode (sprecata 99% del tempo). YAGNI |

## Conseguenze

- **Positive**:
  - **Coerenza canon**: 100 cols allineato con `tui-design.md` e ADR-016.
    Nessun drift documentale.
  - **Modello deterministico**: `layoutMode = f(width)`, funzione pura.
    Facile da testare, modellare in TLA+, riprodurre.
  - **Retro-compatibilità Tab Wide**: nessun cambio per utenti Wide
    (l'80% dei desktop). I cambiamenti sono additivi (Compact-only).
  - **Side-effect minimali al cross-threshold**: layout cambia, ma
    `selectedFolderID`, `activeChatID`, `activeOverlay`, `compactVisible`
    (preservato attraverso il flow) sono coerenti con i principi
    pre-esistenti (ADR-016 active-chat invariance, ADR-015 mutex overlays).
  - **No flicker concerns reali**: resize manuale è discreto; bubbletea
    debounces rendering naturalmente.
  - **TLA+ compatto**: la spec ha un solo invariante chiave
    (`COMPACT_ONE_PANEL`) + un'idempotenza (`THRESHOLD_DETERMINISTIC`),
    < 100 righe.
- **Negative**:
  - **Sidebar auto-close al collapse**: utente sidebar-heavy può
    sentirlo come "perdita". Mitigato: `selectedFolderID` preservato,
    `F` riapre alla stessa folder.
  - **`Tab` ha due comportamenti** (focus cycle in Wide, layout switch
    in Compact). Onere documentale: deve essere chiaro nell'help (`?`)
    e nei keybinding tables.
  - **No hysteresis**: in scenari di flicker patologico (test E2E?
    Animation di resize?), può apparire una transizione visibile.
    Mitigato: in pratica non si presenta; se diventerà un problema,
    aggiungerla è triviale.
- **Rischi**:
  - **Threshold troppo aggressivo per utenti tmux**: chi splitta
    50/50 un terminale 200-col ha 100 cols esatti = boundary. Mitigato:
    a 100 esatti il layout è ancora Wide (≥ 100); deve scendere sotto
    per flippare.
  - **`compactVisible` derivation non intuitiva**: la regola "preserva
    conversazione se attiva" può sorprendere utenti che si aspettano
    "torna sempre alla chat list". Mitigato: documentato in `?` help.
  - **Side-effect cascade**: cambiare D4 in futuro (es. "auto-restore
    sidebar al expand") richiede ADR successiva e nuovo TLA+ run.
    Mitigato: D4 è restrittivo by design (minimal side-effect).
  - **Step 31 (config)**: dovrà gestire `[display] compact_threshold = 100`
    parametricamente. Mitigato: refactor è 1 riga (`const compactThreshold = 100`
    diventa `compactThreshold := config.Display.CompactThreshold`).

## Scope

Questa ADR si applica a:

- **Step 30 — Responsive layout + compact mode**: prima introduzione
  delle quattro decisioni.
- Step futuri che introducono **side-effect su layout-aware state**
  (es. nuovi pannelli inline, nuovi overlay con `placement` constraints):
  ereditano D4 (cross-threshold side-effect minimali).
- Step 31 (theming + config): può rendere il threshold (D1) parametrico
  via `config.toml`; non muta D2/D3/D4.
- Step 33 (status bar polish): può aggiungere indicatore visivo del
  `layoutMode` corrente (es. "Compact" hint in status bar); non muta
  questa ADR.

**Non si applica a**:

- Threshold di **height** (`tea.WindowSizeMsg.Height`): out-of-scope.
  Una soglia minima di altezza per usabilità (~20 righe) è gestita
  da rendering "terminal too small" generico, fuori da
  `layoutMode`.
- **Mouse resize handling**: gestito a livello bubbletea/terminal,
  non da questa ADR. `WindowSizeMsg` arriva indipendentemente dalla
  sorgente.
- **Editing del threshold a runtime** via comando o keybinding: non
  prevista; se in futuro si aggiunge, è una feature `Settings` di
  Step 31+.

## Cross-links

- [`../phase-2-behavioral/responsive-layout.md`](../phase-2-behavioral/responsive-layout.md) §Statechart, §Invarianti
- [`../phase-2-behavioral/ui-statechart.md`](../phase-2-behavioral/ui-statechart.md) §"Responsive Layout States" (esteso da Step 30)
- [`../phase-3-interactions/responsive-layout-flow.md`](../phase-3-interactions/responsive-layout-flow.md) — sequence diagrams resize + Tab
- [`../phase-4-concurrency/responsive_layout.tla`](../phase-4-concurrency/responsive_layout.tla) — invarianti `COMPACT_ONE_PANEL`, `WIDE_TWO_PANELS`, `THRESHOLD_DETERMINISTIC`, `TAB_PRESERVES_LAYOUT`
- [`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md) §Internal UI Messages (esteso con `LayoutModeChangedMsg`, `LayoutPanelSwitchMsg`)
- Pipeline Step 30
- [ADR-016 §D5](ADR-016-folder-source-and-filtering.md) — sidebar skip in compact mode (ereditato da Step 29)
- [ADR-015 §D3](ADR-015-command-palette-whichkey-help.md) — overlay mutex (rispettato; `Tab` no-op se overlay attivo)
- [`../tui-design.md`](../tui-design.md) §"Compact Mode (<100 cols)", §10 Focus Navigation
