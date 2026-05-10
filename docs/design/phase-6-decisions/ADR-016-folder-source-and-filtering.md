# ADR-016: Folder source-of-truth, sidebar non-overlay, persistence, active-chat invariance, compact-mode skip

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 29 introduce la **folder sidebar** (`F` toggle, terzo pannello
a sinistra). Cinque sottodecisioni vanno prese insieme perché si
influenzano:

1. **Folder source-of-truth**: le cartelle vengono dai `DialogFilter`
   server-side di Telegram (sincronizzate fra dispositivi via
   `messages.getDialogFilters`), oppure sono cartelle "locally-defined"
   gestite dal client tuilegram nel suo file di config? La prima
   garantisce parità con altri client Telegram (mobile/desktop); la
   seconda è più semplice da implementare ma fragmenta l'esperienza.

2. **Sidebar è un overlay (Modal) o un pannello inline**? Gli step
   precedenti (26-28) hanno introdotto la primitive `Modal` Crush-style
   e una memoria utente `feedback_modal_charm.md` che impone il riuso
   per gli overlay. La sidebar ha però natura diversa:
   - è layout-impactful (occupa colonne, restringe gli altri pannelli),
   - è persistente (toggle on/off, non auto-dismiss),
   - non blocca il focus (l'utente può navigare alla chat list e poi
     tornare alla sidebar).

3. **Persistence di `selectedFolderID`** tra sessioni: salvata in
   `~/.config/tuilegram/config.toml` o reset a "All Chats" ad ogni
   avvio?

4. **Active chat invariance** sotto folder filter: se la chat aperta
   non è nella folder selezionata, la chat resta aperta o viene
   chiusa? Comportamenti possibili:
   - chiudi la chat (force consistency con la lista filtrata),
   - lascia la chat aperta (decoupling fra filter chat-list e chat
     attiva),
   - apri "All Chats" automaticamente per ripristinare la visibilità.

5. **Compact mode (< 100 cols)**: la sidebar consuma colonne; in
   compact mode il terminale ha già spazio limitato per chat list +
   conversation. Cosa fare se l'utente preme `F` in compact mode?

Bench su altri client Telegram:

- **Telegram Desktop**: cartelle server-side via `DialogFilter`,
  sidebar a sinistra (icone), persistence selezione fra sessioni,
  active-chat invariance (la chat aperta resta aperta anche se
  non in folder), in compact mode la sidebar è collassata a icone
  (no testo).
- **telegram-tui** (lonezor/tg): cartelle locally-defined in
  `~/.config/tg/folders`, no persistence selezione, single-panel
  compact-only.
