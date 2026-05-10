# ADR-012: Reactions storage shape + System message detection

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 25 introduce due feature di rendering correlate:

1. **Reactions** sotto i bubble (`👍 3  ❤️ 2  😂 1`).
2. **System messages** centrati e dimmati (`── Alice joined ──`).

Entrambe richiedono decisioni di shape sui dati di dominio prima
dell'implementazione:

### Decisione A — Reactions storage

`tg.MessageReactions` è un union MTProto che porta una lista di
`tg.ReactionCount{Reaction: tg.ReactionClass, Count: int, Chosen: bool}`.
Le varianti di `tg.ReactionClass` sono:

- `tg.ReactionEmoji{Emoticon: string}` — reazione standard (👍, ❤️, ...).
- `tg.ReactionCustomEmoji{DocumentID: int64}` — emoji custom premium.
- `tg.ReactionEmpty` (raro, placeholder).

Il dominio interno (`domain.MessageMedia`,
[phase-5-data/domain-types.md](../phase-5-data/domain-types.md)) ha già
definito:

```go
type Reaction struct {
    Emoji      string
    Count      int
    ChosenByMe bool
}
type Message struct {
    // ...
    Reactions []Reaction
    // ...
}
```

Le opzioni rilevanti per lo **storage shape** del campo `Reactions`:

- **Opzione A1**: `[]Reaction` slice ordinata (count desc, emoji asc
  tie-break) — già fissata in domain-types.md ma mai motivata
  formalmente.
- **Opzione A2**: `map[string]int` chiave-emoji → count, con un set
  separato `chosenByMe map[string]bool`.
- **Opzione A3**: `*ReactionsBlock` con campi pre-calcolati (totale,
  topN slice, chosen set).

### Decisione B — System message detection

In `gotd/td` la gerarchia `tg.MessageClass` ha tre concrete:

- `tg.Message` — messaggio ordinario (text/media/reactions/ecc.).
- `tg.MessageService` — service message con `Action tg.MessageActionClass`
  (~30 varianti: chatAddUser, chatEditTitle, pinMessage, phoneCall, ...).
- `tg.MessageEmpty` — placeholder server-side (saltato in convert).

Le opzioni per **discriminare** un service message nel dominio interno:

- **Opzione B1**: campo `IsService bool` + `ServiceText string` nel
  domain.Message unificato — già fissato in domain-types.md.
- **Opzione B2**: tipo separato `domain.SystemMessage` e
  `domain.UserMessage` con interface `MessageEntry` o sum-type via
  `MessageKind` esplicito.
- **Opzione B3**: campo `Action *MessageAction` con sotto-struct per ogni
  azione e dispatch al render.

Entrambe le decisioni devono coesistere con le invarianti già stabilite
in `reactions.tla` e con la struttura del viewport esistente (singolo
slice `[]Message`, append-only, accesso per index).

## Decisione

### A — Reactions: slice ordinata `[]Reaction`

**Adottiamo A1**: il campo `Reactions []Reaction` è una slice **già
ordinata** all'ingresso (in `convert/reactions.go`) per `Count desc`,
con `Emoji asc` come tie-break.

Razionale:

- **Order-preserving**: il rendering nel bubble è left-to-right; una
  slice ordinata si itera direttamente in `View()` senza intermediate
  step. `for _, r := range m.Reactions { write(r.Emoji, r.Count) }` —
  zero allocazioni, output deterministico.
- **Coerenza con Telegram desktop / mobile**: i client ufficiali
  ordinano per count desc. La nostra TUI replica lo stesso ordering
  visivo.
- **`ChosenByMe` per-emoji**, non per-message: l'utente può reagire
  con più emoji distinti alla stessa nota (Telegram Premium: fino a 3
  reazioni); il flag `ChosenByMe bool` su ogni `Reaction` cattura
  esattamente "ho usato questo emoji". Una map separata
  `chosenByMe map[string]bool` introduce un secondo source-of-truth da
  mantenere in sync.
- **Snapshot semantics**: `UpdateMessageReactions` invia il **set
  completo** ad ogni evento (non delta). Il client sostituisce
  l'intera slice. Una slice ordinata è il container naturale per uno
  snapshot ordinato.
