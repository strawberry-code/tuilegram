# Concurrency Model — TLA+

## Overview

Il modello TLA+ formalizza l'interazione concorrente tra i tre componenti principali di tuilegram:

1. **TUI Loop** (bubbletea) — thread principale, processa input e renderizza
2. **Telegram Client** (gotd/td) — goroutine separata, comunica con i server
3. **Message Channel** — `tea.Program.Send()`, bridge thread-safe tra i due

```
┌─────────────┐     sendChannel      ┌─────────────┐
│  Telegram    │ ──────────────────►  │  TUI Loop   │
│  Client      │   p.Send(msg)        │  (bubbletea) │
│  (goroutine) │                      │  (main)      │
└──────┬───────┘                      └──────┬───────┘
       │                                     │
       │ MTProto                              │ tea.Cmd
       ▼                                     ▼
┌─────────────┐                      ┌─────────────┐
│  Telegram    │                      │  Async Cmds │
│  Servers     │ ◄─────────────────── │  (goroutines)│
└─────────────┘     API calls         └─────────────┘
```

## Modello

### Variabili di stato

| Variabile | Tipo | Descrizione |
|-----------|------|-------------|
| `tuiState` | enum | Stato del TUI loop: idle, processing, rendering |
| `tuiMsgQueue` | sequence | Messaggi in attesa di processing |
| `uiMessages` | set | Messaggi attualmente visualizzati nella UI |
| `uiConnectionStatus` | enum | Stato connessione mostrato nella UI |
| `tgState` | enum | Stato del client Telegram: disconnected, connecting, connected, reconnecting |
| `tgRecvBuffer` | sequence | Messaggi ricevuti dal server, non ancora inviati al TUI |
| `sendChannel` | sequence | Buffer del canale `p.Send()` |
| `pendingOutgoing` | set | Messaggi in fase di invio |
| `deliveryStatus` | function | Mappa msgID → stato di consegna |
| `serverMessages` | set | Messaggi presenti sul server |

### Azioni

| Azione | Attore | Descrizione |
|--------|--------|-------------|
| `TGConnect` | TG Client | Inizia connessione |
| `TGConnected` | TG Client | Connessione stabilita |
| `TGDisconnected` | TG Client | Connessione persa |
| `TGReconnectAttempt` | TG Client | Tentativo di reconnect |
| `ServerNewMessage` | Server | Nuovo messaggio generato |
| `TGReceiveMessage` | TG Client | Riceve messaggio dal server |
| `TGForwardToTUI` | TG Client | Inoltra messaggio al TUI via p.Send() |
| `UserSendMessage` | User | Utente invia un messaggio |
| `ServerAckMessage` | Server | Server conferma ricezione |
| `TUIReceiveFromChannel` | TUI | Preleva messaggio dal channel |
| `TUIProcessMessage` | TUI | Processa messaggio (Update) |
| `TUIFinishProcessing` | TUI | Completa processing (View) |

## Proprietà verificate

### Safety (invarianti)

| Proprietà | Formula | Significato |
|-----------|---------|-------------|
| **NoLostMessages** | ∀ m ∈ serverMessages: m ∈ uiMessages ∨ m ∈ buffer ∨ m ∈ channel ∨ m ∈ queue | Ogni messaggio sul server è o nella UI o in transito |
| **ConnectionConsistency** | tgState = connected ∧ channel vuoto ∧ queue vuota ⟹ uiConnectionStatus = connected | Lo stato connessione nella UI riflette quello reale (quando non ci sono eventi in transito) |
| **DeliveryMonotonicity** | Lo stato di delivery non regredisce mai (sending → sent → delivered → read) | Gli status delle receipt sono monotonicamente crescenti |

### Liveness (proprietà temporali)

| Proprietà | Formula | Significato |
|-----------|---------|-------------|
| **EventualDelivery** | ∀ m ∈ serverMessages: ◇(m ∈ uiMessages) | Ogni messaggio raggiungerà la UI prima o poi |
| **EventuallyResponsive** | □◇(tuiState = idle) | Il TUI torna sempre in stato idle (niente deadlock) |
| **EventualReconnect** | □(tgState = reconnecting ⟹ ◇(tgState = connected)) | Dopo un disconnect, il sistema si riconnette |

