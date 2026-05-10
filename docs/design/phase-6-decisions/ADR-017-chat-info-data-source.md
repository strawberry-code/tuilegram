# ADR-017: Chat info — Modal reuse + compact-right placement, cache-first data source, shared-counters stub, F-during-info, omit-vs-placeholder per ChatType

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 29 introduce l'**overlay chat info** (`i` toggle, scheda con
metadata della peer aperta). Cinque sottodecisioni vanno prese
insieme perché si influenzano:

1. **Primitive UI**: la chat info riusa la `Modal` Crush-style
   (Step 26, esteso con flag `compact` in Step 28 per which-key)
   oppure è un componente dedicato? La memoria utente
   `feedback_modal_charm.md` impone Modal per overlay; la chat info
   è un overlay (non un pannello inline come la sidebar di
   [ADR-016](ADR-016-folder-source-and-filtering.md) §D2), quindi
   la regola si applica. Ma la chat info ha layout particolare
   (anchored right, sub-screen) che richiede placement diverso da
   palette/help (full-screen) e da which-key (anchored bottom-right).

2. **Data source**: i campi della scheda (nome, username, phone,
   bio, status, counters) vengono dalla **cache locale** materializzata
   da `DialogsLoadedMsg` e dalle update real-time, oppure da una
   **RPC fresh** all'apertura (`users.getFullUser` per private,
   `channels.getFullChannel` per channel/group)? La prima è
   istantanea ma può essere stale; la seconda è always-fresh ma
   richiede spinner / loading state.

3. **Shared media counters**: Telegram fornisce contatori per
   shared media (Photos, Videos, Files, Links, ecc.) tramite
   `messages.getSearchCounters`. Step 29 li mostra? Se sì, fetch
   on-open o in startup? Se no, placeholder?

4. **F durante chat info open**: come argomentato in
   [ADR-016](ADR-016-folder-source-and-filtering.md), la sidebar
   tecnicamente NON è un overlay e quindi NON partecipa al lock
   `activeOverlay`. Quindi `F` durante chat info open potrebbe
   aprire la sidebar in background. Vogliamo questo behaviour, o
   l'overlay deve consumare `F`?

5. **Omit vs placeholder per ChatType**: una chat di tipo `Group`
   non ha username né phone. La sezione "Contact" deve essere
   omessa, oppure mostrata con placeholder `—`?

Bench su altri client:

- **Telegram Desktop** (chat info panel): cache-first con re-fetch
  background (mostra subito, aggiorna se cambia); shared counters
  fetched in lazy on-open; placement anchored-right occupando ~30%
  width; sezioni omesse per type (gruppi non hanno phone).
- **Telegram Mobile**: stesso pattern.
- **telegram-tui** (lonezor/tg): info come comando shell-like (no
  overlay), no shared counters, mostra solo campi cached.
- **Helix file-info preview**: cache-first con lazy expensive ops
  (file size for huge dirs).

## Decisione

**Quintuplice decisione consolidata in una sola ADR.**

### D1 — Modal primitive con `compact: true` + `placement: right`

L'overlay chat info usa la **primitive `Modal`** Crush-style con:

- `compact: true` (sub-screen, non full-screen) — flag già introdotto
  in [ADR-015 §D1](ADR-015-command-palette-whichkey-help.md) per
  which-key.
- **Nuova proprietà** `placement: right` (lipgloss `Place` con
  `horizontal=Right, vertical=Center`). Estensione minima dell'API
  della primitive: aggiunge un enum `Placement` con valori
  `{center, bottomRight, right}` (i primi due già usati da
  palette/help/which-key).

| Overlay | `compact` | `placement` | Larghezza |
|---------|-----------|-------------|-----------|
| Search globale (Step 26) | `false` | `center` | full-screen |
| Edit message (Step 19) | `false` | `center` | full-screen |
| Forward picker (Step 21) | `false` | `center` | full-screen |
| Confirm dialog (Step 20) | `false` | `center` | full-screen (ma piccolo) |
| Command palette (Step 28) | `false` | `center` | full-screen |
| Help (Step 28) | `false` | `center` | full-screen |
| Which-key (Step 28) | `true` | `bottomRight` | ~40 cols × ~10 rows |
| **Chat info (Step 29)** | **`true`** | **`right`** | **~32 cols × full-height** |

**Razionale**:

- **Coerenza con `feedback_modal_charm.md`**: l'overlay È un overlay
  (sovrapposto, modale, dismissed con Esc), quindi usa Modal.