- **Iterability + ergonomia Go**: gli idiomi Go preferiscono slice +
  range loop su map quando l'ordine conta. Una map richiederebbe
  estrazione delle chiavi, sort, walk — più codice e più allocazioni
  per ogni render.
- **Conversione gotd/td → domain in un solo passo**: il dispatch da
  `tg.MessageReactions` filtra le `ReactionEmoji` (skipping
  `ReactionCustomEmoji` per Step 25) e produce direttamente la slice
  ordinata. Vedi [`reactions-flow.md` §Convert layer](../phase-3-interactions/reactions-flow.md).
- **Cardinalità tipica bassa**: una nota su una chat di gruppo ha
  tipicamente 1-5 reazioni distinte, raramente >10. Il costo di
  iteration su slice è dominato dal costo del render della stringa
  emoji.

L'ordering è **invariante di dominio** (vedi
`reactions.tla` `REACTIONS_ORDERED`): la slice prodotta dal convert
layer è sempre sorted; nessun render-time sort.

### B — System message detection: flag `IsService` nel `domain.Message` unificato

**Adottiamo B1**: discriminante boolean `IsService` + payload testuale
pre-formattato `ServiceText` nello stesso struct `domain.Message`. Il
classifier `MessageKind` (vedi
[`reactions-and-system.md`](../phase-2-behavioral/reactions-and-system.md)
§MessageKind) è una funzione **derivata** sui campi:

```
kind(m) =
    "system"  if m.IsService
    "media"   else if m.Media != nil
    "text"    else
```

Razionale:

- **Coerenza con la codebase**: `domain.Message` è già unificato per
  text/media/reply/forward/reactions con campi opzionali. Aggiungere
  un secondo discriminante (oltre `Media != nil`) mantiene il pattern.
  Una sotto-tipo separata romperebbe la `[]Message` slice del viewport.
- **Slice append + render polimorfo**: il viewport è un singolo
  `[]Message` ordered. Con due tipi separati (`UserMessage`,
  `SystemMessage`) servirebbe una `[]MessageEntry` interface o un
  `[]any`; il primo introduce N tipi, il secondo perde type-safety.
- **Service text pre-formattato lato convert**: l'azione MTProto
  (`MessageActionChatAddUser{users:[123]}`) richiede il lookup
  dell'username dell'utente per produrre "Alice joined". Questo lookup
  avviene una volta in `convert/service.go` (con accesso alle entities
  cache della response). Memorizzare `ServiceText string` evita lookup
  ripetuti a render-time; il bubble renderer scrive la stringa
  pre-formattata.
- **Nessun action data nel viewport**: non abbiamo bisogno di
  `MessageActionClass` nel domain — il rendering è solo testuale (Step
  25 display-only). Un campo `Action interface{}` sarebbe payload
  morto. Se in futuro servirà (es. click su "joined" → apri profilo
  user), si potrà aggiungere senza rompere `IsService`/`ServiceText`.
- **Dispatch via discriminator coerente con DeliveryStatus / MediaType
  / ChatType**: tutti gli enum-discriminated dispatch della codebase
  usano `switch field`. Nessun nuovo paradigma introdotto.
- **TLA+ encoding diretto**: nello spec
  ([`reactions.tla`](../phase-4-concurrency/reactions.tla)) il flag
  `isService[id] BOOLEAN` cattura esattamente l'invariante
  `SYSTEM_IMMUTABLE`. Una sotto-tipo separata richiederebbe modellare
  due strutture dati distinte, complicando lo spec senza beneficio.
- **Telegram non muta IsService**: una volta creato, un service message
  non può "diventare" un user message né viceversa (verificato
  empiricamente e dalla schema TL). L'invariante di immutabilità è
  naturale.

## Alternative considerate

