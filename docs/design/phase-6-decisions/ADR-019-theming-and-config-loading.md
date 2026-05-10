# ADR-019: Theming + config loading — TOML, OS-aware path, embedded default, fail-soft, fsnotify hot-reload

**Stato**: accettato
**Data**: 2026-05-09

## Contesto

Lo Step 31 della pipeline introduce il **theming + config**: due file
TOML caricati al boot dell'app — `config.toml` (behavior switches) e
`theme.toml` (palette colori). Il default theme è **embedded nel
binario** (no file required at first boot). Tutti i colori usati dai
sub-modelli UI devono essere derivati dal tema (no literal hex
sparsi in `internal/ui/`).

Lo step continua il filone già anticipato da [ADR-004](ADR-004-theming-system.md)
("Sistema di theming con file TOML"), ma lo **estende** con decisioni
operative concrete che ADR-004 lasciava aperte:

- Path search XDG-aware (vs hard-coded `~/.config/tuilegram/`).
- Strategia di **merge** del tema utente sul default embedded
  (override-only per chiave vs replace-all).
- Comportamento esatto su **file mancante**, **TOML invalido**,
  **chiave sconosciuta**, **chiave colore con valore non parsabile**.
- Schema completo del `theme.toml` (lista canonical delle color keys).
- Schema iniziale del `config.toml` (quali behavior switches sono
  esposti in Step 31, quali sono deferred).
- Strategia di refactor del codice esistente: la palette in
  `internal/ui/styles/colors.go` oggi è **un set di package-level vars
  inizializzate da literal**. Va portata a un **theme accessor pattern**
  che permetta override boot-time.
- Hot-reload: in scope o deferred?
- Validation: type-check al parse o lazy?

Quindi questo ADR si comporta come una **specializzazione operativa**
di ADR-004 e introduce decisioni nuove (D2, D3, D4, D5, D6, D7, D8,
D9, D10) che ADR-004 non copriva. Non sostituisce ADR-004, lo
complementa.

Reference design (file-based config + theming in TUI app):

**Aggiornamento (2026-05-09)**: dopo conferma utente (post primo
design-pass), quattro decisioni operative sono state riviste:

- **D1**: lib TOML scelta esplicita: `pelletier/go-toml/v2`.
- **D2**: Windows `%APPDATA%/tuilegram/` ora **in scope**; usiamo
  `os.UserConfigDir()` (Go std lib) per coprire Linux/macOS/Windows
  con un'unica chiamata.
- **D9 (INVERTED)**: hot-reload **in scope per Step 31**, via
  `fsnotify/fsnotify` watcher su `theme.toml` con atomic swap; nuova
  spec TLA+ `phase-4-concurrency/theming.tla`.
- **D2 (clarification)**: `config.paths.theme` rimane **path
  assoluto only**, no tilde expansion.

Le sezioni qui sotto riflettono lo stato finale post-revisione.

- **btop**: ha file `.theme` sotto `~/.config/btop/themes/`,
  selezionabile via `config.cfg` (chiave `color_theme`); default theme
  embedded. Schema chiave-valore plain (no nesting).
- **lazygit**: `~/.config/lazygit/config.yml` con sezione `gui.theme`;
  YAML, nested. Hot-reload via SIGHUP (controverso: rallenta startup
  per fs watcher).
- **Helix**: TOML, `~/.config/helix/themes/<name>.toml` + `config.toml`
  separati. Default themes embedded. No hot-reload (richiede
  `:config-reload` esplicito).
- **Crush** (charm): TOML, `themes/<name>.toml` embedded; selezionato
  via env var `CRUSH_THEME`. Lipgloss-native. No hot-reload.
- **Telegram Desktop**: tema `.tdesktop-theme` zip; hot-reload sì
  (drag-drop). Out-of-scope per un TUI.
- **Neovim**: Lua, tema applicato a runtime via `:colorscheme`. Pieno
  hot-reload. Onere di complessità altissimo.

Aspettative utente (memoria `feedback_design_approach` + ADR-004):

- File `theme.toml` editabile a mano.
- Colori truecolor `#RRGGBB` (lipgloss downsamplea).
- Default theme **non richiede file esterno** (binario auto-contenuto).
- Restart-to-apply è accettabile per la v1 (consistente con btop /
  Helix / Crush).

## Decisioni

### D1 — Formato file: TOML (conferma di ADR-004)

`theme.toml` e `config.toml` sono **TOML 1.0**, con sezioni:

- `theme.toml`:
  - `[meta]` → `name`, `author`, `version`, `description`.
  - `[colors]` → tutte le 18+ chiavi colore (vedi D7) come stringhe
    `#RRGGBB`.
  - `[gradient]` → `start`, `end` (per la primitive `RenderGradient`).
- `config.toml`:
  - `[meta]` → `version` (per future migrazioni schema).
  - `[display]` → behavior switches UI (compact_threshold, ecc.; vedi
    D6).
  - `[paths]` → opzionale: override del path del tema (`theme = "..."`).
  - Sezioni future (`[network]`, `[notifications]`) sono deferred a
    step posteriori; non documentate in Step 31.

