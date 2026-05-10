# ADR-021: Step 33 polish — pinned bar, link rendering+open, forward display, status bar shortcuts+errors, sender name color (single-ADR multi-feature)

**Stato**: accettato
**Data**: 2026-05-10

## Contesto

Lo Step 33 (FINAL della pipeline) è uno **step di polish** che bundle
cinque piccole feature di rendering/UX accessorie sopra l'app già
funzionalmente completa post-Step 32. La specifica della pipeline
(`development-pipeline.md` §Step 33) elenca:

1. **Pinned message bar**: una barra sotto l'header della conversazione
   mostra il messaggio pinnato della chat (se esiste).
2. **Link rendering + open**: URL nei messaggi sono sottolineati;
   l'utente può aprire un link nel browser di sistema.
3. **Forward display**: messaggi forwardati mostrano `┃ From @source`
   (block prefix) sopra il body. Il *send* di forward è già da Step 21;
   questo step copre il *display* dei forward ricevuti.
4. **Status bar polish**: status bar a 1 riga in fondo allo schermo,
   con shortcuts contestuali a sinistra e ultimo errore/status a destra.
5. **Sender name colorato nei gruppi**: nelle chat di gruppo, il primo
   messaggio di ogni "gruppo per sender" (Step 13 grouping) mostra il
   nome del mittente in un colore deterministico derivato dal sender ID.

Stato pre-Step 33:

- `internal/ui/views/conversation_header.go` (52 righe) renderizza l'header
  della conversazione (titolo + status). Nessun slot per pinned bar.
- `internal/ui/views/conversation_render.go` (110 righe) renderizza i
  messaggi: incoming/outgoing, reply quote, media, reactions. **Nessun**
  rendering di forward header (il campo `ForwardedFrom` non esiste in
  `model.Message`); i link sono testo plain (nessuna detection o
  underline).
- `internal/ui/views/main_view.go` ha già una status bar embrionale
  (`m.statusMsg`) ma senza shortcuts hint sinistra/errori destra
  separati.
- `internal/model/message.go` (40 righe) contiene `Message` con campi
  per reply, media, service, reactions, ma **non** per forward header
  né flag pinned.
- `internal/telegram/convert/` ha moduli per media, reactions, names,
  service. **Manca** un `forward.go` (FwdHeader → model fields).
- ADR-018 (responsive Step 30): pinned bar deve coesistere con Compact
  mode — riserva 1 riga in più sotto l'header.
- ADR-019 (theming Step 31): nuovi colori (link accent, sender palette,
  forward dim, error red, pinned highlight) devono passare per
  `styles.Color*()` — niente literal RGB.
- ADR-020 (mouse Step 32): apertura link via mouse click (OSC 8) è
  fuori scope dell'app (è terminal-mediated); l'app espone un canonical
  path **keyboard-only** (`gx`).

Cinque sotto-decisioni, una per feature, più una **giustificazione TLA+
skip** finale (F). Le feature sono indipendenti per dati/eventi: nessuna
introduce concorrenza nuova; i side-effect sono o pure-View (D-D, D-E)
o `tea.Cmd` sincroni di tipo "spawn process" (D-B). Quindi un singolo
ADR multi-feature è giustificato (mantiene la cardinalità degli ADR
proporzionale alla profondità del trade-off, non al volume di codice).

## Decisione

**Quintupla decisione consolidata in una sola ADR**, una per feature
(A..E), più F (TLA+ skip). Naming convention: `D{letter}{n}` con letter
∈ {A,B,C,D,E}, e.g. `DA1`, `DB2`. Cross-feature invariants raccolti
in §Invarianti finale.

### A — Pinned message bar

#### A1 — Data source: `tg.MessagesGetMessages` per pinned msg ID, fetch su chat open

Telegram API espone in `tg.Dialog` (e in `tg.MessagesDialogs`) il campo
`PinnedMsgID int` — l'ID del messaggio attualmente pinnato (singolo).
Il body del messaggio NON è incluso nel payload `Dialog`: serve una
chiamata separata `api.ChannelsGetMessages` (canali/supergruppi) o
`api.MessagesGetMessages` (cloud chat / megagroup) con `[]InputMessage{InputMessageID{ID: pinnedMsgID}}`.

**Decisione**:

- All'apertura di una chat (`ChatSelectedMsg` → `loadChatCmd`), se la
  chat ha `pinnedMsgID != 0`, lo step convertito da gotd/td include il
  campo `model.Chat.PinnedMsgID`.