### A — Reactions storage shape

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **(scelta) A1: `[]Reaction` ordinata** | Iterability nativa Go; ordering preservato per render; snapshot replace = slice replace; un solo source-of-truth; cardinality tipica bassa | Sort eseguito a convert-time per ogni update (ma snapshot ha tipicamente 1-5 entry; cost trascurabile) |
| A2: `map[string]int` + `map[string]bool chosenByMe` | Lookup O(1) per emoji specifico (es. "ho già reagito con 👍?") | Ordering perso → doppio sort a render-time; doppio source-of-truth (count + chosen); update snapshot diventa wipe+rebuild di due map; più verbose e error-prone |
| A3: struct precomputato `*ReactionsBlock{Total, Top, Chosen}` | Render veloce (campi precalcolati) | Stato derivato extra da invalidare; complessità sproporzionata per Step 25 (display-only); semantica "Top N" arbitrary |
| A4: `[][2]string` array di tuple | Compatto in memoria | Perde struttura semantica; nominativi `m.Reactions[0][0]` illeggibili |

### B — System message detection

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **(scelta) B1: `IsService bool` + `ServiceText string` in `domain.Message`** | Coerenza con codebase; viewport unificato; pre-formatted text evita lookup ripetuti; TLA+ encoding banale | Campi opzionali in `Message` (zero-value su user msg); dipendenza implicita "if IsService → render con ServiceText" |
| B2: tipi separati `UserMessage` / `SystemMessage` con interface `MessageEntry` | Type safety; impossibile `IsService=true && Text != ""` invalido | Rompe `[]Message` slice del viewport; richiede `[]MessageEntry`; convert produce due tipi diversi; rendering switch su tipo concreto (analogo a B1 ma più boilerplate) |
| B3: campo `Action *MessageAction` strutturato | Estendibile per click-handlers futuri | Out of scope Step 25 (display-only); render comunque tornerebbe a switch su action variant; ServiceText resterebbe necessario per la stringa user-facing |
| B4: `Type MessageKind` enum in `Message` (no separate field, tutto unificato) | Singolo discriminante che copre service/media/text | `Media` è ortogonale a service/text (un media-msg non è system); doppio discriminante (Kind + Media-presence) crea stati invalidi (`Kind=text + Media != nil`?). B1 evita questa ambiguità: service è una proprietà a parte, media è opzionale, text è il default |

## Conseguenze

### Positive

- **Implementazione minimale**: il convert di reactions
  (`internal/telegram/convert/reactions.go`) è ~30 LOC: filter +
  sort.Slice + populate. Il convert di service
  (`internal/telegram/convert/service.go`) è un grande switch su
  ~30 action variants (~80 LOC stimate, sotto il limite 120).
- **Render polimorfico chiaro**: nel bubble renderer un'unica
  decision: `if m.IsService → renderCenteredService(m.ServiceText)`
  altrimenti rendering bubble standard. Niente type assertion.
- **Reactions row riusabile**: la funzione `appendReactionsRow(m)` è
  un helper puro (input: `[]Reaction`, output: stringa lipgloss
  styled). Testabile in isolamento.
- **TLA+ minimale**: lo spec `reactions.tla` modella sia store che
  rendering predicate con poche variabili. Le invarianti
  (`SYSTEM_IMMUTABLE`, `SYSTEM_NO_REACT`, `INDEPENDENT_FIELDS`,
  `REACTIONS_ORDERED`) sono check-abili da TLC in <5s.
- **Cross-step compatibility**: lo schema permette di estendere a
  reactions interactive (step futuro) senza rompere le invarianti
  attuali — basterà aggiungere un cmd `addReactionCmd` che invia
  l'API call e `UpdateMessageReactions` arriverà come al solito.
- **Independent updates**: text-edit e reactions-update sono
  ortogonali (`INDEPENDENT_FIELDS`). L'ordine di arrivo non altera
  lo stato finale: race-free by construction.

### Negative

- **Slice resort a ogni update**: `EmitReactionsUpdate` riceve uno
  snapshot da Telegram non garantitamente ordinato; il convert deve
  sortare. Con N <= 10 il cost è O(N log N) trascurabile, ma è una
  computazione per-update extra rispetto a una map naive.
- **`ServiceText` può andare stale per renaming**: se un utente cambia
  il proprio nome dopo che un service message è stato emesso,
  `ServiceText = "Alice joined"` resterà letterale anche se ora si
  chiama "Bob". È coerente col comportamento dei client ufficiali
  (Telegram non re-emit-a service messages al rename), accettabile.