- **Estensione minima dell'API**: aggiungere un valore enum a
  `Placement` (già esistente come campo della primitive Modal) è
  refactor di < 5 righe. Nessun nuovo componente.
- **Visual coherence con tui-design.md §6 "Chat Info (i)"**: il
  wireframe canonico mostra l'overlay sovrapposto al pannello
  conversazione, anchored right. Il `placement: right` realizza
  esattamente questo layout.
- **Differenza con sidebar**: la sidebar è un pannello inline che
  modifica le larghezze degli altri pannelli; la chat info è un
  overlay floating sovra la conversazione. Diverso `Placement`,
  stessa primitive.

**Alternative reiette**:

- Componente dedicato `ChatInfoPanel`: viola
  `feedback_modal_charm.md` (no one-off per overlay). Component
  sprawl.
- Modal full-screen per chat info: nasconderebbe la conversation,
  perdendo il contesto visivo. UX degradato.

### D2 — Data source: cache-first, lazy completion best-effort

L'overlay si apre **immediatamente dalla cache locale**. I campi
mancanti vengono completati via **lazy `tea.Cmd`** best-effort
(`fetchFullUserCmd` / `fetchFullChatCmd`).

```
on ChatInfoOpenMsg:
    if activeOverlay != none:           return         // mutex
    if activeChatID == nil:              return         // guard
    chatInfoTarget := activeChatID
    activeOverlay := chatInfo
    chatInfoCard := buildFromCache(activeChatID)   // sync, instant
    if needsCompletion(chatInfoCard):
        return spawnLazyFetch(activeChatID)         // tea.Cmd
    return                                          // no cmd

on ChatInfoCompletionMsg{chatID, fields}:
    if chatID != chatInfoTarget:        return         // STALE → drop
    chatInfoCard := merge(chatInfoCard, fields)
    cache.writeThrough(chatID, fields)               // benign even if stale
```

**Razionale**:

- **UX istantanea**: nessun spinner all'apertura. L'utente vede subito
  i dati noti.
- **Update progressivo**: campi mancanti (es. bio, photo metadata)
  appaiono in un secondo frame quando arrivano dalla RPC. Spinner
  inline accanto al campo singolo (`Bio: ⠧ loading...`), non
  blocking dell'intero overlay.
- **Drop-stale by `chatInfoTarget`**: il pattern è equivalente al
  `latestQueryID` / `latestPrefixID` di [ADR-013](ADR-013-search-debounce-and-stale-results.md) /
  [ADR-015](ADR-015-command-palette-whichkey-help.md), ma usa il
  `ChatID` come freshness key. È valido perché `chatInfoTarget` è
  set una volta sola per overlay-open e reset su close (uniqueness
  monotonicamente garantita all'interno della sessione overlay).
  Verificato in [`folders_chatinfo.tla`](../phase-4-concurrency/folders_chatinfo.tla)
  invariante `STALE_COMPLETION_DROP`.
- **Cache write-through anche se stale**: la RPC che ritorna su una
  chat che NON è più target (utente ha chiuso e riaperto su altra
  chat) può comunque popolare la cache per il prossimo open. Non è
  un'invariant requirement, è un'optimization opportunistic.

**Quando `needsCompletion` è TRUE**:

- Private chat: bio è `""` (mai fetched).
- Group / Channel: description è `""` (mai fetched).
- Bot: bot description è `""`.

**Quando è FALSE**:

- I campi sono già popolati (cached).
- ChatType è `SavedMessages` (no full profile da fetchare).

**Fail mode della completion**: la RPC fail è **silent**. Il bio
resta `—`; status-bar mostra dim message "Could not load full profile
(offline?)"; l'overlay NON mostra errori (degrada UX). L'utente può
chiudere e riaprire più tardi (Telegram ritry-erà).

**Alternative reiette**:

- **Always-fetch on-open**: spinner blocking → UX pesante per ogni
  apertura, anche per chat appena viste. Network spec consumed
  inutilmente per dati che la cache già ha.
- **Cache-only (no completion)**: bio mai mostrato per chat nuove
  (`""`). Funzionalità monca.
- **Eager fetch all dialogs in startup**: massivo, fa esplodere
  startup time per utenti con 1000+ chat. Out-of-scope.

### D3 — Shared media counters: stub `[?]` in Step 29

Telegram fornisce shared media counters via
`messages.getSearchCounters`, una RPC che ritorna count per filter
(`InputMessagesFilterPhotoVideo`, `InputMessagesFilterDocument`,
`InputMessagesFilterUrl`, ecc.).

**Step 29 mostra i counters come stub `[?]`**, NON fetched:

```
Shared Media   [?]
Shared Files   [?]
Shared Links   [?]
```

Razionale:

- **Out-of-scope per Step 29**: l'azione del fetch (e il caching dei
  counters) è una mini-feature a sé. Step 29 introduce overlay +
  layout; counters reali demandati a step futuro (probabile Step
  33+ o follow-up).
