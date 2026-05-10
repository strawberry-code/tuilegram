# ADR-020: Mouse support — central hit-test router, bbox cache, wheel-by-cursor, click-to-open chat, dismissable overlay close, keyboard parity invariant

**Stato**: accettato
**Data**: 2026-05-10

## Contesto

Lo Step 32 della pipeline introduce il **mouse support**: scroll wheel
sul viewport e sulla chat list, click su chat item che apre la
conversazione, click sul bottone SEND. Tutte le features mouse sono
**aggiuntive**: ogni gesto deve avere un equivalente da tastiera già
funzionante (è il primo principio dello Step 32, "Tutto funziona anche
senza mouse — keyboard-only").

Stato pre-Step 32:

- `cmd/tuilegram/main.go:56` già attiva `tea.WithMouseCellMotion()`. Il
  runtime bubbletea spedisce `tea.MouseMsg` al root model. Sono in
  larga parte **non gestiti** (cadono nel default branch di
  `MainModel.Update`); l'unico consumer attivo è
  `ChatListModel.handleMouse` (`internal/ui/views/chatlist_nav.go:63`)
  che muove `selected` su wheel up/down — ma **solo se** il `MouseMsg`
  riesce a raggiungere il sub-model, cosa che oggi avviene
  accidentalmente nel branch finale di `MainModel.Update`
  (`m.chatList.Update(msg)` come fallback). Il comportamento è
  **incidentale**, non progettato:
  - Wheel su qualsiasi punto dello schermo finisce a `chatList`,
    indipendentemente da dove sia il cursore.
  - Wheel sul viewport conversation **non** scrolla i messaggi.
  - Click sinistro è ignorato dovunque.
  - Click su SEND è ignorato (il bottone è solo decorativo).
- Layout responsive (Step 30, ADR-018) introduce `layoutMode ∈ {Wide,
  Compact}`. In Compact un solo pannello è visibile; click sui pannelli
  nascosti devono essere ignorati.
- Overlay mutex (Step 28, ADR-015) garantisce `≤ 1` overlay attivo. Click
  fuori da un overlay attivo: dismiss o no-op? Dipende dal tipo (modal
  vs dismissable). Va deciso.
- Bubbletea v1.3.10 espone un **singolo** tipo `tea.MouseMsg = MouseEvent`
  con campi `{X, Y, Shift, Alt, Ctrl, Action, Button}` dove
  `Action ∈ {Press, Release, Motion}` e `Button ∈ {None, Left, Middle,
  Right, WheelUp, WheelDown, WheelLeft, WheelRight, Backward, Forward, ...}`
  più helper `IsWheel() bool`. Non esistono in questa versione tipi
  separati `MouseClickMsg` / `MouseWheelMsg`: il discriminante è il
  campo `Button` + `Action`. Riferimento documentato.

Cinque sotto-decisioni vanno prese insieme perché co-determinano il
modello operativo:

1. **Strategia di hit-test**: chi mappa (X, Y) → widget? Centrale
   (router in `MainModel`) vs decentralizzata (ogni sub-model
   filtra per conto suo) vs ibrida.
2. **Caching dei bounding box (bbox)**: ricalcolati ad ogni
   `WindowSizeMsg` (e cached) o derivati on-the-fly per ogni click?
3. **Routing dei wheel events**: per cursor position (cursore sopra
   il widget) o per focus corrente?
4. **Semantica click su chat item**: select-only (1 click muove
   il cursore, 2° click apre) vs select+open atomico (1 click
   apre come fosse Enter)?
5. **Click fuori da overlay**: chiude l'overlay (dismiss) o no-op
   (modal-strict)? La risposta dipende dal tipo di overlay.

Plus tre cross-cutting:

6. **Click su pannello non focused**: focus shift only (1° click) o
   focus + azione (1° click già attiva)?
7. **Drag & text selection nativa del terminale**: in scope o
   deferred?
8. **Keyboard parity**: deve essere un invariante di progetto?

Reference design (mouse handling in TUI apps):

- **Helix editor**: hit-test centralizzato (router in `compositor`),
  bbox cache, wheel routes by cursor position, click su buffer ⇒
  cursor movement (1 click). Drag = native selection (mouse capture
  tipicamente OFF).
- **lazygit**: bbox map per pannello calcolato in `layout()`, hit-test
  centrale in `mouseHandler`, click+open atomico su list items, wheel
  by focus (lazygit usa focus, non cursor — UX criticato).
- **Crush (Charm)**: hit-test ibrido, ogni componente espone
  `WithinBounds(x, y) bool`, root model itera in z-order e dispatcha
  al primo match. Wheel by cursor position. Drag deferred (solo nei
  componenti specifici, es. textinput).
- **Telegram Desktop**: GUI nativa, click+open atomico, drag = text
  selection. Non confrontabile direttamente per pattern, ma il
  comportamento user-facing (single click = open chat) è il
  benchmark UX.

## Decisione

**Quintupla decisione consolidata in una sola ADR**, più cinque
decisioni minori cross-cutting (D6..D10).

### D1 — Hit-test: router centrale in `MainModel.handleMouseMsg`

`MainModel` introduce un nuovo handler `handleMouseMsg(msg tea.MouseMsg)`
chiamato dal `MainModel.Update` switch. Il router:

1. Risolve `(msg.X, msg.Y)` contro la **bbox map** corrente (D2).
2. Determina il **target widget** (chatList, conversation viewport,
   conversation input/textarea, sendButton, folderSidebar, status bar,
   overlay-foreground se overlay attivo).
3. Per click (`Action = Press, Button = Left`): dispatcha al sub-model
   l'azione equivalente al keystroke (es. click su chat item ⇒
   `ChatSelectedMsg{chatID}`; click su SEND ⇒ stessa logica di Enter
   nell'input).
4. Per wheel (`IsWheel() = true`): forwarda `tea.MouseMsg` originale
   al sub-model che possiede il widget sotto il cursore (D3).
5. Per release/motion (`Action ∈ {Release, Motion}`): in Step 32
   **scartati** (D7 — no drag).

**Razionale**:

- **Single source of truth**: la mappa widget→bbox vive solo in
  `MainModel` (che è anche il proprietario del layout). Sub-models non
  conoscono la propria posizione assoluta.
- **Coerenza con la responsabilità di layout**: il calcolo dimensionale
  (`SetSize`/`setWideSize`/`applyCompactSizes`) è già in `MainModel`;
  derivare le bbox lì è naturale (zero dati duplicati).
- **Z-order esplicito**: il router decide in che ordine consultare gli
  overlay (ADR-015 mutex semplifica: 0 o 1 overlay). Fare lo stesso in
  modo decentralizzato richiederebbe protocolli di "consume"/"propagate"
  tra sub-models, costoso.
- **Test scrivibili in unità**: `handleMouseMsg(MouseMsg{X:50, Y:10})`
  con bbox map fissa è una pure function testabile senza render.

**Limitazioni accettate**:

- Tutti i sub-model devono esporre **API per ricevere il click come
  azione semantica** (non come `MouseMsg` raw). Esempio: `ChatListModel`
  espone (concettualmente) `SelectIndex(i int)`. Il router fa il
  reverse-lookup `(X, Y) → chat index → chiama SelectIndex`. Sub-models
  restano UI-position-agnostic.
- Wheel events forwardati come `tea.MouseMsg` raw (sub-model decide
  cosa scrollare). Asimmetria click/wheel motivata: per wheel il
  "delta" è sempre semplice (up/down N units), non vale la pena di
  aggiungere `WheelDeltaMsg` semantici. Per click invece l'azione è
  ricca (open chat vs open chat info vs send) e va tipizzata.

### D2 — Bbox cache: store + invalidate on `WindowSizeMsg`

`MainModel` mantiene una struttura interna `bboxes` (privata, non
esposta a sub-model) che memorizza i bounding box di ogni widget
"clickable" sotto forma `{x0, y0, x1, y1}` (inclusive low, exclusive
high — convenzione standard).

Schema:

```
bboxes ::= {
    chatList         : Rect | nil
    conversationView : Rect | nil   // viewport messaggi (no header, no input)
    conversationHdr  : Rect | nil   // header chat (Step 14)
    inputArea        : Rect | nil   // textarea + sendButton
    sendButton       : Rect | nil   // sub-rect dentro inputArea
    folderSidebar    : Rect | nil   // solo se folderSidebarVisible
    statusBar        : Rect | nil
}
```

Update protocol:

- Trigger primario: `tea.WindowSizeMsg` (ricalcolo dimensionale).
- Trigger secondari (state UI che muta layout senza resize):
  - `LayoutModeChangedMsg` (ADR-018): cross-threshold cambia il numero
    di pannelli visibili.
  - `LayoutPanelSwitchMsg` (ADR-018): in Compact, cambia il pannello
    visibile.
  - `FolderToggleMsg` (ADR-016): apre/chiude la sidebar in Wide.
- Invalidazione: tutti gli handler che mutano layout chiamano
  `recomputeBboxes()` come ultimo step (idempotente).
- In Compact: i pannelli **nascosti** hanno bbox `nil`. Hit-test su
  `nil` ⇒ no match ⇒ click ignorato (D9 NO_HIDDEN_CLICK).
- Overlay attivo: il router consulta prima `overlayBbox(activeOverlay)`
  (D5 OVERLAY_FIRST).

**Razionale**:

- **Performance**: una hit-test sul click path costa O(numero widget)
  ≈ O(7). Ricalcolare le bbox per ogni click è uguale O(7) ma duplica
  logica già fatta in `SetSize`. Cache rende l'hit-test O(7) deterministico.
- **Determinismo**: bbox calcolate **una volta sola** per ogni
  configurazione di layout. Niente race tra "render" e "click handling".
- **Testabilità**: una test suite può forzare `m.bboxes = {...}` e
  verificare il dispatch isolatamente dal render.
- **Invalidazione semplice**: il set di trigger è chiuso e già
  documentato (resize + LayoutModeChanged + LayoutPanelSwitch +
  FolderToggle). Non c'è una "fonte fantasma" di mutazione del layout.

**Alternativa scartata**: derivare bbox on-the-fly dentro
`handleMouseMsg` chiamando `setWideSize`/`applyCompactSizes` "fittizi"
ad ogni click. Pro: zero stato extra. Contro: duplica i calcoli, fa
ri-eseguire codice di layout per ogni evento mouse (che può arrivare
a 60+/s su drag). Il drag non è in scope (D7) ma il principio resta.

### D3 — Wheel events: route by cursor position (NOT by focus)

`tea.MouseMsg` con `IsWheel() = true` viene risolto contro la bbox map
**posizionalmente**: il widget sotto il cursore riceve l'evento, **a
prescindere dal focus corrente** (`activePanel`).

Es.: `activePanel = Input` (textarea ha il focus); l'utente sposta il
cursore sopra il viewport conversation e scrolla. Il viewport scrolla
(NON l'input — dove tra l'altro lo scroll non avrebbe senso).

**Razionale**:

- **UX matching the pointer**: gli utenti si aspettano che la rotella
  agisca sull'oggetto sotto il cursore. È il pattern di Crush, di Helix,
  di tutte le GUI desktop, di tmux mouse mode.
- **Coerenza con D1**: il router già conosce le bbox; usare la stessa
  mappa per wheel risolution è zero codice extra.
- **Riduce gestures spreche**: scrollare con il cursore "lontano"
  sarebbe un'azione senza target visibile. Better UX se è esplicito.
- **No conflitto con focus**: `activePanel` resta usato dal **keyboard
  flow** (`j/k`, scroll-by-key). Il cursore è una secondary navigation
  axis, non muta il focus.

**Conseguenza modellata**: invariante `WHEEL_BY_POSITION` nello
statechart e in [`mouse-routing.md`](../phase-2-behavioral/mouse-routing.md).

**Edge case**: cursore sul border/gap tra due widget (es. coordinata
tra chatList e conversation, sulla linea verticale). Decisione: il
router usa **half-open intervals** (`x0 ≤ x < x1`); la coordinata di
border è del widget a sinistra/sopra. Deterministico, nessun "buco".

**Edge case 2**: cursore fuori da ogni bbox (es. status bar, righe
vuote). Decisione: wheel **no-op silenzioso**. Non è un errore.

### D4 — Click su chat item: select + open (atomico)

Click sinistro (`Action = Press, Button = Left`) su un chat item
emette **lo stesso effetto di `Enter` su quel chat**:

1. Aggiorna `chatList.selected := i` (i = indice della chat colpita).
2. Emette `ChatSelectedMsg{chatID}` (esistente, Step 11+).
3. In Compact: il `MainModel.handleEnter` esistente già fa il flip
   `compactVisible := CompactConversation` come parte dell'apertura
   chat (Step 30). Click eredita lo stesso path.

**Razionale**:

- **Aspettativa GUI**: in Telegram Desktop, in Slack, in Discord, un
  click su un chat item lo apre. "Select then open" è un pattern di
  liste-non-azionabili (file manager con doppio click); nelle chat-app
  è single-click open.
- **Niente double-click**: i terminali non hanno una nozione affidabile
  di "double click" (il timing si rileva ma è inconsistente cross-OS).
  Evitiamo la categoria.
- **Convergenza con keyboard**: `Enter` su selected chat = open.
  Click = "selected becomes the clicked chat" + "Enter implicito".
  Stessa transizione finale.
- **Compact mode UX**: in Compact, click apre la chat E switcha al
  panel Conversation. Coerente con `Enter` keyboard (Step 30, scenario
  3 in `responsive-layout-flow.md`).

**Limitazione accettata**: l'utente che vuole "preview" la chat (vedere
in lista chi è) senza aprirla deve usare `j/k` (keyboard). Click =
commit. Acceptable: in Telegram Desktop si fa così; non è un workflow
mouse-driven comune.