- **Dispatch giant switch in `formatAction`**: ~30 case nel switch.
  Manutenzione: nuovi `MessageActionClass` aggiunti da Telegram
  finiscono nel fallback "Service message" (totalità garantita ma
  reso meno informativo). Mitigato da: code review periodica delle
  varianti gotd/td aggiunte a major version bump.
- **Custom emoji reactions ignored**: lo Step 25 skippa
  `ReactionCustomEmoji`. Per chat che ne fanno uso intensivo (gruppi
  premium), il count visualizzato sarà inferiore al totale Telegram.
  Mitigazione: future step può aggiungere un fallback `?N` per le
  custom skipped (es. `👍 3  ❤️ 2  +5`).

### Rischi

- **Reordering non deterministico tra emoji con stesso count**: il
  tie-break per `Emoji asc` (string comparison) garantisce
  determinismo. Senza tie-break, due render successivi potrebbero
  mostrare `👍 3  ❤️ 3` e poi `❤️ 3  👍 3` con flicker visivo.
  Il tie-break elimina questo rischio.
- **`IsService` flag flip per bug**: se per errore un convert path
  riassegna `IsService` su un message già esistente, l'invariante
  `SYSTEM_IMMUTABLE` cade. Mitigazione: il convert è solo
  `tg.MessageClass → domain.Message` (una direzione), e il viewport
  non muta `IsService` post-creation. TLA+ verifica.
- **Reactions su system msg edge case**: server normalmente non li
  emette, ma siamo difensivi (`SYSTEM_NO_REACT` rendering invariant).
  Il rischio è zero a livello di correctness; rischio cosmetico nullo
  perché il render skippa.
- **Render width overflow**: una nota con 50+ reazioni distinte
  (estremo, ma possibile per messaggi molto popolari) genera una
  reactions row lunga. La `media-rendering.md` §Render UI cita "wrap
  alla riga successiva tra entry, mai dentro un entry". Va testato a
  impl time con casi sintetici.

## Scope

Questa ADR si applica a:

- **Step 25 — Reactions + system messages** (prima applicazione).
- Step futuri che estenderanno reactions / service rendering:
  - Step "interactive reactions" (aggiungere/togliere reactions via
    keybinding) — riuserà `[]Reaction` shape, aggiungerà cmd outbound.
  - Step "custom emoji" — estenderà il convert per gestire
    `ReactionCustomEmoji` (probabilmente con fallback testuale o
    download del documento emoji animato — fuori scope per ora).
  - Step "service message interactivity" (es. click su "joined" →
    apri profilo) — estenderà `domain.Message` con `Action
    *MessageAction` ma manterrà `IsService`/`ServiceText` per il
    render.

Non si applica a:

- Display dei reactions reattivi tra utenti (long-press → lista
  utenti che hanno reagito) — out of scope, richiede una nuova ADR.
- Throttling client-side delle update di reactions (il rate dei
  `UpdateMessageReactions` da server è già moderato; non vediamo
  necessità di debounce in Step 25).

## Cross-links

- [`phase-2-behavioral/reactions-and-system.md`](../phase-2-behavioral/reactions-and-system.md) §MessageKind §Reactions storage
- [`phase-3-interactions/reactions-flow.md`](../phase-3-interactions/reactions-flow.md) §Convert layer §Update race
- [`phase-4-concurrency/reactions.tla`](../phase-4-concurrency/reactions.tla) — invarianti `SNAPSHOT_NONNEG`, `SYSTEM_IMMUTABLE`, `SYSTEM_NO_REACT`, `INDEPENDENT_FIELDS`, `REACTIONS_ORDERED`, `DELETE_PROPAGATES`
- [`phase-5-data/domain-types.md`](../phase-5-data/domain-types.md) §Reaction §Message
- [`phase-5-data/entity-mapping.md`](../phase-5-data/entity-mapping.md) §Reactions Mapping §System Message Mapping
- [`phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md) §Telegram Events
- Pipeline Step 25