- **Helix / Vim file managers (parallel pattern)**: pannelli inline
  (no Modal), persistence in stato di sessione, no compact-mode
  fallback particolare (se non c'è spazio, non disegnare).

## Decisione

**Quintuplice decisione consolidata in una sola ADR.**

### D1 — Folder source: server-side `DialogFilter`, fetch-once-cached

Le cartelle sono lette da Telegram via `messages.getDialogFilters`
(equivalente del campo `folders` nel `DialogsLoadedMsg` startup). Il
contenuto è un `[]ChatFolder` con campi `ID`, `Title`, `IncludedChats`.

Razionale:

- **Parità cross-device**: l'utente che ha "Work" / "Personal"
  configurati sul mobile vede le stesse cartelle nel TUI. Esperienza
  Telegram-native.
- **Source of truth singolo**: nessun rischio di drift fra config
  locale e server. Operazioni di crud sulle cartelle (out-of-scope
  Step 29) andrebbero comunque a Telegram.
- **Telegram fornisce già il dato**: `DialogFilter` è parte di
  `messages.getDialogFilters`, già consumato in `DialogsLoadedMsg`
  (Step 7). Step 29 lo espone in UI.

**Limitazioni accettate**:

- Telegram `DialogFilter` ha campi predicate (`include_unread`,
  `include_muted`, `include_bots`, ecc.) che permettono regole
  dinamiche. Per Step 29 assumiamo `IncludedChats` come lista
  esplicita; i predicati dinamici sono out-of-scope (la maggioranza
  degli utenti configura cartelle come `IncludedChats` esplicito,
  perché è il default di Telegram Desktop).
- Editing folder server-side richiede `messages.updateDialogFilter`:
  out-of-scope Step 29 (read-only). ADR futura.

**Alternativa considerata**: cartelle locally-defined in
`~/.config/tuilegram/config.toml`. Vantaggio: semplicità, nessun
binding a Telegram protocol per Step 29. Svantaggio: drift cross-device,
duplicazione di feature già fornita da Telegram, configurazione
ridondante per l'utente. Reietto in favore di server-side.

### D2 — Sidebar è un pannello inline, NON un Modal overlay

La folder sidebar è modellata come **pannello inline** (sibling di
`ChatListPanel` e `ConversationPanel`). NON usa la primitive `Modal`.
NON partecipa al lock `activeOverlay` di [ADR-015](ADR-015-command-palette-whichkey-help.md).

| Aspetto | Sidebar (pannello) | Overlay (Modal) |
|---------|---------------------|-----------------|
| Posizione | inline, occupa colonne | floating, lipgloss `Place` |
| Layout impact | restringe `chatlist_w` + `conv_w` | nessuno (overlay sopra) |
| Mutex `activeOverlay` | no | sì |
| Esc semantica | torna al pannello precedente, sidebar resta | chiude overlay |
| Persistence | toggle stato | sempre `Closed` quando non attivo |
| Coesistenza con overlay | sì (sidebar visible AND palette open è valido) | no (un solo overlay alla volta) |

**Razionale**:

- **Natura layout-impactful**: la sidebar modifica le larghezze degli
  altri pannelli. Modellata come overlay, dovrebbe essere
  full-screen (incompatibile) o sub-region (incompatibile con la
  primitive `Modal` floating-only).
- **Persistente by design**: l'utente apre la sidebar, naviga via
  Tab a un altro pannello, poi torna alla sidebar via Shift+Tab. Un
  overlay non supporta questo flow: Esc chiuderebbe l'overlay.
- **Nessuna violazione di `feedback_modal_charm.md`**: la regola
  "overlays riusano la primitive Modal" si applica agli **overlay**.
  La sidebar non è un overlay (è un pannello), quindi la regola non
  si applica.
- **Coerenza con `SearchInChat` (Step 27, [ADR-014](ADR-014-inline-search-bar-vs-modal.md))**:
  anche lì abbiamo argomentato che un sub-state inline NON è un
  overlay e NON usa la primitive `Modal`. Stesso pattern qui (con la
  differenza che SearchInChat è dentro `ConversationModel`, mentre
  la sidebar è di livello root).

**Conseguenza pratica**:

- `folderSidebarVisible: bool` è una variabile di livello root
  separata da `activeOverlay`.
- Aprire un overlay (palette, info, ecc.) NON chiude la sidebar.
- Aprire la sidebar (`F`) NON chiude un overlay attivo.
  - **Eccezione UX**: vedi [ADR-017 §D5](ADR-017-chat-info-data-source.md):
    quando l'overlay chat info è aperto, `F` è UX-consumed (no-op
    silenzioso) per evitare layout shift percepito. Decisione di
    consumo è sull'overlay attivo, non sulla sidebar.

### D3 — Persistence di `selectedFolderID`: NO (reset ad ogni avvio)

`selectedFolderID` è **NON persistito** tra sessioni. Ad ogni avvio
torna a `0` ("All Chats").

Razionale:

- **YAGNI per Step 29**: persistence richiede design del config
  (TOML schema, migrazione versioning), out-of-scope.
- **Telegram Desktop** persiste, ma in tuilegram il flusso tipico è
  CLI-orientato (apri, agisci, chiudi); meno valore nella persistence.
- **Triviale futuro**: refactor per persistere è ~3 linee in
  `internal/config/` quando lo step Theming + Config (Step 31)
  sarà implementato. Ad oggi (Step 29) `selectedFolderID` resta
  in memoria per la durata della sessione.
- **Coerente**: Step 29 introduce la feature; Step 31 introduce
  config persistence; la decisione di persistere `selectedFolderID`
  può essere presa lì insieme ad altri preferences (theme, send_key,
  emoji mode).

### D4 — Active chat invariance: la chat aperta resta aperta sotto folder filter

Quando l'utente seleziona una folder che NON include la chat
attualmente aperta (`activeChatID`), la chat **resta aperta** nel
pannello destro. La chat list a sinistra mostra solo le chat della
folder; la conversazione a destra è invariata.

Razionale:

- **Decoupling**: `activeChatID` (cosa è aperto a destra) è
  ortogonale al filter (cosa è visibile a sinistra). Forzare
  l'unione confonde l'utente ("perché la chat che stavo leggendo è
  scomparsa?").
- **Pattern Telegram Desktop**: stesso comportamento.
- **Sicurezza UX**: l'utente non perde mai involontariamente il
  contesto della chat aperta. Per chiudere la conversazione c'è
  l'azione esplicita (Esc dalla messages view).
- **Funzionalità ortogonali continuano a funzionare**: `i` (chat
  info) è basato su `activeChatID`, quindi funziona anche su una
  chat fuori folder. Verificato in `folders_chatinfo.tla` invariante
  `INFO_INDEPENDENT_OF_FOLDER`.

**Side-effect modellato**:

- Cursor della chat list: se la chat sotto cursore è filtered out →
  cursor reset a 0. Se l'`activeChatID` è filtered out → la chat
  list scroll-a per nasconderla, ma la conversazione resta aperta.

**Alternative reiette**:

- Force close chat aperta: rompe l'invariante "Esc chiude" (qui sarebbe
  un side-effect non triggerato dall'utente).
- Auto-switch a "All Chats": override la selezione esplicita
  dell'utente.

### D5 — Compact mode (< 100 cols): sidebar disabilitata, status-bar warning

In compact mode (Step 30 introdurrà il responsive layout, ma il
threshold < 100 cols è già documentato in
[`tui-design.md`](../tui-design.md) §"Responsive Behavior"), la
sidebar **non si apre**. Premere `F` mostra un warning nella
status-bar:

```
"Folders not available in compact mode (<100 cols). Resize terminal."
```

Razionale:

- **Real estate**: in 80×24 (sotto threshold), aprire un terzo
  pannello da 12 colonne lascia ~13 colonne per chat list e ~55 per
  conversation. Inutilizzabile.
- **Pattern industry**: file managers in compact mode collapsano i
  pannelli laterali (ranger, lf con `--multipane=0`).
- **Step 30 può raffinare**: future ADR può introdurre "icon-only
  sidebar in compact mode" (a la Telegram Desktop), ma è
  out-of-scope Step 29.
- **Non muta state**: `F` in compact mode è no-op completo (non
  aggiorna `folderSidebarVisible`). Status-bar message è feedback
  di chiarezza UX.

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **D1+D2+D3+D4+D5 (scelta)** | Parità Telegram, separazione di concerns sidebar-vs-overlay, YAGNI persistence, decoupling chat-attiva-vs-filter, gestione esplicita compact | Reset `selectedFolderID` a ogni avvio può sorprendere utenti che si aspettano persistence (mitigato: Step 31 può aggiungere persistence); impone "chat aperta indipendente da filter" che potrebbe non essere intuitivo per first-time user |
| Cartelle locally-defined (config TOML) | Implementazione semplice senza Telegram protocol | Drift cross-device, duplicazione di feature, configurazione manuale ridondante |
| Sidebar come Modal overlay full-screen | Riuso primitive Modal, coerenza con feedback_modal_charm | Non funziona: la sidebar deve essere inline, layout-impactful; un Modal full-screen nasconde chat list + conversation |
| Sidebar come Modal overlay sub-region (es. left strip) | Riuso primitive Modal | La primitive Modal è floating, non sub-region; estendere la primitive per supportare modalità inline è snaturarla |
| Persistence `selectedFolderID` in config.toml subito | Coerente con Telegram Desktop | Out-of-scope; richiede design config schema; Step 29 non ha lo step 31 ancora |
| Force close chat aperta su folder filter | Lista e conversazione coerenti | Rompe il principio "azioni utente espliciti chiudono la chat"; sorpresa UX |
| Auto-switch a "All Chats" se chat aperta non visibile | Lista e conversazione coerenti senza chiudere | Override la selezione esplicita dell'utente; può loop-are con il nuovo filter; UX confusing |
| Sidebar in compact mode mostra solo icone (a la Telegram Desktop) | Featureful in compact | Step 29 scope eccessivo; demandato a Step 30 (responsive) |
| Sidebar in compact mode si apre comunque (pannello strettissimo) | Coerenza | Inutilizzabile (5-6 cols per nome cartella tronca); peggio di non aprire |

## Conseguenze

- **Positive**:
  - **Parità Telegram-ecosystem**: l'utente che configura folders sul
    mobile vede la stessa configurazione nel TUI. Zero surprise.
  - **Architettura pulita**: sidebar-vs-overlay sono due tipi
    distinti di componente UI con regole diverse. Type-system
    naturale (`folderSidebarVisible: bool` separato da
    `activeOverlay: OverlayKind`).
  - **Coerenza con SearchInChat (ADR-014)**: stesso pattern "sub-state
    inline ortogonale, non Modal". Riduce il numero di concept
    architetturali distinti.
  - **Decoupling robust**: `activeChatID` indipendente da filter →
    tutte le feature basate su `activeChatID` (info, reply, edit,
    forward, ecc.) continuano a funzionare. Nessun bug latente da
    "chat aperta filtered out".
  - **UX safety**: compact mode warning evita layout broken silenzioso.
  - **Modello formale verificato in TLA+**: `folders_chatinfo.tla`
    modella la coesistenza sidebar-vs-overlay e l'invariante
    `ACTIVE_CHAT_INVARIANT`.
- **Negative**:
  - **Niente persistence Step 29**: utenti che abituati a Telegram
    Desktop possono restare sorpresi. Mitigato: Step 31 lo aggiunge
    triviale.
  - **`folderSidebarVisible` aggiunge una dimensione di stato al
    root model**: una bool in più. Aumento minimo della superficie.
  - **Compact mode warning è un'eccezione UX che richiede
    documentazione** (status-bar message + help section). Mitigato:
    una sola riga di docs.
  - **Server-side folders binding**: se Telegram cambia il formato
    dei `DialogFilter` (predicate dinamici diventano default vs
    `IncludedChats` esplicito), Step 29 va rivisto. Mitigato: campi
    dinamici sono già stabili in MTProto da anni.
- **Rischi**:
  - **Active chat invariance può confondere first-time user**: "ho
    selezionato Work, ma vedo ancora una chat di Mom a destra".
    Mitigato: il dot/badge che indica `activeChatID` è chiaro;
    l'utente capisce che la chat è "aperta", non "in folder".
  - **Folder con `IncludedChats` molto grande (>1000 chat)**: il
    re-filter `O(n)` per scan resta sotto 1ms ma può crescere se
    Telegram amplia il limite. Mitigato: building un index `Map<ChatID, []FolderID>`
    in startup è triviale (Step 29 può aspettare l'ottimizzazione).
  - **Drift fra `messages.getDialogFilters` e successive update**:
    se Telegram pusha `UpdateDialogFilter` (utente modifica folders
    sul mobile mentre tuilegram è aperto), Step 29 NON ascolta
    ancora questo update. Mitigato: refresh on next `DialogsLoadedMsg`
    cycle (al riavvio o reconnect). ADR futura per real-time sync.
  - **Compact mode threshold < 100 cols**: hard-coded in Step 29;
    Step 30 lo renderà parametrico.

## Scope

Questa ADR si applica a:

- **Step 29 — Folder sidebar**: prima introduzione delle cinque
  decisioni.
- Step futuri che introducono **pannelli inline aggiuntivi**
  (es. members list per gruppi, threads view): ereditano D2 (pannello
  inline non è overlay).
- Step futuri che introducono **stato UI persistibile** non legato
  agli overlay: ereditano D3 (default no-persistence; persistence
  via Step 31 config.toml).
- Step futuri che introducono **filter / view-mode** sulla chat list:
  ereditano D4 (decoupling activeChatID dal filter).
- Step 30 (responsive layout): erediterà D5 (compact mode behaviour
  per pannelli inline) e potrà raffinare il threshold / icon-only
  fallback.

**Non si applica a**:

- Editing delle cartelle Telegram (creare/rinominare/eliminare):
  out-of-scope Step 29; ADR futura quando si introduce
  `messages.updateDialogFilter`.
- Real-time sync di `UpdateDialogFilter` (utente modifica folder sul
  mobile mentre tuilegram è connesso): out-of-scope; refresh ai
  cycle major (riavvio/reconnect).

## Cross-links

- [`phase-2-behavioral/folder-sidebar.md`](../phase-2-behavioral/folder-sidebar.md) §Statechart, §Invarianti
- [`phase-3-interactions/folder-and-info-flow.md`](../phase-3-interactions/folder-and-info-flow.md) §1, §2 (filter preserva chat aperta)
- [`phase-4-concurrency/folders_chatinfo.tla`](../phase-4-concurrency/folders_chatinfo.tla) — invarianti `MUTEX_OVERLAYS_EXTENDED`, `ACTIVE_CHAT_INVARIANT`, `SIDEBAR_OVERLAY_ORTHOGONAL`, `SENTINEL_PRESENT`
- [`phase-1-context/domain-model.md`](../phase-1-context/domain-model.md) §`ChatFolder`
- [`phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md) §Internal UI Messages (esteso con `FolderToggleMsg`/`FolderCursorMsg`/`FolderSelectMsg`)
- [`phase-2-behavioral/ui-statechart.md`](../phase-2-behavioral/ui-statechart.md) §Responsive Layout States, §Folder Sidebar (Step 29)
- Pipeline Step 29
- [ADR-014](ADR-014-inline-search-bar-vs-modal.md) — pattern "sub-state inline, non Modal" (riusato qui per la sidebar)
- [ADR-015 §D3](ADR-015-command-palette-whichkey-help.md) — overlay mutex (sidebar **NON** partecipa)
- [ADR-017](ADR-017-chat-info-data-source.md) — chat info overlay (companion ADR di Step 29; gestisce `F`-during-info-open)
- [`tui-design.md`](../tui-design.md) §7 Folder Sidebar — wireframe canonical
- Memoria utente: `feedback_modal_charm.md` (regola overlay → Modal; non si applica alla sidebar perché non è overlay)