### D5 — Click fuori da overlay attivo: dismissable vs modal

Tassonomia degli overlay:

| Overlay | Tipo | Click outside |
|---------|------|---------------|
| `cmdPalette` (Step 28) | dismissable | **chiude** (eq. `Esc`) |
| `help` (Step 28) | dismissable | **chiude** (eq. `Esc`) |
| `whichKey` (Step 28) | dismissable | **chiude** (eq. `Esc`/cancel) |
| `searchOverlay` (Step 26) | dismissable | **chiude** (eq. `Esc`) |
| `chatInfo` (Step 29) | dismissable | **chiude** (eq. `Esc`/`i`) |
| `forwardPicker` (Step 21) | **modal** | **no-op** (richiede Enter o `Esc` espliciti — ADR-007 in-flight RPC, ADR-008 batch semantics) |
| `editOverlay` (Step 19) | **modal** | **no-op** (rischio di perdere il testo editato) |
| `confirmDelete` (Step 20) | **modal** | **no-op** (azione distruttiva — richiede conferma esplicita Y/N) |

**Razionale**:

- **Dismissable (palette, help, whichKey, search, chatInfo)**: scopo
  informativo o di selezione semplice. Click fuori = "non mi serve
  più" (intent chiaro). Replica il pattern di
  Spotlight/Alfred/cmd-palette di VSCode/Crush.