- `ConversationModel.Init()` (o `OpenChat()` equivalent) emette un
  `tea.Cmd` aggiuntivo `loadPinnedMessageCmd(chatID, pinnedMsgID)` che
  chiama l'API e ritorna `PinnedMsgLoadedMsg{chatID, *model.Message |
  nil, error}`.
- Risultato cached in `ConversationModel.pinnedMsg *model.Message`
  (single field, sostituibile a ogni cambio chat).
- Su error: `pinnedMsg = nil`, errore loggato in status bar (D-D3),
  bar non renderizzata.

**Razionale**:

- **Cache locale alla conversation**: il pinned msg appartiene
  semanticamente alla chat aperta. Tenerlo in `ConversationModel`
  evita un cache globale prematuro (YAGNI).
- **Fetch on demand**: Telegram non spedisce il body del pinned in
  `dialogs.Get`; serve una RPC esplicita. Fetch lazy (a chat open) è
  il pattern minimo coerente con `loadChatCmd` esistente.
- **Sostituzione su chat switch**: aprire una nuova chat scarta il
  vecchio `pinnedMsg` e ne carica uno nuovo. Pattern coerente con
  `messages` (svuotati a ogni chat switch).
- **Update in tempo reale (out-of-scope)**: gotd/td espone
  `UpdatePinnedMessages` / `UpdateChannelPinnedMessage` per cambi
  in-flight. Step 33 **non** sottoscrive a questi update (deferred):
  il pinned è "snapshot a chat-open", l'utente che vuole vedere un
  unpin/repin riapre la chat. Documentato come limitazione.

#### A2 — UX: single-line truncated, no expand/collapse in Step 33

La barra è alta **2 righe**: 1 riga di contenuto + 1 riga di bordo
inferiore (separator visivo dal viewport). Il body del pinned msg è
**troncato** alla larghezza del pannello con ellipsis (`...`).

Niente expand/collapse, niente jump-to-pinned in Step 33 (deferred).

**Razionale**:

- **Step di polish, non di feature**: aggiungere keybinding per
  jump-to-pinned + statechart per expand/collapse triplica lo scope.
- **Single source of truth UX**: 1 riga è sempre uguale, no flicker
  cross-state.
- **Future**: `JumpToMessageMsg` (già definito Step 26 per search) è
  riusabile per jump-to-pinned in uno step futuro. L'infrastruttura
  esiste ma non è cablata in Step 33.

#### A3 — Visibilità: sempre visibile se `pinnedMsg != nil`

Niente flag "user dismissed this pinned" per-session. Se la chat ha
un pinned msg, la barra è renderizzata. Se l'utente vuole nascondere
una chat con pinned msg pesante, deve unpin lato server (out-of-scope
TUI Step 33).

**Razionale**:

- **Predictability**: stessa chat = stessa UI sempre. Niente "perché
  ora la barra c'è e prima no".
- **Step polish minimalism**: aggiungere uno stato di dismiss richiede
  persist/forget logic (per-session? per-app?). Out-of-scope.

#### A4 — Layout: 2 righe tra header (1 riga) e viewport (variable)

Il layout della conversation passa da:

```
Pre-Step 33 (Step 32 baseline):
┌─ header (1 row) ──────────────────────────────────┐
├─ viewport (height - header - input) ──────────────┤
├─ input area (3+ rows) ────────────────────────────┤
```

a:

```
Step 33 (con pinned msg):
┌─ header (1 row) ──────────────────────────────────┐
│ 📌 truncated pinned msg body...                   │  ← 1 row content
├─ separator ───────────────────────────────────────┤  ← 1 row border
├─ viewport (height - header - pinned bar - input) ─┤
├─ input area ──────────────────────────────────────┤
```

Quando `pinnedMsg = nil`: 0 righe occupate (no rendering, viewport
recupera lo spazio).

**`ConversationModel.contentHeight`** già implementato (Step 14)
sottrae `headerHeight + inputHeight`; estendiamo a sottrarre anche
`pinnedBarHeight ∈ {0, 2}` derivato da `pinnedMsg != nil`.

**Razionale**:

- **Layout deterministico**: bbox del viewport ricalcolata
  esattamente come per le altre dimensioni layout. Compatibile con
  bbox cache (ADR-020 §D2, **trigger di invalidation** aggiunto:
  `PinnedMsgLoadedMsg` muta `pinnedBarHeight`).
- **2 righe è il minimo per "barra visibile + separator"**: 1 riga
  sembrerebbe parte dell'header (no separator).

#### A5 — Compact mode: pinned bar visibile (no skip)

In `layoutMode = Compact`, la barra pinned è renderizzata identica.
Riserva sempre 2 righe.

**Razionale**:

- **Coerenza information density**: in Compact lo schermo è già
  minimale; togliere il pinned msg lo rende **invisibile**, viola
  l'aspettativa "questa chat ha un pin importante".
- **Costo: 2 righe**: in Compact tipico (height ~30) il viewport ha
  ~22 righe disponibili; sottrarne 2 è ~10% — accettabile.
- **Future toggle**: utenti che non vogliono vedere la pinned bar in
  Compact possono in futuro avere un setting in `config.toml`. Non in
  Step 33.

#### A6 — Step 33 scope: **first** pinned message only

Telegram supporta **multiple pinned messages** per chat (lista, non
singleton — campo `tg.MessagesGetSearch{Filter: InputMessagesFilterPinned}`
permette enumerare). `tg.Dialog.PinnedMsgID` è solo il **più recente**.

**Decisione**: Step 33 mostra **solo** il pinned più recente (cioè
`Dialog.PinnedMsgID`). Multi-pinned navigation (carosello con
`Up/Down` sulla pinned bar) è **deferred**.

**Razionale**:

- **YAGNI**: la maggior parte delle chat ha 0 o 1 pinned. Multi-pinned
  è una feature di canali curated.
- **UI minimale per polish step**: carosello pinned introduce
  cursore + navigazione + counter "1/N" — è una feature, non polish.

### B — Link rendering + open

#### DB1 — Link detection: `MessageEntities` (Telegram authoritative)

Il body di un messaggio gotd/td (`tg.Message.Message string`) è plain
text, ma `tg.Message.Entities []MessageEntityClass` contiene la
**parsing già fatta da Telegram** (URL, mention, hashtag, bot command,
text-link, code, pre, italic, bold, ...). I tipi rilevanti per Step 33:

| Entity type | Significato | Trigger Step 33? |
|-------------|-------------|------------------|
| `MessageEntityURL` | URL inline (es. `https://example.com`) | **Sì** (target di `gx`) |
| `MessageEntityTextURL` | Hidden link (es. `[click here](url)`) | **Sì** (target di `gx`) |
| `MessageEntityMention` | `@username` | No (Step 33 — deferred) |
| `MessageEntityHashtag` | `#tag` | No |
| `MessageEntityBotCommand` | `/start` | No |
| Altri (italic, bold, code, ...) | Formatting | No |

**Decisione**: il convert layer (`internal/telegram/convert/links.go` —
nuovo file) trasforma `[]MessageEntityClass` in `[]model.MessageLink` con
shape minimale `{Offset, Length int, URL string}`. Per `MessageEntityURL`,
`URL = msg.Message[Offset:Offset+Length]`. Per `MessageEntityTextURL`,
`URL = entity.URL` (campo separato del tipo gotd).

Niente regex client-side: tutta la link detection è server-authoritative.

**Razionale**:

- **Server-authoritative parsing**: Telegram ha già fatto il lavoro.
  Reinventare con regex significa divergenza (es. URL con parentesi
  bilanciate, IDN, ecc.).
- **Hidden text-links supportati gratis**: `MessageEntityTextURL` è
  esattamente "il URL non è il testo visibile" (e.g. markdown
  `[click](url)`). Una regex perderebbe questi.
- **Costo: 1 file convert, ~30 righe**: trivial.