- **`?` è onesto**: piuttosto che mostrare `0` (errato) o ometterli
  (perde la sezione), `?` comunica chiaramente "dato non disponibile
  ancora".
- **Refactor futuro triviale**: aggiungere `fetchSharedCountersCmd`
  in [ADR-017](ADR-017-chat-info-data-source.md) §D2 stessa struttura
  della completion bio, riusa lo stesso pattern drop-stale.
- **Nessun blocking**: l'utente può comunque vedere identità + bio +
  contact senza counters.

**Variante considerata**: se i counters sono già nel `tg.UserFull` /
`tg.ChatFull` ritornati da `users.getFullUser` (alcuni sì come
`PinnedMsgID`, altri no), li usiamo. Step 29 mostra ciò che è in
cache; tutto il resto è `?`. Decisione operativa: il modeller passa
la responsabilità di "leggere quali counters sono fetchabili
gratuitamente" all'implementer (`tui-architect`).

### D4 — F durante chat info open: overlay UX-consume

Quando l'overlay chat info è **aperto** (`activeOverlay = chatInfo`)
e l'utente preme `F`:

- **Tecnicamente**: la sidebar NON partecipa al lock `activeOverlay`
  (vedi [ADR-016 §D2](ADR-016-folder-source-and-filtering.md)),
  quindi nulla impedisce a `F` di togglare la sidebar.
- **UX**: l'overlay **consuma** `F` (no-op silenzioso). L'utente
  deve chiudere l'overlay (`Esc`) prima di togglare la sidebar.

Razionale:

- **Layout shift visivo**: aprire la sidebar mentre l'overlay è
  ancorato a destra causerebbe un re-layout simultaneo:
  - chat list shrink per fare spazio alla sidebar,
  - conversation panel shrink,
  - overlay re-place (anchored right, ma su un viewport più stretto).
  - L'utente vede tre pannelli ridimensionarsi mentre è già
    focalizzato su un overlay. Disorienting.
- **Coerenza con altri overlay**: palette/help/which-key tutti
  consumano qualunque keystroke che non sia loro keybinding (D5
  pattern di [ADR-015](ADR-015-command-palette-whichkey-help.md)).
  Chat info eredita questo pattern.
- **Non è un'invariante TLA+**: è una scelta UX implementata nella
  state machine dell'overlay (vedi
  [`phase-2-behavioral/chat-info.md`](../phase-2-behavioral/chat-info.md)
  §E "Keybindings"). Non viola
  [ADR-016 §D2](ADR-016-folder-source-and-filtering.md): la sidebar
  resta non-overlay; è l'overlay attivo che decide di consumare la
  keystroke.

**Alternative reiette**:

- **F apre la sidebar in background**: layout shift confusing.
- **F chiude l'overlay e apre la sidebar**: due azioni in una key,
  viola principle-of-least-surprise.
- **F è ignorata silenziosamente senza feedback**: implementato come
  no-op senza message in status bar. Coerente con altre keystroke
  consumed dagli overlay.

### D5 — Omit vs placeholder per ChatType

Le sezioni del body sono mostrate selettivamente in base a
`ChatType`:

| Section | Private | Group | Channel | Bot | SavedMessages |
|---------|---------|-------|---------|-----|---------------|
| Identity | name + @user + dot | title + members | title + subs | name + @bot | "Saved Messages" |
| Contact (phone) | shown if Phone != "" | OMIT | OMIT | OMIT | OMIT |
| Profile (bio/desc) | shown if Bio != "" else "—" | shown if Description != "" else OMIT | shown if Description != "" else OMIT | shown if BotDescription != "" else "—" | OMIT |
| Counters | shown (`?` if not cached) | shown (`?` if not cached) | shown (`?` if not cached) | shown (`?` if not cached) | shown (only Saved is interesting) |

Decisione:

- **Sezioni strutturalmente non applicabili → OMIT** (non si rendono
  affatto, no separator). Es. phone in un gruppo.
- **Sezioni applicabili ma dato non ancora fetched → placeholder
  `—`** (con eventuale spinner inline `⠧ loading...` se completion
  in volo). Es. bio in private chat appena aperta.
- **Counters → sempre presenti** con `[?]` se non fetched (D3).