- **Modal (forward, edit, confirm)**: azioni con stato editabile o
  effetti irreversibili. Click fuori sarebbe un trigger ambiguo
  (chiude e perde il testo? chiude senza eseguire? chiude e esegue?).
  La policy "richiede gesture esplicita Y/N o Enter/Esc" elimina
  l'ambiguità.
- **Coerenza con ADR-007**: per `forwardPicker`, ADR-007 già stabilisce
  che `Esc` durante RPC in volo è gestito esplicitamente (cancel
  request, restore state). Click outside avrebbe stessa semantica
  ma senza il chiaro intent di "annullare" — ambiguo.
- **Coerenza con `editOverlay`**: l'utente che sta modificando un
  messaggio non vuole perdere il testo per un click accidentale fuori.

**Modellato come**: invariante `OVERLAY_FIRST` (overlay vince hit-test
su click) + `DISMISSABLE_OUTSIDE_CLOSES` (click outside su dismissable
emette `OverlayCloseMsg`).

**Edge case**: click *dentro* un overlay non-dismissable. Routato al
sub-model dell'overlay (es. `forwardPicker.Update(MouseMsg)`). In Step
32 la maggior parte di questi sub-models **non gestisce mouse**
internamente: l'effetto sarà no-op silenzioso (l'overlay non si chiude
ma non agisce neanche). Acceptable: il keyboard è la sorgente
canonica per editOverlay/forward/confirm. Future steps possono
aggiungere mouse interno (es. click su voce della lista forward) come
incremento.