Razionale (consistente con ADR-004):

- TOML è più leggibile di YAML (no indent significativo) e supporta
  commenti (a differenza di JSON).
- **Lib scelta**: [`github.com/pelletier/go-toml/v2`](https://github.com/pelletier/go-toml)
  (post-revisione utente, 2026-05-09). Razionale:
  - API moderna basata su `encoding`-compatible `Unmarshal/Marshal`
    (familiare a chi conosce `encoding/json`).
  - Performance superiore a `BurntSushi/toml` su benchmark ufficiali
    (parsing ~2x più rapido, lower allocations).
  - Maintenance attiva (release frequenti, supporto TOML 1.0 completo).
  - Zero-dep tree (no transitive deps oltre stdlib).
- **AskUserQuestion run** in implementazione per confermare
  l'aggiunta della dep al `go.mod`. Pattern coerente con regola
  project rules.

### D2 — Path search: OS-aware via `os.UserConfigDir()` (Linux/macOS/Windows)

**Decisione (post-revisione 2026-05-09)**: usiamo
[`os.UserConfigDir()`](https://pkg.go.dev/os#UserConfigDir) della Go
std lib come **primary path source**. Questa funzione restituisce il
config dir convenzionale per l'OS corrente:

| OS | Path restituito da `os.UserConfigDir()` | Path completo `tuilegram/` |
|----|-----------------------------------------|---------------------------|
| Linux / BSD | `$XDG_CONFIG_HOME` se settato, altrimenti `$HOME/.config` | `$XDG_CONFIG_HOME/tuilegram/` o `$HOME/.config/tuilegram/` |
| macOS | `$HOME/Library/Application Support` | `$HOME/Library/Application Support/tuilegram/` |
| Windows | `%APPDATA%` (typically `C:\Users\<user>\AppData\Roaming`) | `%APPDATA%\tuilegram\` |

`os.UserConfigDir()` ritorna un errore solo se `$HOME` (Linux/macOS)
o `%APPDATA%` (Windows) non sono settati — caso edge gestito
fail-soft (vedi D4: cade a default embedded, no crash).

Algoritmo di risoluzione finale (boot-time, deterministico):

```
config_path  := first existing of:
  1. $TUILEGRAM_CONFIG_DIR/config.toml      (env override, escape hatch)
  2. os.UserConfigDir() + "/tuilegram/config.toml"
                                            (OS convention via Go std lib)

theme_path  := first existing of:
  1. config.paths.theme                     (if set — ABSOLUTE PATH ONLY,
                                            no tilde expansion, no env
                                            substitution; vedi sotto)
  2. $TUILEGRAM_CONFIG_DIR/theme.toml
  3. os.UserConfigDir() + "/tuilegram/theme.toml"
```

Razionale:

- **`os.UserConfigDir()` è il pattern canonical Go** per config files
  cross-platform. Documentato nella std lib, manutenuto, semantica
  stabile. Evita reinvenzione e gestione manuale di
  `$XDG_CONFIG_HOME` / `$HOME` / `%APPDATA%`.
- **Linux**: la funzione rispetta XDG Base Directory Spec (consulta
  `$XDG_CONFIG_HOME` con fallback a `$HOME/.config`). Friendly verso
  NixOS/Guix che usano `XDG_CONFIG_HOME` custom.
- **macOS**: la funzione ritorna `$HOME/Library/Application Support`
  (Apple HIG-conformant). Diverge dalla scelta originale ("usiamo
  `~/.config` anche su macOS perché è il pattern dei CLI tool"). La
  revisione preferisce **il pattern Go std**: utenti Homebrew sono
  abituati a entrambi i layout; `os.UserConfigDir()` è il default
  che meno sorprende sviluppatori Go (Crush, lazygit-go fork,
  charmbracelet/glow seguono lo stesso pattern).
- **Windows**: pieno supporto via `%APPDATA%/tuilegram/`. Step 31
  apre la porta a Windows-first contributors. Costo marginale
  (la funzione std lib gestisce tutto).
- **`$TUILEGRAM_CONFIG_DIR`**: escape hatch per dev/testing
  (override esplicito), pattern coerente con `TELEGRAM_APP_ID` in
  `internal/telegram/config.go`. Su tutti gli OS.
- **`config.paths.theme`**: **path assoluto SOLO**. No tilde
  expansion (`~/themes/x.toml` non viene espanso), no env var
  substitution (`$HOME/themes/x.toml` non viene risolto). Razionale:
  - **Simplicity**: un singolo path resolution algoritmo, zero
    edge case.
  - **Predicibilità**: lo stesso `theme = "..."` produce lo stesso
    risultato in qualsiasi ambiente (no dipendenza da env runtime).
  - **Edge case avoidance**: la tilde è una shell-feature; espanderla
    in Go richiede `os/user`, gestione `~user/...` per utenti
    diversi, comportamento divergente su Windows. `os.UserConfigDir()`
    + path assoluto utente = no ambiguità.
  - **Pratica**: utenti che vogliono path relativi possono usare
    symlink o `$TUILEGRAM_CONFIG_DIR`. Power-user comfort
    accettabile.
  - **Validation**: il loader verifica `filepath.IsAbs(path)`; se
    falso → log_warning, ignora il setting, fallback alla priority
    chain. Coerente con D5 fail-soft.

### D3 — Default theme: embedded via `embed.FS`, override-by-key merge

Il default theme è **embedded** nel binario tramite `//go:embed`
(directory `internal/config/themes/default.toml`). Il file è il
single source-of-truth per la palette baseline.

**Strategia di merge** quando l'utente fornisce `theme.toml`:

```
result.theme = parse(default_embedded)        // baseline 100% complete
for each key in parse(user_theme.toml):
    if key is recognized AND value is parsable:
        result.theme[key] = user_value         // override
    else:
        log_warning(key, reason)               // skip, keep baseline
```

**Conseguenza chiave**: il theme **risultante è SEMPRE total**
(funzione totale sull'insieme delle color keys conosciute). L'utente
non può accidentalmente "rompere" la palette omettendo una chiave.

Razionale:

- **Override-only merge** > replace-all merge: un utente che vuole
  cambiare solo il colore primary non deve copiare l'intero theme.
- **Embed > read-from-disk**: zero failure surface al boot
  (il file embedded esiste sempre, è valido per costruzione testato
  in CI con `go test ./internal/config/...`).
- **Determinismo**: il theme is `default_embedded` se l'utente non ha
  `theme.toml`, oppure `merge(default, user)` se ce l'ha. In nessun
  caso il theme è **partial**.

### D4 — Missing-file behavior: silent default (no bootstrap stub)

Se `config.toml` o `theme.toml` mancano da tutti i path D2:

- **No errore al boot**. L'app parte normalmente con default
  embedded (theme) o default values (config).
- **No bootstrap stub**. NON viene scritto un file di default sul
  disco al primo run. L'utente che vuole personalizzare crea il file
  manualmente.
- **Log line a `stderr` opzionale**: `[config] no theme.toml found,
  using default` — solo se `TUILEGRAM_DEBUG=1` (non spammoso di
  default).

Razionale:

- **Silent default** è il principio del "least surprise" per CLI:
  l'utente che lancia `tuilegram` per la prima volta non vuole essere
  bombardato di prompt o file appena creati.
- **No bootstrap stub** evita di sporcare `~/.config/tuilegram/`
  silenziosamente. Pattern coerente con Helix, lazygit.
- **Override-by-key (D3)**: l'utente può creare un `theme.toml`
  con UNA sola chiave (es. solo `[colors] primary = "#FF00FF"`) —
  non serve che sia "completo".
- **Deferred**: in futuro un comando `tuilegram --init-config` può
  scrivere uno stub commentato; out-of-scope Step 31.

### D5 — Invalid TOML / unknown key / bad value: fail-soft con warning

Tre livelli di errore di parsing:

| Livello | Esempio | Comportamento Step 31 |
|---------|---------|----------------------|
| **TOML syntax error** (es. parentesi non chiuse) | `[colors\nprimary =` | log_error a stderr; ignora il file utente; cade a default embedded; app parte normalmente |
| **Chiave sconosciuta** (es. `[colors] foo = "#FFF"`) | extra key non in schema | log_warning; chiave ignorata; altre chiavi del file applicate normalmente |
| **Valore non parsabile** (es. `primary = "not a hex"`) | hex malformato, fuori range | log_warning per la chiave; baseline default per quella chiave; altre chiavi applicate |

**Principio**: **never crash on user file**. Il theming è un
"enhancement layer"; un errore qui non deve impedire all'utente di
usare l'app.

Razionale:

- **Fail-soft** > fail-loud per file di config opzionali. Coerente
  con btop / lazygit / Helix.
- **Warning visibile**: log a stderr (non in TUI overlay — sarebbe
  troppo invadente). L'utente debug-ger trova subito il messaggio.
- **Per-key validation** (D10): permette al theme di essere
  "parzialmente valido" (es. typo su una chiave non rompe le altre).
- **Alternative considerate** (vedi tabella sotto): "fail-loud" è
  utile in produzione "config-as-code", ma per file utente locale
  è UX hostile.

### D6 — Scope di `config.toml` in Step 31

**In scope Step 31** (chiavi esposte):

- `[display] compact_threshold = 100` — soglia in cols per Wide vs
  Compact (parametrizza ADR-018 §D1, future-proof). Default 100.
- `[paths] theme = ""` — override del path del theme.toml.

**Deferred a step futuri** (NON in `config.toml` Step 31, ma chiavi
"prenotate" nello schema doc):

- `[display] message_density` (cozy/compact, Step 33+)
- `[display] show_typing_indicator` (Step 23 polish)
- `[display] show_read_receipts` (privacy switch, Step futuro)
- `[network] proxy = ""` (Step 32+)
- `[notifications] enabled = true` (Step 33+)
- `[keybindings.*]` (custom keybindings — feature complessa, Step >> 33)

Razionale:

- **Minimum viable surface** in Step 31: `compact_threshold` come
  proof-of-concept che il config influenza l'app (test plan dello
  step lo richiede). `paths.theme` serve a separare config dal theme.
- **No keybindings**: troppo invasivo, richiede ridisegno del
  dispatcher. Posticipato.
- **Schema versioning** (`[meta] version = 1`) prepara a future
  migrations.

### D7 — Schema theme: 18 color keys + gradient

Il `theme.toml` esporrà esattamente queste chiavi nella sezione
`[colors]` (mappate uno-a-uno alle vars di `internal/ui/styles/colors.go`):

| Chiave TOML | Mapping codice (Step 30) | Uso semantico |
|-------------|--------------------------|---------------|
| `primary` | `ColorPrimary` (`#7D56F4`) | accent generico, focused border, palette/help/whichKey, gruppi |
| `incoming` | `ColorIncoming` (`#38BDF8`) | messaggi incoming, canali, dot unread, receipt blue |
| `success` | `ColorSuccess` (`#50FA7B`) | dot online, connected status, selected (multi-select cursor) |
| `warning` | `ColorWarning` (`#FBBF24`) | bot border, status reconnecting, away dot |
| `error` | `ColorError` (`#FF5555`) | active chat border, errori, no-match search, modal warning border |
| `private` | `ColorPrivate` (`#FF79C6`) | chat private (color-coding) |
| `text` | `ColorText` (`#FAFAFA`) | testo principale, foreground default |
| `text_dim` | `ColorTextDim` (`#6B7280`) | timestamp, hint, label dim, system message |
| `surface` | `ColorSurface` (`#1E1E2E`) | background dark, contrast bg per palette/search/folder selected |
| `border` | `ColorBorder` (`#374151`) | bordi strutturali (chat list non focused, conversation, chat info dividers) |
| `search_secondary` | `ColorSearchSecondary` (`#4A3F6B`) | highlight search non-current (Step 27) |
| `search_inline_bg` | (literal `#2D2D40` in `conversation_search_view.go`) | bg della search bar inline (Step 27 — promoting al theme) |
| `button_fg` | (literal `#FAFAFA` in `components/button.go`) | foreground del button attivo (alias di `text`, ma esposto separatamente per sovrascrivere) |
| `button_bg` | (literal `#7D56F4` in `components/button.go`) | background del button attivo (alias di `primary`) |
| `button_disabled_fg` | (literal `#6B7280` in `components/button.go`) | foreground del button disabled (alias di `text_dim`) |
| `reaction` | (oggi `text_dim` via `styles/reactions.go`) | reaction default (esplicito per overridability) |
| `reaction_chosen` | (oggi `primary` via `styles/reactions.go`) | reaction chosen-by-me |
| `system_message` | (oggi `text_dim` italic) | system message text |

**Sezione `[gradient]`** (per `RenderGradient`):

| Chiave | Default | Uso |
|--------|---------|-----|
| `gradient.start` | `#FF60FF` (Dolly) | inizio gradiente titolo / Modal title |
| `gradient.end` | `#6B50FF` (Charple) | fine gradiente |

**Total**: 18 color keys + 2 gradient keys = **20 entries**.

Razionale per la lista:

- **Derivata empiricamente** dai grep di `lipgloss.Color` su
  `internal/ui/` (Step 30 codebase). Tutti i literal hex hardcoded
  identificati (button.go, conversation_search_view.go) sono
  promossi al theme.
- **Aliasing intenzionale** (es. `button_bg` ≡ `primary` di default):
  l'utente che vuole un button bg diverso può overrideare solo quello
  senza toccare `primary`. Coerente con Material/Tailwind design
  tokens dove "semantic" e "primitive" tokens sono separati.
- **No granularità eccessiva**: `chatlist_active_border` non è esposto
  separatamente perché coincide con `error` (active = rosso). Se in
  futuro si vuole dis-aliasare → ADR successivo.

### D8 — Refactor strategy: `Theme` struct + global accessor

Il pattern attuale (`var ColorPrimary = lipgloss.Color("#7D56F4")`)
è **incompatibile con override boot-time** perché le var sono
inizializzate prima di `main()`. Refactor minimale:

```
// internal/ui/styles/theme.go (NEW)
type Theme struct {
    Primary, Incoming, Success, Warning, Error, Private,
    Text, TextDim, Surface, Border, SearchSecondary,
    SearchInlineBg, ButtonFg, ButtonBg, ButtonDisabledFg,
    Reaction, ReactionChosen, SystemMessage lipgloss.Color
    GradientStart, GradientEnd colorful.Color
}

// internal/ui/styles/active.go (NEW)
var active *Theme = embeddedDefault()

func Active() *Theme { return active }
func SetActive(t *Theme) { active = t }   // boot-time only

// internal/ui/styles/colors.go (REFACTORED — wrappers retro-compatibili)
func ColorPrimary() lipgloss.Color { return active.Primary }
// ...
```

**Nota**: il file deliverable Go è scope dell'agent `tui-architect` /
implementazione, NON di questo ADR (design phase). Qui descriviamo
solo la **forma del refactor** che l'implementatore deve seguire.

Trade-off:

- **Vars → funzioni**: rompe la sintassi `styles.ColorPrimary` (no
  paren) → diventa `styles.ColorPrimary()`. ~30 call sites toccati
  (vedi grep). Refactor meccanico (regex), ma rumore in diff.
- **Alternativa "package var assigned at init"**: tenere
  `var ColorPrimary lipgloss.Color` ma assegnarlo in `func init()`
  da `active.Primary`. Problema: `active` deve essere settato
  PRIMA di `init()` di styles, ordering controllabile via
  `internal/config` import che inizializza per primo. Più fragile.
- **Decisione**: **funzioni accessor** (`styles.ColorPrimary()`).
  Esplicito, no init-order issue, costo runtime trascurabile (un
  dereference). Compromesso accettato.

### D9 — Hot-reload: **IN SCOPE** in Step 31 via `fsnotify` + atomic swap (INVERTED 2026-05-09)

**Decisione (post-revisione)**: hot-reload del **`theme.toml`** è
**in scope per Step 31**. Cambio di `config.toml` resta restart-only
(scope-limited; behavior switches sono boot-time-only per coerenza).

Implementazione (descrittiva, deliverable scope `tui-architect` /
`telegram-dev`):

- **Watcher**: [`github.com/fsnotify/fsnotify`](https://github.com/fsnotify/fsnotify) —
  cross-platform fs watcher (Linux inotify, macOS FSEvents/kqueue,
  Windows ReadDirectoryChangesW). De-facto standard Go per fs watching.
- **Goroutine lifecycle**: lanciata da `main()` post-bootstrap, prima
  di `tea.NewProgram(...).Run()`. Cleanup via `defer watcher.Close()`
  + `program.Send(QuitMsg)` linkage. Vedi statechart §H per i punti
  di start/stop.
- **Pipeline reload** (statechart in
  `phase-2-behavioral/theming-and-config.md` §J, sequence in
  `phase-3-interactions/theming-config-flow.md` §7):
  1. `fsnotify.Event` (Write o Create su `theme.toml`).
  2. Goroutine watcher legge bytes + `toml.Unmarshal` + `MergeTheme`
     (sopra `EmbeddedDefault`, identico al boot-flow).
  3. Se merge OK → `program.Send(ThemeChangedMsg{newTheme})`.
  4. Se parse/merge KO → `program.Send(ConfigWatcherErrMsg{err})`.
  5. `App.Update` riceve `ThemeChangedMsg` → `styles.SetActive(theme)` →
     ritorna `tea.WindowSizeMsg{}` per forzare re-render con i nuovi
     colori.
- **Atomic swap**: il merge avviene **interamente in goroutine
  watcher** prima di `Send`. `App.Update(ThemeChangedMsg)` esegue un
  singolo `styles.SetActive(theme)` (write atomico a un package
  global) — nessun render intermedio osserva uno stato torn. Vedi
  invariante `NO_TORN_RELOAD` in `theming.tla`.
- **Invalid file**: `ConfigWatcherErrMsg` triggera un warning visibile
  in status bar (Step 31: log su stderr; Step 33+ futuro: status
  bar inline) **e preserva il theme corrente**. Invariante
  `INVALID_PRESERVES_THEME`.
- **fsnotify coalescing**: editor moderni (vim, VSCode) generano
  multiple Write events vicini per un singolo save. fsnotify a livello
  OS coalesce; in Step 31 nessun debounce custom (la pipeline reload
  è single-inflight, drop di pendingFile se reload è già in corso).
  Pattern documentato in `theming.tla` action `FileEvent`.

Razionale (revisione):

- **Power-user feedback loop**: editing del theme con `vim` + reload
  istantaneo è la UX target di tools come xterm/alacritty
  (live-reload via SIGUSR1) e neovim. Step 31 lo abilita di base.
- **Concurrency cost mitigato**: la pipeline reload è single-inflight
  (`SINGLE_RELOAD_INFLIGHT` invariant) e l'unica mutazione cross-thread
  è un puntatore atomico in `styles` package. Modello TLA+
  (`theming.tla`) verifica `THEME_TOTAL`, `NO_TORN_RELOAD`,
  `MERGE_ATOMIC_ON_UPDATE`, `EVENTUALLY_APPLIED`.
- **Trade-off vs btop/Helix**: btop/Helix non hanno hot-reload; tools
  più moderni (Crush ha env-var-driven, neovim ha `:colorscheme`) si
  stanno muovendo verso interattività. tuilegram sceglie la
  posizione più ergonomica.
- **Cost contained**: una nuova dep (`fsnotify`), una nuova
  goroutine (lifecycle bound al `main`), un nuovo TLA+ spec
  (`theming.tla`). Non destabilizza nessun pattern esistente.

Out-of-scope di questa decisione (ancora):

- Hot-reload di `config.toml` (`compact_threshold`, etc.). Behavior
  switches restano boot-time. Razionale: alcuni richiedono full
  re-init di sub-models (es. `compact_threshold` cambia threshold
  responsive layout — semplice; ma future keys come `proxy` o
  `keybindings` sono complesse). Decisione conservativa, può essere
  estesa in step futuri.

### D10 — Validation: per-key fail-soft, type-checked al parse

Validation strategy:

- **Color values** (`#RRGGBB`): regex `^#[0-9a-fA-F]{6}$`. Strict.
  Valori non matching → log_warning + fallback a default per quella
  key.
- **Boolean values** (config bool flags): TOML native bool. Errore
  parse → log_warning + default.
- **Integer values** (es. `compact_threshold`): TOML int + range
  check (es. `compact_threshold ∈ [40, 400]`; out-of-range → default
  100 + warning).
- **String values** (es. `meta.name`): no validation, free text.
- **Validation atomic**: tutte le chiavi sono validate al parse-time.
  Nessuna validazione lazy (no "discover error at first render").
  L'app dopo il boot ha un `Theme` e `Config` già completamente
  validati e total.

Razionale:

- **Strict per-value** + **soft per-key** (D5) è il combo che
  massimizza usabilità: l'utente sa subito cosa è errato (warning
  on stderr al boot) senza essere bloccato.
- **Boot-time validation** > runtime: il render path non deve fare
  controlli (perf, complessità). Tutti i valori sono "garantiti
  validi" al raggiungimento di `tea.NewProgram(app).Run()`.

## Decisione su TLA+ (concurrency model)

**TLA+ spec NUOVA in Step 31**: [`theming.tla`](../phase-4-concurrency/theming.tla)
(post-revisione 2026-05-09, conseguenza dell'inversione D9).

Razionale:

- **Concurrency reale**: con D9 INVERTED, abbiamo due goroutine
  cooperanti — fsnotify watcher (W) e bubbletea message-loop (M) —
  che condividono `theme` (letto a ogni `View()` via
  `styles.Active()`). Il pattern "monotonic counter + drop-stale" di
  ADR-013/015 non applica (no debouncing user-side; coalescing è OS-side).
- **Pattern**: atomic publish via single TLA+ action `AtomicSwap`,
  preceduto da staging in `candidate` durante `Validating`. Nessun
  consumer osserva torn state.
- **Invarianti modellate** (full statement in `theming.tla`):
  - `THEME_TOTAL`: a ogni stato, `theme` è total (D3 garantisce che
    candidate è total per costruzione; AtomicSwap pubblica solo
    candidate).
  - `NO_TORN_RELOAD`: nessuno stato in cui `theme` è mid-merge
    (rendering è atomic-read del puntatore).
  - `MERGE_ATOMIC_ON_UPDATE`: il merge default+override è completato
    in `Validating` prima di Swapping; `AtomicSwap` pubblica il
    valore finale **una volta sola** in una singola transizione.
  - `WATCHER_BOUND_TO_LIFECYCLE`: `FileEvent` può sparare solo se
    `watcherAlive = TRUE` (start dopo Boot, stop prima di Shutdown).
  - `SINGLE_RELOAD_INFLIGHT`: `state` è una singola variabile in
    `{Idle, Reading, Validating, Swapping}`; un solo reload alla volta.
  - `INVALID_PRESERVES_THEME`: `ParseErr` lascia `theme` UNCHANGED
    (warning emesso, no swap).
- **Liveness**: `EVENTUALLY_APPLIED` — sotto fairness su
  `StartReload`/`ParseOk`/`AtomicSwap`, un `FileEvent` con file valido
  eventualmente porta `theme` al merged-result.

Le invarianti sono **anche** enunciate testualmente nel statechart
[`../phase-2-behavioral/theming-and-config.md`](../phase-2-behavioral/theming-and-config.md)
§E + §J (sub-statechart hot-reload), e validate via Go unit tests
(`loader_test.go` per la merge purity, `watcher_test.go` per
l'atomic swap con un `bytes.Buffer` mock di file).

**Pattern a runtime** (deliverable scope tui-architect):

- `styles.SetActive(theme)` è un singolo write a un package-global
  `*Theme`. Su Go AMD64/ARM64 un pointer write è atomico per ABI;
  su altre arch (improbabile) si può promuovere a `atomic.Pointer[Theme]`.
  Documentato come implementation note nello statechart §J.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **D1+...+D10 (scelta)** | Coerente con ADR-004, allineato a btop/Helix/Crush, fail-soft, no concurrency overhead | `styles.ColorPrimary()` rumore in diff vs var; deferimento hot-reload può sorprendere "power user" |
| YAML invece di TOML | Più diffuso | ADR-004 già pro-TOML; YAML più verboso e ambiguo (1 vs 1.0) |
| JSON invece di TOML | Universale, rigoroso | No commenti, scomodo a editare; ADR-004 lo aveva già scartato |
| Path hard-coded `~/.config/tuilegram/` (no XDG) | Più semplice | Hostile a NixOS/Guix che usano `XDG_CONFIG_HOME` custom; pattern non standard |
| Default theme NON embedded (read da `themes/default.toml` su disco) | Editabile dal manutentore senza ricompilare | Failure surface: file mancante = boot fail; ADR-004 già pro-embedded |
| Replace-all merge (user file deve essere completo) | Più semplice da implementare | UX hostile: utente deve copiare tutto per cambiare 1 colore |
| Bootstrap stub (`tuilegram` scrive `theme.toml` di default al primo run) | Discoverability ("c'è un file da editare!") | Sporca `~/.config/`; comportamento "magico" sgradito |
| Fail-loud su TOML invalido (exit 1) | Forza utente a fixare subito | App inusable per typo; UX hostile |
| Fail-loud su chiave sconosciuta | Cattura typo precoce | Rompe forward-compat: utente con tema "del futuro" (chiavi nuove) non può aprire app vecchia |
| **Hot-reload via fs watcher (scelta dopo revisione)** | UX moderna, edit-loop tight, pattern allineato a neovim/alacritty | Nuova concorrenza, nuova TLA+ spec; mitigato da single-inflight pipeline |
| Hot-reload via SIGHUP signal | Più semplice di fs watcher | UNIX-only (no Windows), nascosto agli utenti |
| No hot-reload (decisione originale, scartata in revisione) | Zero concorrenza | UX peggiore per power-user; restart ogni edit theme |
| Validation lazy (al primo uso del colore) | Zero parse cost | Errori scoperti tardivamente; render path complicato |
| Schema con 30+ chiavi (granularità massima) | Override fine-grained | Onere doc + manutenzione; D7 lista 18+gradient è il sweet spot |
| Schema con 8 chiavi (solo "primitives": fg, bg, accent, ecc.) | Minimale | Costringe a derivare colori "semantici" (active border, system msg, ecc.) hard-coded → riproduce il problema attuale |
| Theme accessor pattern via `context.Context` | Test-friendly | Onere passare context a ogni View; sproporzionato per uso boot-time-only |
| Theme accessor pattern via package var assigned in `init()` | Mantiene `styles.ColorPrimary` (no paren) | Init-order fragile (config init deve precedere styles init); il diff è più piccolo ma il rischio init-order > beneficio sintattico |
| Config con keybindings custom in Step 31 | Power-user appeal | Richiede ridisegno del dispatcher (Step 28 whichKey + Step 30 dispatch context-aware); deferito |
| `compact_threshold` non parametrizzato (resta hard-coded a 100) | Coerente con ADR-018 D1 | Step 31 test plan richiede "modifica config.toml → behavior changed". Espone almeno una key "live" |

## Conseguenze

- **Positive**:
  - **Nessun literal hex sparso in `internal/ui/`** dopo il refactor
    Step 31. Audit semplice via grep `lipgloss.Color\("#`
    (deve dare zero match fuori da `internal/config/themes/` e
    `internal/ui/styles/theme.go`).
  - **App self-contained**: zero file su disco richiesti per partire.
  - **Forward-compat schema**: `[meta] version = 1` permette
    migrazioni schema in step futuri senza rompere theme degli utenti.
  - **Fail-soft completo**: nessun input utente può crashare l'app.
  - **No concurrency new code**: caricamento sincrono boot-time;
    zero TLA+ aggiuntivo, zero rischio race.
  - **`compact_threshold` parametrizzabile**: ADR-018 D1 era
    "100 hard-coded"; ora è override-able. ADR-018 stesso lo
    anticipava ("Step 31 può rendere il threshold parametrico").
  - **Pattern riusabile**: la struttura `[meta]` + `[section]` +
    embed + merge è il template per tutti i file di config
    dell'app (futuri `keybindings.toml`, `proxy.toml`, ecc.).
  - **ADR-004 onorato e specializzato**: nessun re-design dalla
    fondamenta, solo concretizzazione di scelte aperte.

- **Negative**:
  - **Refactor `ColorX` → `ColorX()` esteso**: ~30 call sites in
    `internal/ui/`. Diff rumoroso ma meccanico. Mitigato:
    refactor-split agent può eseguire rename atomico.
  - **Hot-reload solo per `theme.toml`** (post-revisione D9): edit
    di `config.toml` (es. `compact_threshold`) richiede restart.
    Mitigato: behavior switches sono raramente toccati dopo setup
    iniziale; theme è il file ad alta-frequenza-edit.
  - **Concurrency overhead** (post-D9-inversion): nuova goroutine
    fsnotify, nuova TLA+ spec da mantenere aggiornata. Mitigato:
    pipeline reload single-inflight, modello formale verifica
    invarianti chiave (`THEME_TOTAL`, `NO_TORN_RELOAD`).
  - **No bootstrap stub**: utente "scopre" l'esistenza del theme
    solo da docs. Mitigato: README + comando `--init-config`
    futuro; bandiera `--print-default-theme` low-effort future.
  - **18 color keys**: l'utente che vuole tema chiaro deve
    overrideare 12+ chiavi (text, text_dim, surface, border,
    primary, ecc.). Mitigato: il merge by-key facilita; in
    futuro `themes/light.toml` embedded come "selectable preset"
    via `config.toml [theme] preset = "light"`.

- **Rischi**:
  - **Schema drift**: una nuova View introdotta in step futuro che
    aggiunge un literal hex senza promuoverlo a key di theme rompe
    l'invariante "nessun literal hex". Mitigato: lint rule
    (CI custom check via `grep`) o code-review pass;
    `code-reviewer` agent deve flaggarlo.
  - **TOML lib dependency**: se `BurntSushi/toml` o
    `pelletier/go-toml` non sono in `go.sum`, va aggiunta.
    AskUserQuestion in implementazione (project rule
    "Dependency Management"). Mitigato: candidati noti, low-risk.
  - **`embed.FS` requires Go 1.16+**: già garantito da `go.mod`
    (verificato — la pipeline è su Go 1.21+). Non-issue.
  - **Validation drift**: se un futuro step aggiunge una chiave
    di config con tipo nuovo (es. `time.Duration`), il pattern
    di validation D10 va esteso. Mitigato: pattern is "one
    validator per type", facilmente additivo.
  - **Test E2E theme reload**: senza hot-reload, il test di
    "modifica theme.toml → restart → colori cambiati" è manuale.
    Mitigato: scope dello step (test plan dello step lo elenca
    esplicitamente come test manuale).

## Scope

Questa ADR si applica a:

- **Step 31 — Theming + config**: prima introduzione operativa di
  D2..D10. Include refactor `internal/ui/styles/` per
  accessor-pattern.
- Step futuri che introducono **nuovi behavior switch** in
  `config.toml` (es. Step 33 polish, Step 32 network):
  ereditano D1, D2, D5, D6 (estendere schema), D10 (validation
  per-key).
- Step futuri che introducono **nuovi color tokens**: ereditano
  D7 (aggiungere chiave allo schema + a `default.toml` embedded;
  ADR successivo se la decisione è non-trivial).

**Non si applica a**:

- **Hot-reload di `config.toml`**: out-of-scope (D9 inverted copre
  solo `theme.toml`). Behavior switches restano boot-time.
- **Custom keybindings**: out-of-scope D6. Step >> 33.
- **Theme presets** (light/dark/solarized embedded): pattern
  derivabile da D3 ma deferred. ADR successiva quando richiesto.
- **Theme editor in-app** (modificare colori da TUI): out-of-scope
  totale (richiede UI flow nuovo).
- **Localization / i18n** (lingua, formati data): out-of-scope; non
  è theming, sarebbe `locale.toml` separato.

## Cross-links

- [ADR-004](ADR-004-theming-system.md) — decisione originale
  high-level (TOML + embedded default + no hot-reload). Specializzato
  da questa ADR.
- [ADR-018](ADR-018-responsive-layout-threshold-and-tab.md) §D1
  — `compact_threshold = 100` hard-coded; questa ADR lo rende
  parametrizzabile via `config.toml [display] compact_threshold`.
- [ADR-016 §D5](ADR-016-folder-source-and-filtering.md) — sidebar
  skip in compact mode (dipende da `compact_threshold`); transitivamente
  parametrizzabile.
- [ADR-017 §D2](ADR-017-chat-info-data-source.md) — `chatInfo`
  overlay con `placement: right`; il tema può overrideare il
  border color via `primary` o nuova key futura.
- [`../phase-2-behavioral/theming-and-config.md`](../phase-2-behavioral/theming-and-config.md)
  — statechart bootstrap → load → validate → apply.
- [`../phase-3-interactions/theming-config-flow.md`](../phase-3-interactions/theming-config-flow.md)
  — sequence diagrams (boot success, missing-file, invalid-toml,
  partial override, env override, deferred hot-reload).
- [`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md)
  §"Internal UI Messages" — `ConfigLoadedMsg`, `ThemeAppliedMsg`,
  `ThemeChangedMsg` (ATTIVO post-D9-inversion), `ConfigWatcherErrMsg`.
- [`../phase-4-concurrency/theming.tla`](../phase-4-concurrency/theming.tla)
  — TLA+ spec hot-reload (post-revisione 2026-05-09).
- Pipeline Step 31 — [`../development-pipeline.md`](../development-pipeline.md).
- Tui design canonical: [`../tui-design.md`](../tui-design.md)
  §Color (palette baseline).

## Dependencies

Step 31 introduce due nuove dipendenze Go (entrambe da approvare via
AskUserQuestion in implementazione, regola project "Dependency
Management"):

| Lib | Versione target | Ruolo | Razionale |
|-----|-----------------|-------|-----------|
| [`github.com/pelletier/go-toml/v2`](https://github.com/pelletier/go-toml) | latest stable (≥ v2.2) | Parser TOML 1.0 | D1 — performance, API moderna, manutenzione attiva |
| [`github.com/fsnotify/fsnotify`](https://github.com/fsnotify/fsnotify) | latest stable (≥ v1.7) | fs watcher cross-platform | D9 — hot-reload watcher per `theme.toml` |

Entrambe sono dep mature, low-risk, zero transitive non-stdlib.