Razionale:

- **Omettere sezioni non applicabili evita visual clutter**: un
  gruppo non ha "Phone: —" — non ha telefono per definizione,
  mostrarlo con placeholder confonde.
- **Mostrare placeholder per dato in-fetch**: l'utente capisce che
  il dato sta arrivando, distingue da "non c'è" (omit).
- **Counters sempre presenti**: la sezione esiste per ogni chat type
  (anche savedMessages ha shared media implicito).

## Alternative considerate

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **D1+D2+D3+D4+D5 (scelta)**: Modal+compact+right placement / cache-first lazy / counters stub / F UX-consumed / omit non-applicable + placeholder pending | Coerenza con feedback_modal_charm; UX istantanea; stub onesto per counters; layout stabile; rendering pulito per type | Aggiunge un nuovo `placement: right` enum value alla primitive Modal (refactor minimo); counters `[?]` può sembrare incompleto a first-glance (mitigato da follow-up) |
| Componente dedicato `ChatInfoPanel` | Layout custom liberamente | Viola feedback_modal_charm; component sprawl; duplicazione di logic Modal (Esc handling, focus, animation) |
| Modal full-screen per chat info | Riusa `placement: center` esistente | Nasconde conversation; perde contesto visivo; UX degradato vs Telegram-native |
| Always-fetch on-open (no cache-first) | Always fresh data | Spinner blocking ogni apertura; overhead di rete; UX pesante per chat già viste |
| Eager-fetch full profile per tutti i dialog in startup | Sempre cached | Massivo overhead startup; spazio memoria; out-of-scope |
| Cache-only (no completion) | Semplice, no RPC | Bio mai mostrato per chat nuove; funzionalità monca |
| Counters reali fetched on-open in Step 29 | Featureful | Aumenta complessità Step 29 (un altro pattern drop-stale); demandato a step futuro |
| F apre sidebar in background durante chat info | Coerente con "sidebar non è overlay" | Layout shift simultaneo confusing; viola principio least-surprise |
| F chiude overlay e apre sidebar | Featureful | Doppia azione in una key; non vim/helix-aligned; user sorpresi |
| Mostrare TUTTE le sezioni con placeholder per type non applicabili | Layout consistente | Visual clutter ("Phone: —" su un gruppo è non senso); confonde l'utente |
| OMIT anche sezioni con dato pending (tutto silenzioso fino al fetch) | Layout pulito durante load | Utente non sa che il bio sta arrivando; sembra non essercene |

## Conseguenze

- **Positive**:
  - **Coerenza visuale**: tutti gli overlay usano `Modal` con
    `placement` differenziato. Una sola primitive, varianti minime
    (`compact`, `placement`). `feedback_modal_charm.md` rispettata.
  - **UX istantanea**: cache-first elimina spinner all'apertura.
  - **Pattern consolidato `drop-stale by ID`**: chatInfoTarget è la
    quarta applicazione (dopo typing timestamp Step 23, search
    queryID Step 26, whichKey prefixID Step 28). Pattern canonico
    del progetto.
  - **Modello formale verificato in TLA+**:
    `folders_chatinfo.tla` modella `STALE_COMPLETION_DROP`,
    `INFO_TARGET_COHERENCE`, `INFO_REQUIRES_OPEN_CHAT`.
  - **Refactor futuro triviale**: counters reali, real-time
    `UpdateUserFullUser`, photo rendering, ecc. si innestano nel
    pattern senza change strutturali.
  - **Layout stabile durante overlay**: F-consumed evita layout
    shift confuso.
- **Negative**:
  - **Nuovo enum `Placement.right`**: estensione API Modal. Refactor
    di pochi righe ma deve essere allineato in `Modal`.
  - **Stub `[?]` per counters**: utenti possono chiedersi "perché
    non vedo i numeri?". Mitigato: tooltip / status-bar message
    nella prima versione (out-of-scope; placeholder text è
    auto-explanatory).
  - **Cache-first ha rischio di mostrare dati stale (es.
    online status di 5 minuti fa)**: mitigato dal fatto che
    `UserStatusMsg` real-time refresh-a in background; per status
    > 30s il TUI mostra "last seen at HH:MM" che è già
    auto-explanatory.
  - **F-during-info è UX-decision implicita**: documentata in
    [`chat-info.md`](../phase-2-behavioral/chat-info.md) §E ma non
    è un invariante TLA+. Reviewer deve verificare l'implementazione
    in fase code-review.
  - **`chatInfoTarget` aggiunge una variabile al root model**: un
    `ChatID | nil` in più. Aumento minimo della superficie.