### D6 — Click su pannello non-focused: focus + action atomic

Click su un widget di un pannello che **non** è `activePanel` produce
**due effetti atomici** nella stessa `Update` cycle:

1. `activePanel := <panel-of-widget>`. Esempio: click su chat item
   con `activePanel = Input` ⇒ `activePanel := ChatList`.
2. **Esegue l'azione** del click (D4: select + open chat).

NON è "primo click focus, secondo click action" (che è un pattern
GUI). Reasoning:

- **Coerenza con D4**: D4 dice click = open. "Focus first, action second"
  contraddice "single click = open".
- **TUI pattern**: i terminali non hanno la nozione di "window focus"
  separata da "active widget". Click su qualsiasi widget = "ora opero
  qui". Pattern di Helix, lazygit.
- **Riduce friction**: non costringe l'utente a "sprecare" un click
  per spostare il focus.

**Edge case** (composer textarea): click sull'`inputArea` con
`activePanel = ChatList` deve:

1. Switchare focus: `inputFocus := true`, `activePanel := Conversation`.
2. Far entrare in input mode (equivalente di `i` keyboard).
3. NOT inviare nulla: il click sull'area textarea è "place cursor",
   non "submit".

Click sul **bottone SEND** (sub-rect dentro `inputArea`) è invece
"submit": equivalente di Enter su textarea con testo non vuoto. Vedi
D10.