#### DB2 — Rendering: lipgloss underline + accent color, OSC 8 hyperlinks opt-in

Ogni `MessageLink` nel testo renderizzato è wrappato in lipgloss style
con `Underline(true).Foreground(styles.ColorLink())` (nuovo color key
in `theme.toml`, default = primary accent).

**OSC 8 hyperlinks** (escape sequence ANSI `ESC ] 8 ; ; URL ESC \`)
sono inclusi: terminali moderni (iTerm2, WezTerm, Kitty, Alacritty
recente, GNOME Terminal recente) li rendono **clickabili nativamente**
(Cmd/Ctrl+click apre il browser, gestito **dal terminale**, non
dall'app). Terminali che non supportano OSC 8 ignorano gli escape (no
visual artifact); l'underline+color resta.

**Decisione**: rendering pipeline = OSC 8 wrapping + lipgloss
underline+color. Graceful degrade: terminali no-OSC8 vedono
underline+color senza click nativo.

**Razionale**:

- **Underline+color è universale**: lipgloss funziona ovunque.
  L'utente vede sempre il link distinto dal testo plain.
- **OSC 8 è additive**: escape sequence ignorate dai terminali che
  non le capiscono (no rendering breakage). I terminali moderni
  forniscono il "Cmd+click apre browser" gratis — feature mouse a
  costo zero per l'app.
- **No conflitto con keyboard path (DB4)**: OSC 8 click è
  terminal-mediated (l'evento mouse non passa per l'app). Il path
  canonical app-mediated è `gx` keyboard. Coesistono pacificamente.
- **Coerenza con ADR-020 §D8 (KEYBOARD_PARITY)**: il path canonical
  resta keyboard. OSC 8 mouse è un *bonus* terminal-driven, non
  un'API dell'app.

#### DB3 — Open command: `runtime.GOOS` switch + `exec.Command`

Apertura di un URL via process spawn:

| OS | Comando |
|----|---------|
| `darwin` | `open <url>` |
| `windows` | `cmd /c start <url>` (o `rundll32 url.dll,FileProtocolHandler <url>`) |
| `linux` / `freebsd` / others | `xdg-open <url>` |

**Decisione**: `internal/ui/cmd/openlink.go` (nuovo, ~25 righe) espone
un `tea.Cmd` `openLinkCmd(url string)` che fa il dispatch via
`runtime.GOOS`. Errori (`exec.Command` failure, comando non trovato)
sono loggati in status bar via `statusMsg` (D-D3).

**Razionale**:

- **Pattern standard cross-platform**: identico a quello di
  `webbrowser.Open()` di Go (gotd ne ha uno equivalente nei contrib).
  Nessuna libreria esterna richiesta.
- **No external dep**: importare `pkg/browser` è overkill per ~25
  righe di codice già stabile.
- **`tea.Cmd` async**: l'apertura del browser non blocca l'Update
  loop; il completamento è ignorato (fire-and-forget) — il browser
  apre in un processo a parte.

#### DB4 — Trigger: `gx` chord (vim-style), apre il **primo** link nel messaggio selezionato

Step 22 ha introdotto il message cursor (un cursore che scorre i
messaggi nel viewport). Step 33 estende il keymap del messaggio
selezionato con il chord `gx` (mnemonic: "go (open) external"):

- `g` (prefix) → schedule `WhichKeyTickCmd` (300ms timeout, ADR-015 §D5).
- `x` (continuation) → emette `OpenLinkMsg{URL: links[0].URL}` se
  `len(links) > 0` per il messaggio selezionato; altrimenti
  `statusMsg := "no links in selected message"`.

**Decisione**: `gx` apre il **primo** link (`links[0]`) del messaggio
selezionato. Multi-link picker (es. `gx` apre overlay "select which
link") è **deferred**.

**Razionale**:

- **Vim-parity**: `gx` è il chord standard di Vim/Neovim/elinks/Mutt
  per "open URL under cursor". Familiare per utenti TUI-savvy.
- **Riuso whichkey infrastructure**: `g` è già un prefix Step 28
  (per `gg` scroll-top, ADR-015 §D5). `gx` è una continuation in più
  nello stesso registry, zero nuova infrastruttura.
- **First-link policy**: la maggioranza dei messaggi con link ne ha
  1. Per multi-link, l'utente può sempre fallback su mouse Cmd+click
  (OSC 8) sui link successivi. Multi-link picker è polish del polish.
- **No-op + status hint**: messaggi senza link → no-op visibile
  ma esplicito ("no links"). Non silenzioso (sarebbe confondente).

#### DB5 — Cursor model: NO sub-cursor per i link

Alternativa scartata: aggiungere un sub-cursore intra-messaggio per
selezionare quale link aprire (es. `Tab` cicla tra i link del messaggio
corrente, `Enter` apre).

**Decisione**: niente sub-cursore. `gx` apre `links[0]`. Multi-link
gestito post-Step 33.

**Razionale**:

- **Complessità sproporzionata**: sub-cursore richiede stato
  (`linkCursor int`), reset su cambio messaggio, rendering
  highlight del link "focused", keymap dedicate. Tutto per la **minoranza**
  di casi multi-link.
- **OSC 8 mouse copre il caso**: terminale moderno → click su
  qualsiasi link funziona già.
- **Future**: ADR successivo può aggiungere sub-cursore. Niente in
  Step 33 lo blocca.

#### DB6 — Step 33 scope: solo `http(s)://...` aperti, no telegram://, no mailto:

Filtro su `URL` prefix:

- `http://...` → aperto.
- `https://...` → aperto.
- `tel:...`, `mailto:...`, `telegram://...`, `tg://...`, custom schemes
  → **non aperti** in Step 33 (logged "scheme not supported" in
  status bar).

**Razionale**:

- **`xdg-open`/`open`/`start` aprono qualsiasi scheme**: rischio di
  side-effect non voluti (es. `tel:` su macOS apre FaceTime). Per uno
  step di polish, restringiamo a http(s) sicuri.
- **`telegram://` deep links**: aprirebbero un'altra app Telegram.
  Out-of-scope per un client TUI; gestione futura via deep-link
  routing interno.
- **Whitelist semplice**: il filtro è una check su prefix string
  (~3 righe). Implementabile e auditable.

### C — Forward display

#### DC1 — Data source: `tg.Message.FwdFrom *MessageFwdHeader` → convert

`tg.MessageFwdHeader` è un campo opzionale di `tg.Message`; quando
presente indica che il messaggio è un forward. Campi rilevanti:

| Campo gotd | Significato |
|------------|-------------|
| `FromID PeerClass` | Peer originale (user/chat/channel) |
| `FromName string` | Nome visualizzato dell'autore originale |
| `Date int` | Data del messaggio originale |
| `ChannelPost int` | Se forward da channel post, l'ID originale |
| `PostAuthor string` | Per channel post, autore (se signed) |

**Decisione**: nuovo file convert `internal/telegram/convert/forward.go`
trasforma `tg.MessageFwdHeader` in due campi nuovi di `model.Message`:

```go
type Message struct {
    ...existing...
    ForwardedFrom   string  // display label: "@username" | "Display Name" | "Hidden"
    IsForwarded     bool    // discriminante; FALSE se FwdFrom == nil
}
```

Logica del label (DC3):

1. Se `FromID` è risolvibile a un user con `Username != ""`:
   `ForwardedFrom = "@" + Username`.
2. Se `FromID` è user senza username: `ForwardedFrom = FirstName + LastName`.
3. Se `FromID` è channel/chat: `ForwardedFrom = Channel.Title`.
4. Se `FromID == nil` AND `FromName != ""`: `ForwardedFrom = FromName`
   (channel con autore nascosto firma comunque il nome).
5. Else: `ForwardedFrom = "Hidden"` (forward da private chat con
   privacy "hide my account").

**Razionale**:

- **`IsForwarded` esplicito**: serve come trigger del rendering
  block `┃` (DC2). Avere solo `ForwardedFrom != ""` come check è
  fragile (Hidden è valido string ma non un peer reale).
- **Logica fallback layered**: copre tutti i casi gotd documentati,
  con "Hidden" come ultimo fallback (matching UX Telegram official).
- **Convert layer è il single point of truth**: il rendering layer
  vede solo `IsForwarded + ForwardedFrom`, non tocca `tg.*` mai.

#### DC2 — Rendering: prefix `┃ ` + dim color, su ogni linea del body

Block prefix: U+2503 (`HEAVY VERTICAL BAR`) + spazio, prepended a
**ogni linea** del body multi-line. Header line (la prima) ha
`┃ From <ForwardedFrom>` in dim italic; lines successive hanno solo
`┃ ` + body.

Esempio multi-line:

```
┃ From @alice
┃ Hello, here is a long
┃ multi-line message that
┃ Alice originally wrote.
```

Stile lipgloss: `Foreground(styles.ColorTextDim()).Italic(true)` per
header line; body lines `Foreground(styles.ColorTextDim())`.

**Decisione**: prefix per-line, header dedicata (1 riga in più).

**Razionale**:

- **Visual consistency**: il `┃` block style è il pattern di
  Telegram official Desktop / Android (block prefix per quote/forward).
  Familiare.
- **Per-line prefix**: senza, multi-line forward con riga vuota (es.
  paragraph break) "spezza" il blocco visualmente. Per-line lo
  mantiene coeso.
- **Header line aggiunta (no inline)**: distinguere `From <X>` dal
  body via riga separata è più leggibile di "From X: body" inline.

#### DC3 — Sender label: gerarchia username > display name > hidden

Vedi DC1 per la gerarchia. **Sintesi**:

```
Priority:
  1. @{username}             (if user has username)
  2. {first} {last}          (if user, no username)
  3. {channel/chat title}    (if forward from channel/group)
  4. {fwd_header.from_name}  (if explicit display name preserved)
  5. "Hidden"                (default fallback)
```

#### DC4 — Indentazione: `┃ ` a colonna 0 della bubble, NO doppio bordo

La bubble incoming/outgoing ha già un proprio styling (Step 12). Il
prefix `┃` è renderizzato **dentro** la bubble (a colonna 0 della text
area, non sopra al border della bubble).

Visivo:

```
SenderName:
  ┃ From @alice
  ┃ Original body line 1
  ┃ Original body line 2  10:23 ✓✓
```

Non si aggiunge un border outer per il forward (sarebbe doppio bordo
con la bubble esistente). Il `┃` interno è il signaling sufficiente.

**Razionale**:

- **No doppio chrome**: bubble esterna + block prefix interno = chiaro.
  Doppio bordo sarebbe rumoroso.
- **Coerenza con `replyBarStyle()` esistente** (Step 18,
  `conversation_render.go:106`): pattern già adottato per reply
  quote — stesso paradigma per forward.

#### DC5 — Step 33 scope: solo display, no compose-as-forward in questo step

Step 21 ha già introdotto **compose forward** (l'utente può forwardare
un suo messaggio). Step 33 copre il **display dei forward ricevuti**
(messaggi nel viewport con `FwdFrom != nil`).

Per evitare confusione: il path "user `f` su msg → forward picker →
send" (Step 21) emette **un nuovo messaggio nella chat target** che
porta `FwdFrom` settato. Quando il destinatario riceve quel messaggio
(o quando chi ha forwardato lo rilegge), Step 33 si attiva nel render.

### D — Status bar polish

#### DD1 — Layout: 1 riga, left = shortcuts hint, right = errori/status

La status bar è un singolo `lipgloss.JoinHorizontal()` di due slot:

```
[ shortcuts hint left ............................ status/error right ]
```

- **Slot sinistro**: dipende dal focus corrente (`activePanel` +
  `activeOverlay`). Es.:
  - `activeOverlay = none, activePanel = ChatList` →
    `j/k navigate · Enter open · / search · F folders · ? help`
  - `activeOverlay = none, activePanel = Conversation` →
    `j/k cursor · r reply · e edit · gx open link · Esc back`
  - `activeOverlay = palette` →
    `↑/↓ select · Enter run · Esc cancel`
  - `activeOverlay = forward` →
    `↑/↓ select · Enter forward · Esc cancel`
- **Slot destro**: ultimo `statusMsg`. Distinguibile per prefix:
  - errore: `✕ <msg>` con `Foreground(styles.ColorError())`.
  - info: `<msg>` plain con `Foreground(styles.ColorTextDim())`.

Width split:
- Right slot: width della stringa renderizzata (no padding extra).
- Left slot: `available - right - 2` (gap 2 colonne).
- Se left overflow: ellipsize (`...`) sul margine destro del slot left.
- Se right overflow: ellipsize (`...`) sul margine destro del slot
  right.

**Razionale**:

- **Information density**: 1 riga dual-slot massimizza info utile in
  spazio minimo. Pattern di Vim/Neovim/Mutt status line.
- **Focus-aware shortcuts**: cambiare slot left in base al focus
  riduce il carico cognitivo (l'utente vede solo le keys rilevanti).
- **Right-aligned errors**: convenzione UI (notifiche, badge, error
  count) sono sempre a destra. Coerente.

#### DD2 — Shortcuts source: function `keymapHint(focus, mode)` deterministica

**Decisione**: una funzione pure-View `keymapHint(activePanel, activeOverlay) string`
ritorna la stringa hint. Non un evento, non un msg: viene chiamata
durante `View()` ad ogni render.

Implementazione concettuale:

```
function keymapHint(panel, overlay):
    if overlay != none:
        return overlayHint(overlay)        // palette/help/forward/...
    if multiSelect:
        return "Space toggle · f forward · D delete · Esc cancel"
    switch panel:
        case ChatList:        return "j/k · Enter · / search · F folders · ? help"
        case Conversation:    return "j/k · r reply · e edit · gx link · Esc back"
        case Folders:         return "j/k · Enter select · F close"
    default: return ""
```

**Razionale**:

- **Pure function**: deterministica da stato corrente → niente
  invariant violations.
- **No evento dedicato**: aggiungere `StatusBarHintMsg` e ricalcolare
  su mutazione del focus introduce overhead per zero gain (la View()
  già rerender ogni cycle).
- **Single function per refactor centralizzato**: future modifiche
  delle keybinding sono concentrate in `keymapHint()`.

#### DD3 — Error vs status: prefix `✕` per errori, plain per info, no auto-clear

Già esiste `m.statusMsg string`; estendiamo a:

```go
type statusEntry struct {
    Text    string
    IsError bool
}
```

(Field `m.status statusEntry`). Tutti i `*ResultMsg{err}` con `err != nil`
settano `m.status = {err.Error(), IsError: true}`. Tutti i path info
settano `IsError: false`.

**Auto-clear**: out-of-scope Step 33. Il messaggio resta finché un
nuovo `statusMsg` lo sostituisce. (ADR futuro: TTL 5s come typing
indicator pattern.)

**Razionale**:

- **Distinzione visuale chiara**: `✕` + colore rosso = errore
  inequivocabile. `colordim` plain = info passiva.
- **Sticky behavior preservato**: utenti che leggono lentamente non
  perdono l'errore. Auto-clear richiederebbe `tea.Tick` + invariant
  TLA-style "stale clear benigno"; YAGNI per polish step.

#### DD4 — Width: gap di 2 colonne, ellipsize entrambi gli slot

Quando sommato il width supera quello disponibile:

```
left = "j/k · r reply · e edit · gx link · Esc back"  (44 cols)
right = "✕ failed to send: timeout"                  (28 cols)
gap = 2
total_needed = 44 + 2 + 28 = 74 cols
available = 50 cols
```

Algoritmo: priorità al **right** (errori sono critici). Tronca il
left a `available - right - gap - 3` (3 = "..."). Se ancora overflow,
tronca anche il right.

**Razionale**:

- **Error visibility-first**: meglio perdere uno shortcut hint che
  perdere un errore.
- **Deterministic ellipsis**: `lipgloss.Width()` + slicing manuale
  (UTF-8 aware via `runewidth`).

#### DD5 — Theme integration: tutti i colori da `styles.Color*()`

Nuovi color keys in `theme.toml` (estensione ADR-019):

| Key | Default | Uso |
|-----|---------|-----|
| `error` (esistente?) | `#ff5555` | prefix `✕` + error text |
| `link` (nuovo) | derived from `primary` | underline + foreground |
| `senderPalette[]` (nuovo) | 8 colors | hash deterministic per sender (E2) |
| `pinned` (nuovo) | `#ffd700` (giallo dim) | icon `📌` + bordo bottom della pinned bar |
| `forwardLabel` (nuovo) | derived from `textDim` | "From <X>" italic |

**Decisione**: estendere `theme.toml` schema con i 4 nuovi key (più
palette di 8). ADR-019 §D7 (`styles.Color*()` accessor) si applica
naturalmente: nuovi accessor `styles.ColorLink()`, `styles.ColorPinned()`,
`styles.ColorSenderPalette() []lipgloss.Color`, `styles.ColorForwardLabel()`.

**Razionale**:

- **Coerenza ADR-019**: theming è centralizzato. Step 33 aggiunge
  key, non bypassa.
- **Default ragionevoli**: `link` derivato da `primary` evita
  duplicazione; utenti che custom-izzano `primary` ottengono link
  coerenti gratis.

### E — Sender name colorato nei gruppi

#### DE1 — Rule: primo messaggio di ogni "group-block" mostra nome colorato

Step 13 ha introdotto il **grouping per sender** (messaggi consecutivi
dello stesso autore sono raggruppati visualmente: il nome appare solo
sopra il primo del blocco, le repliche heredano implicitamente). Step
33 colora il **nome del primo messaggio del blocco**.

Render:

```
Pre-Step 33 (Step 13+):
Alice:
  Hello there
  How are you?
Bob:
  Doing fine

Step 33 (in group chat):
[Alice colored]:           ← nome in colore X derivato da Alice.ID
  Hello there
  How are you?
[Bob colored]:             ← nome in colore Y derivato da Bob.ID
  Doing fine
```

Body resta `Foreground(styles.ColorText())` (non colorato).

**Decisione**: solo il **nome** è colorato, body invariato.

**Razionale**:

- **Visual distinction senza overload**: colorare il body sarebbe
  rumoroso (8 sender = 8 colori di body). Solo il nome basta per
  parsing veloce di "chi sta scrivendo".
- **Coerenza con Telegram Desktop**: stesso pattern.

#### DE2 — Color hash: `senderID % len(palette)`, palette 8 colori

Funzione deterministica:

```
function senderColor(senderID int64, palette []Color) Color:
    idx := abs(senderID) % len(palette)
    return palette[idx]
```

Palette di 8 colori (definita in theme):
1. red-tone, 2. orange-tone, 3. yellow-tone, 4. green-tone,
5. cyan-tone, 6. blue-tone, 7. purple-tone, 8. pink-tone.

Default theme: derivati da hue equispaziate, saturazione moderata
(non strident).

**Decisione**: hash modulo, palette di 8 colori theme-defined. Stesso
sender ID → stesso colore sempre (deterministico).

**Razionale**:

- **Stabilità cross-session**: utente vede Alice sempre arancione,
  Bob sempre verde — non importa quando apre la chat. Build trust
  visivo.
- **8 colori sufficienti**: in chat di gruppo tipici (5-20
  partecipanti attivi), 8 colori distinguono bene; collisioni rare
  e non dannose.
- **No hash crypto**: `senderID` è già un int64 Telegram, ben
  distribuito → modulo è sufficiente. Niente FNV/xxhash.

#### DE3 — Applies to: solo `model.ChatGroup`

| ChatType | Sender name colorato? |
|----------|----------------------|
| `ChatPrivate` (1-1) | No (irrilevante: 1 sender + me) |
| `ChatGroup` (group/megagroup) | **Sì** |
| `ChatChannel` (broadcast) | No (1 source, già reso bold) |
| `ChatBot` | No (1 sender) |

**Decisione**: rendering switch su `chat.Type == model.ChatGroup`.

**Razionale**:

- **Information density**: in 1-1 / channel / bot, il sender è
  ovvio. Colorarlo non aggiunge info.
- **Group-only riduce visual clutter**: utente percepisce il colore
  come signaling significativo, non decorazione.

#### DE4 — Integrazione: `incomingNameStyle()` riceve color override

`conversation_render.go:47` attuale:

```go
name := incomingNameStyle().Render(msg.SenderName + ":")
```

diventa concettualmente (pseudo-code):

```
color := defaultNameColor
if chat.Type == ChatGroup:
    color = senderColor(msg.SenderID, styles.ColorSenderPalette())
name := lipgloss.NewStyle().Foreground(color).Bold(true).Render(msg.SenderName + ":")
```

**Decisione**: refactor `incomingNameStyle()` per accettare un Color
parametrico, con default = ColorText. ChatGroup branch passa il
colore hash-derivato.

**Razionale**:

- **Single point of change**: il refactor è confinato a una funzione.
- **Backward compatible**: chiamata senza param (es. test) ritorna
  default color.

### F — TLA+: skip giustificato

**Step 33 NON produce nuova TLA+ spec.** Reasoning:

1. **Tutte le 5 sub-feature sono sincrone**:
   - **A (pinned)**: `loadPinnedMessageCmd` è un singolo `tea.Cmd`
     async (RPC); il result `PinnedMsgLoadedMsg` è dispatched al
     single Update channel. Già modellato dal pattern generale di
     `tea.Cmd → Result Msg` (ADR-001 concurrency model). Nessun
     interleaving non-modellato.
   - **B (link)**: `gx` chord usa whichkey infrastructure (ADR-015
     §D5 — già modellata in `whichkey.tla`). `openLinkCmd` è
     fire-and-forget, no result expected → no concurrency to model.
   - **C (forward)**: pure-View rendering. Zero stato dinamico.
   - **D (status bar)**: pure-View rendering (`keymapHint()` is a
     pure function); error state mutation è side-effect dei
     `*ResultMsg` esistenti.
   - **E (sender color)**: pure-View rendering (hash di `senderID`
     deterministico).

2. **Nessuna nuova goroutine**: tutti gli effetti async passano per
   `tea.Cmd → tea.Msg` (single Update channel di bubbletea, già
   serializzato). Nessun canale custom, nessun lock, nessun shared
   memory cross-thread.

3. **Invarianti documentati staticamente**: vedi §Invarianti finale.
   Sono tutti **stato-locale** o **funzionali** (es. "rendering è
   deterministico"), verificabili via property test, non temporal
   logic.

4. **Pattern già accettato**: ADR-020 (Step 32, mouse), ADR-019
   (Step 31, config sync part), ADR-018 (Step 30, layout), ADR-017
   (Step 29 chat info) — tutti hanno skippato la TLA+ con reasoning
   analogo. Step 33 è il caso più semplice (zero nuova concorrenza).

**Trade-off**: rinunciamo alla verifica formale degli interleaving
"pinned-load completes after chat switch" (es. Alice apre Chat A,
chiude prima che `PinnedMsgLoadedMsg` arrivi, apre Chat B → vede il
pin di A?). **Mitigation**: il pattern di stale-completion-drop è già
modellato in `folders_chatinfo.tla` (`STALE_COMPLETION_DROP`
invariant per `ChatInfoCompletionMsg`); applichiamo lo stesso pattern
qui (vedi §Invarianti `PINNED_STALE_DROP`). Pattern collaudato → spec
TLA+ dedicata sarebbe ridondante.

**Re-opening**: se in futuro un mouse-driven path su pinned bar (es.
click su pinned msg per jump) introduce un'interazione concorrente
con search overlay o altri RPC, allora una spec dedicata diventerà
necessaria. Step 33 non introduce queste interazioni.

## Invarianti (cross-feature)

Da modellare in `phase-2-behavioral/step33-polish.md` come property:

| ID | Feature | Statement |
|----|---------|-----------|
| `PINNED_OFFSET_RESERVED` | A | `viewport.Height = chrome - inputHeight - headerHeight - pinnedBarHeight` con `pinnedBarHeight ∈ {0, 2}` derivato da `pinnedMsg != nil`. |
| `PINNED_STALE_DROP` | A | `PinnedMsgLoadedMsg{chatID}` con `chatID != activeChatID` è dropped (no mutation). Pattern `STALE_COMPLETION_DROP` riusato. |
| `PINNED_SINGLE_PER_CHAT` | A | `ConversationModel.pinnedMsg` è singleton (no list); ogni `ChatSelectedMsg` lo sostituisce o azzera. |
| `LINK_OPEN_KEYBOARD_PARITY` | B | `gx` è il **canonical app-mediated** path; OSC 8 mouse click è terminal-mediated (out-of-app). Coesistono ma `gx` è sempre disponibile. (Estende ADR-020 §D8 KEYBOARD_PARITY.) |
| `LINK_DETECTION_AUTHORITATIVE` | B | Link detection deriva esclusivamente da `tg.Message.Entities`, mai da regex client-side. |
| `LINK_OPEN_HTTP_ONLY` | B | `openLinkCmd` apre solo `http://` / `https://`. Altri scheme → no-op + status hint. |
| `FORWARD_PREFIX_PER_LINE` | C | Ogni linea di un forward ha prefix `┃ `; mai un body line "nudo" senza prefix se `IsForwarded = TRUE`. |
| `FORWARD_LABEL_FALLBACK_CHAIN` | C | Sender label segue gerarchia username > display name > channel title > from_name > "Hidden". Mai stringa vuota visibile. |
| `STATUSBAR_TWO_SLOT` | D | Status bar = exactly 2 slot (left hint, right error/info). Nessun terzo slot. |
| `STATUSBAR_ERROR_PRIORITY` | D | Su overflow, ellipsize il **left** prima del right. Errori non sono mai droppati silenziosamente. |
| `STATUSBAR_KEYMAP_DETERMINISTIC` | D | `keymapHint(panel, overlay)` è pure function: stessi argomenti → stessa stringa, sempre. |
| `SENDER_COLOR_DETERMINISTIC` | E | `senderColor(id) = palette[abs(id) % len(palette)]`. Stesso `id` → stesso colore, cross-session. |
| `SENDER_COLOR_GROUP_ONLY` | E | Color override applicato sse `chat.Type == ChatGroup`. Altri chat type → default color. |
| `RENDER_NO_NEW_GOROUTINE` | F | Step 33 NON spawna nuove goroutine. Tutto async è `tea.Cmd` → single Update channel. |

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **Single ADR multi-feature (scelta)** | Cardinalità ADR proporzionale a depth del trade-off, non LOC; cross-references chiari (invarianti raccolti); skip TLA+ giustificato in un posto | File più lungo (6.5k caratteri vs. 5×1.5k); review più pesante |
| **5 ADR separati (021-pinned, 022-links, 023-forward, 024-statusbar, 025-sender)** | Granularità maggiore | Frammentazione; le decisioni sono intercorrelate (theme keys, render pipeline); reviewer paga overhead di context-switch |
| **Pinned bar: cache globale `pinnedMsgs[chatID]`** | Switch chat senza re-fetch | YAGNI; introduce invalidation policy; overhead memoria; no concrete user need |
| **Pinned: subscribe a `UpdatePinnedMessages`** | Real-time pin/unpin visibile | Out-of-scope polish; richiede dispatcher extension; mitigato da "snapshot at chat-open" |
| **Multiple pinned con carosello** | Feature parity con Telegram official | Non è polish, è feature; richiede sub-cursore + counter UI |
| **Link detection via regex client** | Self-contained, no dependency su gotd entities | Fragile (URL edge cases), divergenza da Telegram, hidden text-link impossibili |
| **No OSC 8 (solo lipgloss underline)** | Universal terminal compat | Perde click nativo gratis su terminali moderni; choice asimmetrica |
| **OSC 8 only, no lipgloss underline** | Terminali moderni: OSC 8 già evidenzia link | Terminali no-OSC8 non vedono link distinto; UX inconsistente |
| **Open link via `gh` CLI / `pkg/browser`** | Cross-platform abstracted | Extra dep per ~25 LOC; `runtime.GOOS` switch è banale |
| **Open all schemes (mailto, tel, telegram://)** | Power user flexibility | Side-effect imprevisti (FaceTime, mail client unwanted); polish step deve essere safe |
| **Sub-cursore link con `Tab` cycle** | Multi-link gestito | Stato extra, rendering highlight, complessità sproporzionata; OSC 8 mouse copre il caso |
| **Forward inline `From X: body`** | Compatto | Multi-line forward perde coesione visiva; pattern Telegram block è canonico |
| **Forward come reply (riusa `replyBarStyle()`)** | Zero nuovo codice | Confonde semanticamente reply vs forward; user-cognitive cost |
| **Status bar 1-slot full width** | Più semplice | Perde info shortcuts O perde info errori; non c'è middle ground |
| **Status bar 3-slot (hints, mode, error)** | Più info | Ognuno troppo stretto in Compact; invariante più complessa |
| **Auto-clear errori dopo 5s** | UX: notifiche non si accumulano | Richiede `tea.Tick` + stale-tick benign invariant; YAGNI per polish |
| **Sender color: random per session** | Niente cross-session caching | Confonde utente ("Alice era verde, ora è blu?"); rotto principio di stabilità |
| **Sender color: hash crittografico (FNV/xxhash)** | Distribuzione "perfetta" | Overkill; modulo su int64 Telegram è già ben distribuito; +20 LOC zero gain |
| **Sender color: body intero colorato** | Massima distinzione | Visual clutter; legge male su sfondi diversi; nome basta |
| **Sender color in private chat / channel** | Uniformità | Add zero info (1 sender); contraddice DE3 reasoning |
| **TLA+ spec dedicata Step 33** | Verifica formale | 5 feature sync, zero concorrenza nuova; spec sarebbe trivial e ridondante; pattern skip già accettato 4 volte |

## Conseguenze

- **Positive**:
  - **Polish step focused**: 5 micro-feature con scope chiaro,
    ognuna < 100 LOC stimato. Ogni feature singolarmente dimostrabile.
  - **Server-authoritative parsing** (link entities, forward header):
    no regex client, no edge case divergenze.
  - **OSC 8 hyperlink + lipgloss underline graceful**: terminali
    moderni clicabilità nativa gratis; vecchi terminali underline
    visibile.
  - **Keyboard parity preserved** (ADR-020 §D8): `gx` è canonical;
    OSC 8 mouse è bonus terminal-mediated.
  - **Theme integration**: 4 nuovi color keys + palette di 8;
    estensione naturale ADR-019.
  - **Status bar dual-slot**: pattern Vim/Mutt-like; deterministico;
    error visibility prioritized.
  - **Sender color deterministic + group-only**: utility max,
    clutter min.
  - **TLA+ skip giustificato**: zero nuova concorrenza; pattern di
    skip già stabilito.
  - **Extension points puliti**: multi-pinned, multi-link picker,
    auto-clear errori, sub-cursore link → tutti deferrabili a step
    futuri senza refactor.
- **Negative**:
  - **Pinned re-fetch on chat switch**: costo network +1 RPC per
    ogni chat aperta che ha pinned msg. Mitigato: piccolo, single
    msg fetch.
  - **No real-time pin update**: utente che vuole vedere unpin/repin
    deve riaprire la chat. Documented limitation.
  - **First-link-only su `gx`**: messaggi multi-link richiedono OSC 8
    o future picker. Deferrable.
  - **Sender color collision possibile**: 8 colori, più di 8 sender
    → due sender stesso colore. Acceptable (hash è il pattern
    standard).
  - **Status bar: no auto-clear**: errori restano finché un altro
    msg li sostituisce. Sticky può confondere. Future ADR può aggiungerlo.
  - **Theme schema breaking**: i 4 nuovi color keys sono **additive**
    (default fallback se assenti); non rompono `theme.toml`
    esistenti. Coerente con ADR-019 §D5 (override-by-key, missing key
    → default).
- **Rischi**:
  - **Pinned bbox invalidation mancata**: se un futuro step muta
    `pinnedBarHeight` senza invalidare bbox, mouse click malposizionato.
    Mitigato: `PinnedMsgLoadedMsg` aggiunto al set di trigger ADR-020 §D2.
  - **OSC 8 in terminali quirky**: alcuni terminali parziali stampano
    raw escape (`]8;;url\`) come testo. Mitigato: detection via env
    `TERM` o opt-out via `config.toml` flag (futuro). In Step 33,
    accettiamo il rischio (terminali noti supportano OSC 8 OR
    ignorano silenziosamente).
  - **Link click su `tg://` deep link**: utente clicca aspettandosi
    di aprire un'altra chat dentro tuilegram. Step 33 dice "scheme
    not supported". Acceptable: deep linking interno è una feature,
    non polish.
  - **Sender color su sender molto chiari/scuri**: palette default
    deve essere readable su tutti i background. Mitigato: palette è
    theme-defined; utenti dark/light mode possono custom.
  - **Status bar overflow ellipsize troppo aggressivo**: utente in
    terminale molto stretto (compact mode <60 cols) potrebbe vedere
    `j/k...` invece di hint utile. Mitigato: in Compact, hint è
    abbreviato (`j/k · / · ?` invece di full); D-D2 può essere esteso
    con compact-aware hints.

## Note su TLA+

**Skip giustificato** — vedi §F per il reasoning completo. Cinque
sub-feature sincrone, zero nuova goroutine, invarianti
verificabili staticamente (no temporal logic). Pattern di skip
allineato con ADR-017, ADR-018, ADR-019, ADR-020.

## Scope

Questa ADR si applica a:

- **Step 33 — Pinned + links + forward display + status bar polish**:
  prima introduzione delle decisioni A1..A6, B1..B6, C1..C5, D1..D5,
  E1..E4, più F (TLA+ skip).
- Step futuri che introducono **rendering polish features** simili
  (es. sticker preview, voice waveform full): ereditano il pattern di
  "5 micro-features in 1 ADR" dove appropriato.
- Step futuri che estendono **una delle 5 feature**:
  - Multi-pinned navigation: nuovo ADR (estende A6).
  - Multi-link picker: nuovo ADR (estende DB6).
  - Auto-clear status: nuovo ADR (estende DD3).
  - Sender color custom mapping: estensione `theme.toml` (no nuovo ADR).

**Non si applica a**:

- **Forward compose / send** (Step 21): già coperto da ADR-008
  (batch forward semantics). Step 33 è solo display.
- **Link detection in mention/hashtag** (`@user`, `#tag`):
  out-of-scope DB1; futuri step possono aggiungerli.
- **OSC 8 detection / opt-out**: nessuna detection runtime in Step 33.
  Tutti i link wrappati con OSC 8; terminali quirky accettano il
  rischio.
- **Pinned msg cache cross-session**: nessuna persistenza on disk.

## Cross-links

- [`../phase-2-behavioral/step33-polish.md`](../phase-2-behavioral/step33-polish.md) — statechart per ognuna delle 5 sub-feature + invarianti consolidati
- [`../phase-3-interactions/step33-polish-flow.md`](../phase-3-interactions/step33-polish-flow.md) — sequence diagrams per gli scenari principali
- [`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md) §Internal UI Messages (esteso con `OpenLinkMsg`, `PinnedMsgLoadedMsg`)
- Pipeline Step 33 (`development-pipeline.md`)
- [ADR-013 §debounce](ADR-013-search-debounce-and-stale-results.md) — pattern stale-result-drop riusato per `PINNED_STALE_DROP`
- [ADR-015 §D5 (whichkey)](ADR-015-command-palette-whichkey-help.md) — `gx` chord riusa whichkey registry (prefix `g` esistente)
- [ADR-017 §STALE_COMPLETION_DROP](ADR-017-chat-info-data-source.md) — pattern di drop di completion stale, riusato per pinned msg
- [ADR-018 §D2](ADR-018-responsive-layout-threshold-and-tab.md) — Compact mode rendering pinned bar (DA5)
- [ADR-019 §D5, §D7](ADR-019-theming-and-config-loading.md) — theme schema extension (4 nuovi color keys + palette)
- [ADR-020 §D2, §D8](ADR-020-mouse-support.md) — bbox invalidation trigger esteso con `PinnedMsgLoadedMsg`; KEYBOARD_PARITY esteso a link open (`gx` canonical)
- [`../tui-design.md`](../tui-design.md) §"Pinned bar", §"Link styling", §"Forward display", §"Status bar" (high-level user-facing)