### Fairness assumptions

- **Weak fairness** su: `TUIReceiveFromChannel`, `TUIProcessMessage`, `TUIFinishProcessing`, `TGForwardToTUI`, `TGConnected`
- Significato: se un'azione è continuamente abilitata, verrà eseguita prima o poi

## Come eseguire il model checking

### Prerequisiti
- TLA+ Toolbox o VS Code extension TLA+

### Configurazione TLC

```
CONSTANTS
    MaxMessages = 3
    MaxReconnects = 2

SPECIFICATION
    Spec

INVARIANTS
    TypeOK
    NoLostMessages

PROPERTIES
    EventualDelivery
    EventuallyResponsive
    EventualReconnect
```

### Valori raccomandati per model checking

| Parametro | Valore | Note |
|-----------|--------|------|
| `MaxMessages` | 3 | Sufficiente per trovare bug, tiene lo spazio degli stati gestibile |
| `MaxReconnects` | 2 | Verifica il ciclo reconnect senza esplosione combinatoria |

Con questi valori, TLC esplora ~10⁴-10⁵ stati in pochi secondi.

## Mapping al codice Go

| Concetto TLA+ | Implementazione Go |
|----------------|-------------------|
| `tuiState` | Stato implicito del loop `tea.Program.Run()` |
| `tuiMsgQueue` | Channel interno di bubbletea |
| `sendChannel` | `tea.Program.Send(msg)` (thread-safe, buffered) |
| `tgRecvBuffer` | Buffer locale nel dispatcher gotd/td |
| `tgState` | Stato implicito di `client.Run()` |
| `uiMessages` | `[]Message` nel `tea.Model` |
| `pendingOutgoing` | Messaggi con `DeliveryStatus = Sending` |

### Garanzie di thread-safety

| Operazione | Meccanismo | Garanzia |
|-----------|-----------|----------|
| `p.Send(msg)` | Channel interno bubbletea | Thread-safe per design |
| `tea.Cmd` result | Goroutine → channel interno | Thread-safe per design |
| `Model.Update()` | Single-threaded (solo TUI loop) | Nessun lock necessario |
| `Model.View()` | Single-threaded (solo TUI loop) | Nessun lock necessario |
| gotd/td dispatcher | Goroutine del client | I handler non accedono al Model |

**Regola fondamentale**: il `tea.Model` è accessibile SOLO dal TUI loop. Nessun'altra goroutine legge o scrive il Model. La comunicazione avviene esclusivamente via `p.Send()` (da Telegram goroutine) e `tea.Cmd` returns (da async commands).

## Extension modules

### Forward RPC (Step 21)

File: [`forward_picker.tla`](forward_picker.tla)

Modella il ciclo di vita del **forward picker overlay** e la sua interazione
con la goroutine `forwardMessageCmd`.

| Invariante | Significato |
|-----------|-------------|
| `RPC_ATOMICITY` | Al più una RPC `messages.forwardMessages` in volo per picker |
| `NO_ESC_DURING_RPC` | Esc è no-op durante `rpcInFlight` (vedi [ADR-007](../phase-6-decisions/ADR-007-overlay-in-flight-rpc.md)) |
| `SOURCE_SNAPSHOT` | I messaggi da forwardare sono congelati all'apertura del picker |
| `STATUS_CONSISTENCY` | Lo status bar riflette l'ultimo risultato RPC |

| Proprietà temporale | Significato |
|---------------------|-------------|
| `EVENTUAL_CLOSE` | Il picker torna sempre prima o poi a `closed` |
| `NO_STUCK_RPC` | Ogni RPC in volo produce un risultato (success o failure) |

Configurazione TLC raccomandata: `MaxSourceMsgs = 1` (Step 21) o `>= 2`
(Step 22, multi-select), `Chats = {1, 2, 3}`. Spazio di stati ~10³,
esplorato in <1s.