### D7 — Drag / native text selection: DEFERRED (out of Step 32)

`tea.WithMouseCellMotion()` cattura tutti gli eventi mouse del
terminale, **inclusi** drag e motion. Conseguenza: il browser-style
"trascina-per-selezionare-testo" del terminale **non funziona** mentre
l'app è running, perché il terminale non riceve il drag per la propria
selezione (lo riceve l'app via `MouseMsg` con `Action = Motion` e
`Button = Left`).

Step 32 **non** implementa text selection custom dentro l'app. Eventi
con `Action ∈ {Motion, Release}` sono **scartati** dal router (no-op).
Conseguenze:

- L'utente **non può** selezionare testo dei messaggi con il mouse
  (drag) durante l'esecuzione dell'app. Il terminale-bypass (es.
  Cmd+Option+drag su iTerm2/macOS, Shift+drag su molti terminali Linux,
  fn+drag su altri) **rimane attivo** come escape-hatch (gestito a
  livello di emulator, non dall'app).
- L'utente che vuole copiare un messaggio ha due opzioni: (a) usare
  l'escape-hatch del terminale; (b) (futuro) `y` keybinding per yank
  del messaggio selezionato (out of scope Step 32).

**Razionale**:

- **Step 32 è "scroll wheel + click su chat items + click SEND"**.
  Drag/select è una feature significativa (richiede UI di highlight,
  buffer copy, integrazione clipboard) che merita uno step dedicato.
- **L'escape-hatch terminale è universalmente noto**: utenti
  TUI-savvy lo usano già (è il pattern di tmux con mouse mode,
  vim con `set mouse=`).
- **YAGNI**: non aggiungere una feature di selezione custom prima di
  capire cosa l'utente vuole davvero (link click? messaggio yank?
  paragraph select?).

**Conseguenza modellata**: invariante `NO_PHANTOM_DRAG` — eventi con
`Action != Press` non producono mai mutazione di stato in Step 32.

**Switch alternativo** considerato e scartato: usare
`tea.WithMouseAllMotion()` invece di `tea.WithMouseCellMotion()`
oppure dropparla del tutto. Il drop di mouse capture lascerebbe la
selezione nativa del terminale ma **rinuncia a tutto il mouse
support** (incluso click e wheel) — contraddice Step 32.

### D8 — Keyboard parity: hard project-level invariant

Ogni gesto mouse di Step 32 ha già un equivalente keyboard
**funzionante a parità di feature** (non "ne deriva"). La parity è
un'invariante di progetto, modellata come `KEYBOARD_PARITY` nello
statechart:

| Mouse | Keyboard equivalente | Verificato in step |
|-------|----------------------|--------------------|
| Wheel sul viewport conversation | `j`/`k`/`Ctrl+D`/`Ctrl+U` | Step 11+ |
| Wheel sulla chat list | `j`/`k` | Step 7+ |
| Click su chat item | `j`/`k` per cursore + `Enter` | Step 7+ |
| Click su SEND button | `Enter` in textarea con testo | Step 15+ |
| Click su panel non-focused | `Tab` (Wide) o `Tab` (Compact) | Step 6+, Step 30 |
| Click outside dismissable overlay | `Esc` | Step 26+ |

**Implicazioni**:

- **Mai** introdurre una feature mouse-only senza scriverne il
  keyboard equivalent **prima**. Step 32 si limita a wheel/click esistenti
  → l'invariante è automatica.
- Ogni futura feature mouse (es. click su link Step 33, drag-to-select
  futuro) deve documentare l'equivalente keyboard nello stesso ADR.
- Test plan di Step 32 verifica esplicitamente "tutto funziona anche
  senza mouse (keyboard-only)" come ultima riga del test plan
  (`development-pipeline.md` §Step 32).

### D9 — Compact mode: click su pannello nascosto = no-op

In `layoutMode = Compact`, solo `compactVisible ∈ {ChatList, Conversation}`
è renderizzato. Il pannello hidden è **non visibile** sullo schermo,
quindi l'utente non può cliccarci sopra in modo visibile, MA:

- Le coordinate `(X, Y)` di un click sono nel range completo del
  terminale.
- Le bbox dei widget hidden sono `nil` (D2): il router non risolve
  nulla.
- Conseguenza: click in qualsiasi punto durante Compact che non sia
  il pannello visibile + status bar = no-op silenzioso.

**Wheel events**: wheel by cursor position (D3) routa al pannello
visibile se il cursore ci è sopra; altrimenti no-op. Il pannello
hidden non riceve wheel.