- **Rischi**:
  - **Bio fetch fail mode silent**: utente può non capire perché
    bio resta `—`. Mitigato: status-bar message dim ("Could not
    load full profile"); retry su prossimo open.
  - **Stale completion può rinfrescare la cache di una chat NON
    aperta**: benigno (write-through opportunistic), ma può sorprendere
    se la RPC arrivasse dopo lungo tempo (es. l'utente ha già visto
    altri 10 chat). Mitigato: il dato in cache è comunque aggiornato,
    non corrotto.
  - **Counters `?` può percepirsi come bug**: lo è (di scope, non
    di implementazione); follow-up issue obbligatorio per Step 33
    o successivo.
  - **Modal primitive estesa con `placement: right`**: se Modal non
    supporta ancora un dispatcher di placement, l'estensione richiede
    coordinamento con Step 26 (la primitive). Mitigato: il
    refactor è isolato, e l'implementer di Step 29 deve solo
    aggiungere un valore enum.

## Scope

Questa ADR si applica a:

- **Step 29 — Chat info overlay**: prima introduzione delle cinque
  decisioni.
- Step futuri che introducono **overlay con dati derivati da Telegram
  cache + completion lazy**: ereditano D2 (cache-first + drop-stale
  by ID).
- Step futuri che introducono **overlay con layout
  ancorato/specifico**: ereditano D1 (Modal + `placement` enum).
- Step futuri che introducono **shared content counters / aggregates**:
  ereditano D3 (stub-first se out-of-scope, real fetch in step
  successivo).
- Step futuri che introducono **overlay con conditional sections per
  tipo di entità**: ereditano D5 (omit vs placeholder rule).
- Step futuri che introducono **interazioni fra overlay e pannelli
  inline non-overlay (sidebar, search bar, ecc.)**: ereditano D4
  (overlay decide di consumare keystroke globali per evitare layout
  shift).

**Non si applica a**:

- Editing dei campi della chat info (cambio nome, set bio): out-of-scope
  Step 29; ADR futura quando si introduce `account.updateProfile` /
  `messages.editChatTitle`.
- Real-time `UpdateUserFullUser`, `UpdateChannelFull`: out-of-scope;
  refresh on next ChatInfoOpenMsg cycle (lazy completion ogni open).
- Avatar / photo rendering nell'overlay: out-of-scope; demandato a
  step "media in panel" futuro.

## Cross-links

- [`phase-2-behavioral/chat-info.md`](../phase-2-behavioral/chat-info.md) §Statechart, §Body layout, §Data sourcing, §Invarianti
- [`phase-3-interactions/folder-and-info-flow.md`](../phase-3-interactions/folder-and-info-flow.md) §3 (cache hit), §4 (cache miss), §5 (live update), §6 (stale), §7 (F UX-consume)
- [`phase-4-concurrency/folders_chatinfo.tla`](../phase-4-concurrency/folders_chatinfo.tla) — invarianti `INFO_TARGET_COHERENCE`, `INFO_REQUIRES_OPEN_CHAT`, `STALE_COMPLETION_DROP`, proprietà `EVENTUAL_INFO_CLOSE`, `EVENTUAL_COMPLETION`
- [`phase-1-context/domain-model.md`](../phase-1-context/domain-model.md) §`User`, `Chat`, `OnlineStatus`
- [`phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md) §Internal UI Messages (esteso con `ChatInfoOpenMsg`/`CloseMsg`/`CompletionMsg`)
- [`phase-2-behavioral/ui-statechart.md`](../phase-2-behavioral/ui-statechart.md) §Overlay State Machine (esteso con `chatInfo` formale)
- Pipeline Step 29
- [ADR-015 §D1, §D3](ADR-015-command-palette-whichkey-help.md) — Modal primitive con `compact`, mutex `activeOverlay` (chat info eredita)
- [ADR-016](ADR-016-folder-source-and-filtering.md) — folder sidebar (companion ADR di Step 29; sidebar non partecipa al mutex; F-during-info gestita dall'overlay come da §D4)
- [ADR-013](ADR-013-search-debounce-and-stale-results.md) — pattern monotonic counter + drop-stale (qui adattato a drop-stale by ChatID)
- [ADR-010](ADR-010-typing-ttl-strategy.md) — pattern timestamp + re-arm (precursore concettuale)
- [`tui-design.md`](../tui-design.md) §6 Overlays "Chat Info (`i`)" — wireframe canonical
- Memoria utente: `feedback_modal_charm.md` (overlay → Modal primitive; rispettata)