Cross-ref: [`../phase-2-behavioral/forward-picker.md`](../phase-2-behavioral/forward-picker.md),
[`../phase-3-interactions/forward-flow.md`](../phase-3-interactions/forward-flow.md).

### Multi-Select & batch actions (Step 22)

File: [`multi_select.tla`](multi_select.tla)

Modella la modalità multi-selezione `S` (set di MessageID) e l'apertura
ortogonale degli overlay batch (forward picker, confirm delete) che riusano
i cmd `forwardMessageCmd` / `deleteMessageCmd` con payload `[]MessageID`.

| Invariante | Significato |
|-----------|-------------|
| `MODE_COHERENCE` | `mode = multiSelect` ⇔ `|S| > 0` |
| `SELECTION_SCOPE` | `S` contiene solo ID della chat attiva |
| `SOURCE_SNAPSHOT` | snapshot frozen al momento dell'apertura overlay |
| `SNAPSHOT_FROZEN_DURING_RPC` | snapshot non muta durante `rpcInFlight` |
| `BATCH_ATOMICITY` | al più una RPC batch in volo (forward XOR delete) |
| `NO_DOUBLE_OVERLAY` | forward picker e confirm delete mutuamente esclusivi |
| `FALLBACK_OK` | con `S=∅`, `f`/`D` operano su `{cursor.id}` (snapshot non vuoto) |
| `NO_REPLY_EDIT_IN_MULTI` | `r`/`e` non possono creare stato (`multiSelect`, `S=∅`) |

| Proprietà temporale | Significato |
|---------------------|-------------|
| `EVENTUAL_EXIT` | il sistema torna sempre prima o poi a `browsing + overlay=none` |
| `NO_STUCK_BATCH` | ogni RPC batch in volo produce un risultato |

Configurazione TLC raccomandata: `MaxMessages = 5`, `Cursors = {1..5}`,
`Targets = {1, 2}`. Spazio di stati ~10⁴, esplorato in pochi secondi.

Cross-ref: [`../phase-2-behavioral/multi-select.md`](../phase-2-behavioral/multi-select.md),
[`../phase-3-interactions/multi-select-flow.md`](../phase-3-interactions/multi-select-flow.md),
[ADR-008](../phase-6-decisions/ADR-008-batch-forward-semantics.md),
[ADR-009](../phase-6-decisions/ADR-009-batch-delete-confirm.md).

### Typing indicator (Step 23)

File: [`typing.tla`](typing.tla)

Modella il ciclo di vita per-peer dello stato `typing` con TTL di 5s e
l'interazione tra `UpdateUserTypingMsg` (push da Telegram) e
`TypingTimeoutMsg` (pull da `tea.Tick`). La strategia "timestamp+re-arm"
è giustificata in [ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md).

| Invariante | Significato |
|-----------|-------------|
| `TYPING_TTL_BOUND` | `typing[p] = TRUE ⟹ clock − lastTypingAt[p] < TTL` (freschezza garantita) |
| `STALE_TICK_BENIGN` | Tick scaduto per peer `p` con `lastTypingAt[p]` rinfrescato dopo lo schedule è no-op (non clear-a lo stato) |
| `NO_FALSE_NEGATIVE` | Una `PeerTypes(p)` rende `typing[p]` TRUE nello stesso step (no race che nasconde update) |
| `PER_PEER_INDEPENDENCE` | Azioni su peer `p1` non mutano `typing[p2]` |
| `PENDING_TICKS_SANE` | Ogni tick pendente ha `scheduledFor >= 1` (sanity) |

| Proprietà temporale | Significato |
|---------------------|-------------|
| `EVENTUAL_CLEAR` | In assenza di nuovi `UpdateUserTyping`, ogni peer torna a Idle |
| `NO_STUCK_TYPING` | Nessun peer resta bloccato in Typing senza refresh |

Configurazione TLC raccomandata: `Peers = {1, 2}`, `TTL = 3`,
`MaxTime = 6`. Spazio di stati ~10³, esplorato in <1s.