**Razionale**:

- **No surprise**: l'utente non clicca su qualcosa di non visibile;
  se accade per errore (es. region è status bar), no-op è il
  comportamento atteso.
- **Coerenza con D2**: la bbox map è la single source of truth.
  Niente "phantom hits" su widget non-renderizzati.
- **Coerenza con ADR-018 §D2** (`COMPACT_ONE_PANEL`): exactly one
  panel rendered. Mouse handling rispetta lo stesso invariante.

**Edge case**: durante una transizione Wide→Compact lo stesso
`Update` cycle ricalcola bbox e poi processa eventuali pending
`MouseMsg`. Bubbletea serializza tutto via single Update channel: non
c'è race tra resize e click.

### D10 — Click sulla composer (input area) e su SEND

Sub-rect del pannello conversation in fondo (Step 15+):

```
┌── conversation viewport (messaggi) ──────────────┐
│ ...                                              │
├── inputArea ─────────────────────────────────────┤
│ textarea  (input message)              ╭──────╮  │
│                                        │ SEND │  │
│                                        ╰──────╯  │
└──────────────────────────────────────────────────┘
```

Bbox: `inputArea` contiene `textarea` (left) e `sendButton` (right).
Click handling:

- **Click su `sendButton` bbox**:
  - Se `textarea.Value() != ""` → emette evento equivalente a `Enter`
    in textarea (`m.appendOptimistic(text)` flow).
  - Se `textarea.Value() == ""` → no-op silenzioso (pulsante
    "disabled" non invia, coerente con `sendBtn.Active = false` UI).
  - In edit mode → analogo a Enter in edit mode (`submitEdit()`).
- **Click su `textarea` bbox** (resto di `inputArea` escluso il
  bottone):
  - `inputFocus := true`, `sendBtn.Active := true`, focus al textarea.
  - **NON** invia. Il click "place cursor" interno alla textarea è
    delegato a `bubbles/textarea` se vuole gestirlo (in Step 32
    forwardiamo il `MouseMsg` al textarea: `bubbles/textarea` v1.0.0
    non gestisce mouse internamente, quindi è effettivamente no-op
    sulla posizione del cursore — è il comportamento accettato).
- **Click su `inputArea` da pannello non-focused** (es. con
  `activePanel = ChatList`):
  - Combina D6 (focus shift) + sopra: `activePanel := Conversation`,
    `inputFocus := true`, eventualmente apre la conversation in
    Compact (`compactVisible := Conversation`).

**Razionale**:

- **Click su SEND = submit** è l'aspettativa universale (è un
  bottone con label "SEND", non c'è ambiguità).
- **Click su textarea = focus** è la convenzione GUI; non vogliamo
  inviare per click (sarebbe accidentale).
- **Bottone disabled**: già modellato da `sendBtn.Active = false`.
  Click no-op coerente con visual feedback.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **D1+D2+D3+D4+D5+D6+D7+D8+D9+D10 (scelta)** | Modello unico, central router, bbox cache deterministica, parity garantita, drag deferred (focus su Step 32 scope), overlay tassonomizzati per dismiss-vs-modal | Sub-models devono esporre azioni semantiche (refactor leggero); requires bbox invalidation discipline (ma il set di trigger è chiuso) |