Cross-ref: [`../phase-2-behavioral/typing-indicator.md`](../phase-2-behavioral/typing-indicator.md),
[`../phase-3-interactions/typing-flow.md`](../phase-3-interactions/typing-flow.md),
[ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md).

### Reactions store + system message immutability (Step 25)

File: [`reactions.tla`](reactions.tla)

Modella la sotto-componente **dati** del viewport per quanto riguarda
reactions e service messages: store per-message `reactions[id]`,
`text[id]`, `isService[id]`, e le azioni `NewMessage`,
`EmitReactionsUpdate`, `EmitTextEdit`, `ApplyHead`, `DeleteMessage`.
Verifica le invarianti che giustificano il rendering branch in
[`reactions-and-system.md`](../phase-2-behavioral/reactions-and-system.md)
e l'ADR-012.

| Invariante | Significato |
|-----------|-------------|
| `SNAPSHOT_NONNEG` | Ogni reaction count è `>= 0` (server è autoritativo, snapshot replace) |
| `SYSTEM_IMMUTABLE` | `isService[id]` non cambia mai dopo la creazione del messaggio |
| `SYSTEM_NO_REACT` | Il rendering predicate `ShouldRender(id) = FALSE` per ogni service message, anche se `reactions[id]` è non-vuoto (sanity guard) |
| `INDEPENDENT_FIELDS` | TextEdit non muta `reactions`; ReactionsUpdate non muta `text` (commutatività) |
| `REACTIONS_ORDERED` | `reactions[id]` è ordinata per `count desc, emoji asc` (invariante di rendering, garantita dal convert) |
| `DELETE_PROPAGATES` | Dopo `DeleteMessage(id)`, `ShouldRender(id) = FALSE` (no orphans) |

| Proprietà temporale | Significato |
|---------------------|-------------|
| `EVENTUAL_CONVERGENCE` | In assenza di nuovi update, `pending` si svuota (no infinite loop) |
| `NO_LOST_UPDATE` | Ogni update emesso lascia un effetto nel store (o il messaggio è stato cancellato) |

Configurazione TLC raccomandata: `MessageIDs = {1, 2}`,
`Emojis = {"a", "b"}`, `MaxCount = 2`, `MaxUpdates = 4`. Spazio di stati
~10⁴, esplorato in <5s.

Cross-ref: [`../phase-2-behavioral/reactions-and-system.md`](../phase-2-behavioral/reactions-and-system.md),
[`../phase-3-interactions/reactions-flow.md`](../phase-3-interactions/reactions-flow.md),
[ADR-012](../phase-6-decisions/ADR-012-reactions-storage-and-system-detection.md).

### Search overlay — debounce + stale-result drop (Step 26)

File: [`search.tla`](search.tla)

Modella il ciclo di vita dell'overlay search globale (`/`) e l'interazione
fra tre produttori concorrenti: keystrokes utente, tick di debounce
`tea.Tick(300ms)`, e risultati RPC `messages.searchGlobal`. Lo schema di
freschezza è un **monotonic counter** (`latestQueryID`) bumped ad ogni
keystroke e ad ogni `Close`; sia tick stale sia risultati RPC stale sono
droppati al fire/return time. La motivazione formale è in
[ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md).

| Invariante | Significato |
|-----------|-------------|
| `MONOTONIC_QUERYID` | `latestQueryID` è monotonicamente non-decrescente (mai reset, mai decremento) |
| `STALE_DEBOUNCE_BENIGN` | Un debounce tick con `qID < latestQueryID` non spawna RPC (no-op) |
| `STALE_RESULT_DROP` | Un `SearchResultMsg` con `qID != latestQueryID` non muta lo stato visibile (`appliedQueryID`) |
| `CLOSE_INVALIDATES_INFLIGHT` | Dopo `Close`, qualunque RPC pendente ha `qID < latestQueryID` → il result sarà droppato anche se arriva successivamente |
| `AT_MOST_ONE_FRESH_RPC` | Al più una RPC in volo ha `qID = latestQueryID`; le altre sono stale ma benigne |
| `EMPTY_QUERY_NO_RPC` | Una query vuota non emette mai `MessagesSearchGlobal` (filtro client-side) |