| **Hit-test decentralizzato (ogni sub-model filtra)** | Ogni sub-model autonomo | Sub-models devono conoscere la propria pos assoluta (cross-cutting concern); duplicazione logica; protocolli "consume/propagate" complessi |
| **Bbox derivate on-the-fly** | Zero stato extra in `MainModel` | Duplica calcoli di `SetSize`; rieseguito ad ogni MouseMsg (più costoso con Motion attiva); harder to test (non c'è snapshot) |
| **Wheel by focus (lazygit-style)** | Coerente con keyboard scroll (entrambi vanno su focus) | UX hostile: scrollare con cursore "lontano" agisce su widget invisibile; contraddice convention GUI |
| **Click su chat item = select-only** | Esplicito due-step | Doppio gesto per aprire = friction; doppio click in TUI è inaffidabile; contraddice pattern Telegram/Slack |
| **Click outside qualsiasi overlay = chiude** | Uniformità | Perde testo editing in editOverlay; chiude forwardPicker che ha RPC in volo (ADR-007 violation); modal non più modal |
| **Click outside qualsiasi overlay = no-op** | Modal-strict ovunque | Sgradevole UX su palette/help (gesture comune); contraddice convenzione macOS Spotlight/cmd-palette |
| **Click su pannello non-focused = focus only (1° click), action su 2° click** | Pattern GUI desktop "explorer" | Contraddice D4 (click = open); doppio gesto inutile in TUI; raddoppia il numero di click per ogni nav |
| **Drag-to-select dentro l'app (custom)** | Permette copy senza terminal escape-hatch | Rich UI (highlight buffer, clipboard integration); out-of-scope Step 32; richiede ADR dedicato |
| **Disable mouse capture (no `tea.WithMouseCellMotion`)** | Native terminal selection funziona | Rinuncia a mouse support → contraddice Step 32 |
| **`tea.WithMouseAllMotion()` invece di `CellMotion`** | Eventi anche su motion senza press | Più traffico (60+/s su drag); zero feature aggiunta in Step 32; YAGNI |
| **Tipizzare `MouseClickMsg` / `MouseWheelMsg` interni separati** | Symmetric con tassonomia message | Extra layer per zero gain; bubbletea v1.3.x non differenzia internamente; complicazione gratuita |
| **Click sul textarea = invia (come Enter)** | Single-click compose-and-send | Distruttivo: click accidentale invia testo; viola convenzione GUI ovunque |

## Conseguenze

- **Positive**:
  - **Modello deterministico**: hit-test = pure function `(MouseMsg, bboxes) → action`.
    Testabile in isolamento, senza render harness.
  - **Bbox cache stabile**: invalidata solo da eventi noti
    (resize + 3 eventi UI). Nessuna fonte fantasma di mutazione.
  - **Wheel-by-position**: UX naturale, coerente con tutti i benchmark
    (Crush, Helix, GUI desktop). Nessuna sorpresa.
  - **Click+open atomic**: matching dell'aspettativa Telegram/Slack;
    nessun gesto sprecato.
  - **Overlay tassonomia chiara**: dismissable vs modal documentato
    una volta sola; click outside obbedisce alla tassonomia.
  - **Keyboard parity invariante**: garantito da costruzione (Step 32
    non aggiunge feature mouse-only); future-proof.
  - **Drag deferred**: scope Step 32 minimale, escape-hatch terminale
    documentato; future drag/select può arrivare con ADR dedicato.
  - **Compact-aware**: bbox `nil` per pannelli hidden = ignorati
    naturalmente, senza casi speciali.
- **Negative**:
  - **Refactor leggero per sub-model**: chatList deve esporre
    `SelectIndex` (o accettare un msg semantico). Conversation deve
    esporre un'API per "click su SEND" e "click su textarea" come
    azioni separate. Acceptable: `bubbles/textarea` non ha mouse
    handling proprio, quindi la responsabilità è già del wrapper.
  - **Bbox invalidation discipline**: ogni nuovo trigger di layout
    futuro (Step 33+) dovrà chiamare `recomputeBboxes()`. Mitigato:
    il set è chiuso e documentato; il TLA+ skip dell'ADR documenta
    perché è OK senza spec formale.
  - **Drag/text-select non disponibile in-app**: utenti senza
    escape-hatch del terminale (es. su SSH stretto) non possono
    selezionare testo. Mitigato: documentato come limitazione,
    `y` keybinding futuro (ADR successivo) coprirà il caso.
  - **No double-click handling**: utenti che cercano "click per
    preview, double-click per open" non hanno il preview. Acceptable:
    `j/k` keyboard è il preview equivalente.
- **Rischi**:
  - **Bbox stale dopo nuovo overlay non documentato**: se un futuro
    step aggiunge un overlay senza updating del set di "trigger di
    invalidazione bbox", click outside potrebbe sembrare hittato.
    Mitigato: D2 elenca tutti i trigger; nuovi overlay ereditano
    l'invariante OVERLAY_FIRST; ADR successivi devono documentare
    extension del bbox set.
  - **Click su `sendButton` bbox quando `Active = false`**: oggi
    `sendBtn.Active = false` quando textarea è vuota. Decisione: no-op.
    Rischio: utente clicca pensando di inviare e non succede nulla.
    Mitigato: il bottone è renderizzato in tonalità "disabled" (Step
    15 `ColorButtonDisabledFg`); è visuamente chiaro.
  - **Cursore su border tra panel = ambiguità**: D3 usa half-open
    intervals; nessun "buco". Rischio residuo: se utente scrolla
    sulla riga di separator (1 col verticale), wheel routes a chatList
    (ha la coordinata x0 di chatList = x1 del border). Acceptable e
    deterministico.

## Note su TLA+ (skip giustificato)

**Step 32 non produce una nuova spec TLA+**. Riasoning:

1. **Mouse handling è sincrono**: tutti gli eventi (`MouseMsg`)
   passano per il single Update channel di bubbletea, serializzati.
   Nessuna goroutine, nessun canale custom, nessun lock.
2. **Stato mutato**: lo stesso stato già modellato dalle altre spec
   (`responsive_layout.tla` per `layoutMode`/`compactVisible`,
   `whichkey.tla`/`folders_chatinfo.tla` per `activeOverlay`,
   `multi_select.tla` per `selection`). Il mouse handling è solo un
   **dispatcher addizionale** sulle stesse transizioni; le invarianti
   esistenti restano valide.
3. **Invarianti del mouse routing sono local-pure**: `KEYBOARD_PARITY`,
   `BBOX_TOTAL`, `NO_HIDDEN_CLICK`, `OVERLAY_FIRST`,
   `WHEEL_BY_POSITION`, `CLICK_FOCUS_SHIFT`, `NO_PHANTOM_DRAG` sono
   tutte verificabili statiche/property-test (no temporal logic).
4. **Pattern già accettato**: Step 31 ha skippato la TLA+ per la
   parte sincrona del config loading (la parte async hot-reload ha
   `theming.tla`); stesso schema qui. Step 30 / 29 / 28 hanno fatto
   lo stesso quando la parte modellata era sincrona.

**Trade-off**: rinunciamo alla verifica formale dell'interleaving
mouse↔keyboard↔Telegram. Acceptable perché bubbletea **garantisce**
l'ordering serializzato (single goroutine consumes Update channel);
non c'è interleaving da modellare.

**Re-opening**: se in futuro un mouse handler emette un `tea.Cmd`
asincrono che concorre con un altro flow (es. click che fa partire
un RPC mentre un debounce è pending), allora una spec TLA+ dedicata
diventerà necessaria. Step 32 non introduce queste interazioni.

## Scope

Questa ADR si applica a:

- **Step 32 — Mouse support**: prima introduzione delle dieci decisioni
  (D1..D10).
- Step futuri che introducono **gesti mouse aggiuntivi** (link click
  Step 33, drag-to-select futuro): ereditano D8 (keyboard parity),
  D2 (bbox cache discipline), D5 (overlay tassonomia).
- Step futuri che introducono **nuovi widget cliccabili**: devono
  estendere la `bboxes` map e aggiornare il set di trigger di
  invalidazione (D2).

**Non si applica a**:

- **Mouse capture mode** (`tea.WithMouseCellMotion` vs
  `WithMouseAllMotion`): scelta già fatta in `cmd/tuilegram/main.go:56`
  (CellMotion); ADR-020 lo eredita. Cambio futuro richiede ADR
  successivo.
- **Cursore interno al textarea** (place cursor su click dentro la
  textarea): delegato a `bubbles/textarea`. In v1.0.0 della libreria
  questo è no-op; ADR-020 non se ne occupa.
- **Right click / middle click**: out-of-scope Step 32. `Button =
  Right` e `Button = Middle` sono **scartati** dal router (no-op).
  Ridiscutibile in futuro (es. context menu).
- **Hover effects / tooltips**: richiederebbero `Action = Motion`
  abilitato e processato. Out-of-scope Step 32 per D7.

## Cross-links

- [`../phase-2-behavioral/mouse-routing.md`](../phase-2-behavioral/mouse-routing.md) §Statechart, §Invarianti, §Bbox lifecycle
- [`../phase-3-interactions/mouse-routing-flow.md`](../phase-3-interactions/mouse-routing-flow.md) — sequence diagrams per i 9 scenari principali
- [`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md) §Terminal Events (esteso con dettagli `tea.MouseMsg`)
- Pipeline Step 32 (`development-pipeline.md`)
- [ADR-015 §D3](ADR-015-command-palette-whichkey-help.md) — overlay mutex (ereditato; click outside obbedisce alla mutex)
- [ADR-016 §D5](ADR-016-folder-source-and-filtering.md) — folder sidebar in compact mode (bbox `nil` in compact)
- [ADR-018 §D2, §D4](ADR-018-responsive-layout-threshold-and-tab.md) — `COMPACT_ONE_PANEL` invariante (rispettato dal mouse handler con bbox `nil`); cross-threshold trigger di bbox invalidation
- [ADR-007](ADR-007-overlay-in-flight-rpc.md) — forwardPicker in-flight RPC (rispettato da D5: forwardPicker è modal; click outside no-op)
- [ADR-008](ADR-008-batch-forward-semantics.md) — batch forward (rispettato da D5)
- [`../tui-design.md`](../tui-design.md) §"Mouse Support" (high-level user-facing documentation)