| Proprietà temporale | Significato |
|---------------------|-------------|
| `EVENTUAL_QUIESCENCE` | In assenza di nuove keystroke, ogni RPC pendente ritorna e l'overlay raggiunge stato stabile (no pending tick, no in-flight RPC) |
| `RESPONSIVE_CLOSE` | `Close` è sempre abilitato da `open` (deroga ADR-007: `Esc` non aspetta la RPC) |

Configurazione TLC raccomandata: `MaxKeystrokes = 3`, `MaxRpcLatency = 2`.
Spazio di stati ~10⁴, esplorato in <5s.

Cross-ref: [`../phase-2-behavioral/search-overlay.md`](../phase-2-behavioral/search-overlay.md),
[`../phase-3-interactions/search-flow.md`](../phase-3-interactions/search-flow.md),
[ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md),
[ADR-007](../phase-6-decisions/ADR-007-overlay-in-flight-rpc.md) (deroga),
[ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md) (pattern timestamp/counter + re-arm).

### Search in chat — inline bar + re-index (Step 27)

File: [`search_in_chat.tla`](search_in_chat.tla)

Modella il ciclo di vita della barra di **ricerca locale** nella
conversazione attiva (`Ctrl+F`) e l'interazione concorrente fra tre
produttori che mutano la lista messaggi (e quindi l'indice di ricerca)
mentre la barra è aperta: keystroke utente nella barra
(`QueryChange(q)`), `NewMessageMsg` push da Telegram
(`NewMessageArrive`), `LoadMoreMsg` da `loadHistoryCmd`
(`LoadMoreArrive`). A differenza di `search.tla`, **non c'è RPC, no
debounce, no stale-result drop**: tutte le transizioni sono sincrone
nello stesso `Update` cycle. La motivazione formale è in
[ADR-014](../phase-6-decisions/ADR-014-inline-search-bar-vs-modal.md).

| Invariante | Significato |
|-----------|-------------|
| `MATCH_IDENTITY_PRESERVED_NEW` | `NewMessageArrive` (append) non cambia l'identità (msgID) del match al `currentIdx` |
| `MATCH_IDENTITY_PRESERVED_LOADMORE` | `LoadMoreArrive` (prepend) shift-a `currentIdx += len(newMatches)`, preservando l'identità del match corrente |
| `NO_PHANTOM_MATCH` | Ogni entry in `matches` ha il `msgID` presente in `index` (no orphan match) |
| `SYSTEM_NOT_INDEXED` | Messaggi con `isService = TRUE` non sono mai in `index` ne in `matches` |
| `CURSOR_BOUNDED` | `0 <= currentIdx < |matches|` se `|matches| > 0`; `currentIdx = 0` se `|matches| = 0` |
| `INDEX_CONSISTENT_WITH_MESSAGES` | Per ogni `id \in index`, esiste un msg corrispondente in `messages` con `isService = FALSE` |
| `QUERY_EMPTY_NO_MATCHES` | `query = "" ⟹ matches = <<>>` |
| `INACTIVE_CLEAN` | `active = FALSE ⟹ index, matches, query, currentIdx tutti zero/empty` |
| `LOCAL_ONLY` | Nessuna RPC viene mai spawned (encoded structurally: nessuna variabile rpc-related) |

| Proprietà temporale | Significato |
|---------------------|-------------|
| `EVENTUAL_CLOSE` | La barra prima o poi torna `inactive` (sotto fairness su input utente) |
| `NO_STUCK_REINDEX` | Ogni `NewMessage`/`LoadMore`/`MessageDelete` viene processato (mai droppato silenziosamente) |

Configurazione TLC raccomandata: `MaxMessages = 4`, `MaxKeystrokes = 2`,
`MaxNewMsgs = 2`, `MaxLoadMore = 1`. Spazio di stati ~10⁴, esplorato
in pochi secondi.

Cross-ref: [`../phase-2-behavioral/search-in-chat.md`](../phase-2-behavioral/search-in-chat.md),
[`../phase-3-interactions/search-in-chat-flow.md`](../phase-3-interactions/search-in-chat-flow.md),
[ADR-014](../phase-6-decisions/ADR-014-inline-search-bar-vs-modal.md).

### Which-key + overlay mutex (Step 28)

File: [`whichkey.tla`](whichkey.tla)

Modella il timer 300ms del **which-key prefix-disambiguation** overlay
e la **mutua esclusione** fra i tre overlay UI-only introdotti da
Step 28 (palette, which-key, help) + gli overlay esistenti
(search/edit/forward/confirm/chatInfo, abstract come `"other"`). Lo
schema di freshness è un **monotonic counter** (`latestPrefixID`)
bumped ad ogni prefix press, chord resolution, cancel; sia tick stale
sia chord-after-cancel sono droppati al fire-time. La motivazione
formale è in [ADR-015](../phase-6-decisions/ADR-015-command-palette-whichkey-help.md).

| Invariante | Significato |
|-----------|-------------|
| `MUTEX_OVERLAYS` | Al più un overlay attivo alla volta (palette XOR whichKey XOR help XOR other XOR none); coerente con `state = "PrefixPending" => activeOverlay = "none"` |
| `WHICHKEY_VISIBILITY_CONSISTENT` | `state = "Visible" <=> activeOverlay = "whichKey"` (no inconsistency tra rendering e state-machine) |
| `MONOTONIC_PREFIXID` | `latestPrefixID` non decresce mai (counter monotono process-wide) |
| `STALE_TICK_BENIGN_WHICHKEY` | Un tick fire con `t.prefixID < latestPrefixID` non apre l'overlay (no-op) |
| `FAST_CHORD_NO_OVERLAY` | Se un chord risolve PRIMA del tick fire, `activeOverlay` resta `"none"` per tutto il path (l'overlay non viene mai mostrato) |
| `PREFIX_PRESENCE_CONSISTENT` | `activePrefix = "none" <=> state = "Idle"` (no orphan prefix state) |

| Proprietà temporale | Significato |
|---------------------|-------------|
| `EVENTUAL_RESOLUTION` | Ogni `state = "PrefixPending"` raggiunge prima o poi `Idle` o `Visible` (nessun prefix bloccato) |
| `EVENTUAL_OVERLAY_CLOSE` | Ogni overlay attivo prima o poi torna a `none` (sotto fairness su input) |
| `RACE_CONVERGENCE` | Tick fire e chord press concorrenti convergono allo stesso stato finale (`Idle`, `activeOverlay = none`, action eseguita) indipendentemente dall'ordine |

Configurazione TLC raccomandata: `PrefixKeys = {"g", "z"}`,
`MaxPrefixPresses = 2`, `MaxOpenAttempts = 2`. Spazio di stati ~10⁴,
esplorato in <5s.

Cross-ref: [`../phase-2-behavioral/command-palette-help-whichkey.md`](../phase-2-behavioral/command-palette-help-whichkey.md),
[`../phase-3-interactions/whichkey-timing-flow.md`](../phase-3-interactions/whichkey-timing-flow.md),
[ADR-015](../phase-6-decisions/ADR-015-command-palette-whichkey-help.md),
[ADR-013](../phase-6-decisions/ADR-013-search-debounce-and-stale-results.md) (pattern counter + drop-stale),
[ADR-010](../phase-6-decisions/ADR-010-typing-ttl-strategy.md) (pattern timestamp + re-arm, precursore).

### Folder sidebar + chat info overlay (Step 29)

File: [`folders_chatinfo.tla`](folders_chatinfo.tla)

Modella la coesistenza fra il **pannello inline `FolderSidebar`**
(non-overlay, fuori dal lock `activeOverlay`) e l'**overlay
`ChatInfo`** (overlay full-fledged, dentro il lock). Verifica le
nuove variabili condivise (`folderSidebarVisible`, `selectedFolderID`,
`chatInfoTarget`) e l'interazione fra tre produttori concorrenti:
keystroke utente, Telegram update push (`UserStatusMsg`,
`ChatUpdateMsg`), e tea.Cmd async (`ChatInfoCompletionMsg` da
`fetchFullUserCmd`). Il pattern di freshness per il completion è
**drop-stale by `chatInfoTarget` (ChatID)**, equivalente al
`latestQueryID` / `latestPrefixID` di ADR-013 / ADR-015 (la freshness
key è un ChatID invece di un counter, ma uniqueness è garantita
dalla semantica set-once-per-open-reset-on-close).

| Invariante | Significato |
|-----------|-------------|
| `MUTEX_OVERLAYS_EXTENDED` | Estende `MUTEX_OVERLAYS` di whichkey.tla con `chatInfo`; un solo overlay attivo (chatInfo XOR palette XOR whichKey XOR help XOR other XOR none) |
| `INFO_TARGET_COHERENCE` | `chatInfoTarget != nil ⟺ activeOverlay = "chatInfo"` (variabile populated solo quando overlay open) |
| `INFO_REQUIRES_OPEN_CHAT` | `activeOverlay = "chatInfo" ⟹ activeChatID != nil` (guard di apertura) |
| `STALE_COMPLETION_DROP` | `ChatInfoCompletionMsg{chatID}` con `chatID != chatInfoTarget` non muta il rendered card |
| `FOLDER_SELECTION_PRESERVED` | `FolderToggleMsg` non cambia `selectedFolderID` (toggle preserva selezione) |
| `ACTIVE_CHAT_INVARIANT` | `FolderSelectMsg` non muta `activeChatID` (filter agisce solo su chat list) |
| `INFO_INDEPENDENT_OF_FOLDER` | `ChatInfoOpenMsg` succeede indipendentemente dalla folder selezionata (legge da `activeChatID`) |
| `SIDEBAR_OVERLAY_ORTHOGONAL` | `folderSidebarVisible = TRUE ∧ activeOverlay != "none"` è uno stato VALIDO (sidebar non è overlay) |
| `SENTINEL_PRESENT` | `0 ∈ Folders ∧ FolderMembers[0] = Chats` (pseudo-folder "All Chats" sempre disponibile) |
| `FILTER_SYNC` | re-filter atomico nello stesso `Update` step di `FolderSelectMsg` |

| Proprietà temporale | Significato |
|---------------------|-------------|
| `EVENTUAL_INFO_CLOSE` | Ogni `chatInfo` overlay attivo torna prima o poi a closed (sotto fairness su input utente) |
| `EVENTUAL_COMPLETION` | Ogni `fetchFullUserCmd` pendente prima o poi delivery-ra un risultato (applicato se fresh, droppato se stale) |

Configurazione TLC raccomandata: `Chats = {c1, c2, c3}`,
`Folders = {0, 1, 2}`, `FolderMembers = (0:>Chats, 1:>{c1}, 2:>{c2})`,
`MaxKeyPresses = 8`, `MaxRpcLatency = 3`. Spazio di stati ~10⁴-10⁵,
esplorato in <5s.

Cross-ref: [`../phase-2-behavioral/folder-sidebar.md`](../phase-2-behavioral/folder-sidebar.md),
[`../phase-2-behavioral/chat-info.md`](../phase-2-behavioral/chat-info.md),
[`../phase-3-interactions/folder-and-info-flow.md`](../phase-3-interactions/folder-and-info-flow.md),
[ADR-016](../phase-6-decisions/ADR-016-folder-source-and-filtering.md),
[ADR-017](../phase-6-decisions/ADR-017-chat-info-data-source.md),
[ADR-015 §D3](../phase-6-decisions/ADR-015-command-palette-whichkey-help.md) (mutex pattern, esteso),
[ADR-014](../phase-6-decisions/ADR-014-inline-search-bar-vs-modal.md) (sub-state inline non-overlay, riusato per la sidebar).
