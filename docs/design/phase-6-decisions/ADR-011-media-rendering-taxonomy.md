# ADR-011: Media rendering taxonomy + braille waveform charset

**Stato**: accettato
**Data**: 2026-04-25

## Contesto

Lo Step 24 introduce il rendering inline dei messaggi media (Photo, Video,
Voice, Document, Sticker) nei bubble della conversazione. È un Display-only
step: nessun download, nessun viewer, nessun upload. Le esigenze:

1. **Tassonomia dei media**: come modellare in Go il polimorfismo
   `Photo / Video / Voice / Document / Sticker / ...` con due metodi
   visuali `Icon() string` e `Summary() string`?
2. **Dispatch da gotd/td**: `tg.MessageMediaClass` è un'unione MTProto con
   ~10 varianti; di queste, `tg.MessageMediaDocument` è a sua volta
   poliforfa per **mime + attributes** (un voice è un document con
   `AttributeAudio.Voice = true`; uno sticker è un document con
   `AttributeSticker`; un video è un document con `AttributeVideo` o
   `mime_type=video/*`). Serve un classifier deterministico, totale,
   non-ambiguo.
3. **Voice waveform**: Telegram fornisce `[]byte` con campioni a 5 bit
   packed (range 0..31). Va renderizzato come stringa di glifi monospace
   nel bubble. Quale charset? Quanti glifi? Come gestire boundary cases
   (waveform vuoto, lunghezza non standard)?
4. **`tg.MessageMediaWebPage` (link previews)**: include scope o
   rimanda?
5. **Caching dei metadata**: visto che non scarichiamo nulla, ha senso
   cachare i `MessageMedia` decodificati?

Il design del **domain.MessageMedia** è già fissato in
[`phase-5-data/domain-types.md`](../phase-5-data/domain-types.md) come
**struct unico** con campo discriminante `Type MediaType`. L'ADR-011
ratifica e motiva questa scelta (mai documentata formalmente prima) e
aggiunge le decisioni sul charset waveform e sulla gestione di WebPage.

## Decisione

### 1. Tassonomia: tagged-struct unico con `Type MediaType`

`MessageMedia` è uno **struct singolo** con tutti i campi possibili (alcuni
opzionali per tipo) e un discriminante `Type`. I metodi `Icon()` e
`Summary()` sono **dispatch via `switch m.Type`**, identici al pattern già
usato per `DeliveryStatus.Symbol()` (Step 16).

Razionale:

- **Go non ha sum types**: l'alternativa "interface + struct concreti per
  kind" introduce N tipi (`PhotoMedia`, `VoiceMedia`, ..., `StickerMedia`)
  e una factory che boxes. Maggior superficie, no benefici concreti per
  un display-only step.
- **Coerenza con la codebase esistente**: `DeliveryStatus`, `ChatType`,
  `OnlineStatus` usano tutti il pattern enum + switch. Introdurre
  un'interfaccia ad-hoc per `Media` rompe la coerenza.
- **Embedding diretto in `Message`**: il campo `*MessageMedia` (puntatore
  nullable) è semanticamente "0 o 1 media per messaggio". Con
  un'interfaccia servirebbe un `MessageMedia` interface field con stesso
  effetto ma più boilerplate (nil-check dell'interface, impossibilità di
  zero-value default).
- **Serializability futura**: se in futuro vorremo persistere i media
  (cache su disco), un singolo struct serializza in JSON banalmente; con
  interface servirebbe custom MarshalJSON.

### 2. Dispatch order: attributi document > mime

Per `tg.MessageMediaDocument`, l'ordine di check è **fisso**:

1. `DocumentAttributeAudio.Voice == true` → `MediaVoice`
2. `DocumentAttributeAudio.Voice == false` → `MediaAudio`
3. `DocumentAttributeSticker` → `MediaSticker`
4. `DocumentAttributeVideo` → `MediaVideo`
5. `mime_type: "video/*"` → `MediaVideo`
6. `mime_type: "audio/*"` → `MediaAudio`
7. (default) → `MediaDocument`

Razionale: gli attributi sono **dati specifici** marcati esplicitamente
dal mittente; il mime è metadata generico spesso impreciso (es. voice ogg
ha `mime=audio/ogg` ma è semanticamente Voice, non Audio). Privilegiare
attributi su mime evita classificazioni errate.

L'ordine è anche **deterministico** rispetto a `Attributes []Class` che
in MTProto può avere ordering arbitrario: noi iteriamo cercando la prima
match in priority order.

### 3. Charset waveform: 8-level block-element `▁▂▃▄▅▆▇█`, N=10 glifi fissi

Charset scelto: i caratteri Unicode block-element (U+2581..U+2588). 8
livelli di intensità verticale. **N = 10 glifi fissi** indipendente dalla
durata del voice.

Razionale charset:

- **Monospace puro**: tutti i terminali emettono questi glifi a
  larghezza fissa identica al `space` ASCII. Niente sorprese di
  layout.
- **Fontmap universale**: presenti in font terminale di default
  (Cascadia, Menlo, JetBrains Mono, Fira Code, DejaVu). Mai box vuoti.
- **Già pattern noto in TUI**: `gotop`, `btop`, `gh`, `lazygit` usano
  block-element per spark / chart. Riconoscibilità immediata.
- **Sufficiente granularità**: 8 livelli = 3 bit di output da 5 bit di
  input, perdita 2 bit. Sufficiente per percepire dinamica del voice
  (silenzio → picchi).

Razionale N=10 fisso:

- **Larghezza prevedibile**: il bubble ha un layout fisso; un waveform
  variable-width complicherebbe il rendering (wrapping, overflow).
- **10 glifi ≈ 30 caratteri di larghezza con padding**: comodo nei
  bubble di larghezza tipica (60-80 col).
- **Indipendenza dalla durata**: una nota da 30s e una da 5min hanno
  entrambe 10 glifi. È un'informazione di **inviluppo**, non di scala
  temporale assoluta. La durata è già mostrata accanto (`0:42`).
- **Resampling banale**: la funzione `brailleWaveform(data, N=10)` fa
  mean-bucket, totale e deterministica
  ([`media_waveform.tla`](../phase-4-concurrency/media_waveform.tla)).

### 4. WebPage media: out-of-scope Step 24

`tg.MessageMediaWebPage` (link previews: titolo, descrizione, thumbnail)
**non produce un `*MessageMedia`** in Step 24. Razionale:

- È strutturalmente diverso dagli altri media: porta **due payload
  testuali** (title, description) + URL + opzionale thumbnail. Non
  rientra nel template `Icon() + " " + Summary()` single-line.
- Il vero rendering naturale è un **box riassuntivo multi-riga sotto al
  testo** (come Telegram Desktop), che richiede una sotto-vista
  dedicata. Step 33 introduce "links navigabili" e potrebbe estendersi
  per coprire web previews.
- Rimando esplicito: per messaggi con `WebPage`, il `Message.Text`
  contiene già l'URL → è renderizzato come testo normale, l'utente vede
  il link. Nessuna perdita di informazione critica.

### 5. No caching dei metadata media

In Step 24 non si introduce caching dei `MessageMedia` decodificati.
Razionale:

- I media sono già contenuti in `domain.Message`, che vive nel viewport
  del Conversation model (lifecycle = chat aperta). Non c'è ricomputo
  inter-render: `Icon()` e `Summary()` sono pure e veloci (string
  concat + format).
- La decodifica braille avviene **una volta a render** (cost: O(N) con
  N=10 + decode di un waveform di tipica lunghezza ~64 byte). Misurato
  sub-microsecondo.
- Cachare richiederebbe invalidation policy → tech debt non
  giustificato per Step 24 (che è display-only).
- Caching potrebbe diventare rilevante con download (Step futuro): in
  quel caso si farà un'ADR dedicata che copre file caching su disco.

## Alternative considerate

### Tassonomia

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **(scelta) Struct unico + `Type MediaType` + switch** | Coerenza con `DeliveryStatus.Symbol()`; nullable via `*MessageMedia`; serializable; minimo boilerplate | Campi opzionali per kind (es. `StickerEmoji` ha senso solo per Sticker); zero-value lecito ma semanticamente "rumoroso" |
| Interface `MessageMedia` + struct concreti (`PhotoMedia`, `VoiceMedia`, ..., `StickerMedia`) | Type-safety per kind: ogni struct ha solo i suoi campi; render polimorfo via interface | Boilerplate (5+ tipi); rompe pattern codebase; nil-check dell'interface insidioso (typed-nil); serialization custom |
| Struct unico + helper functions free (`PhotoSummary(m)`, `VoiceSummary(m)`) | No metodi sul tipo (POD pure) | Dispersa la logica; dispatch fatto al call-site, non incapsulato; perde discoverability `m.Summary()` |
| Generics (`MessageMedia[T MediaPayload]`) | Type-safe parametrico | Eccesso di complessità per un dominio chiuso fisso; non ha precedenti nella codebase; richiede Go 1.18+ con tutti i vincoli |

### Charset waveform

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **(scelta) Block-element `▁▂▃▄▅▆▇█` (8 lvl)** | Monospace garantito; fontmap universale; pattern noto TUI | Solo 3 bit di intensità (perdita 2 bit dai 5 di input) |
| Braille pattern Unicode (U+2800..U+28FF, 256 glifi) | Densità 2x (2 sample per colonna); 4 livelli per colonna | Spesso proporzionale (rompe monospace); fontmap incompleta su molti temi; visivamente meno "barra" e più "rumore" |
| Solo ASCII `_.-=oO0#` | Compatibile ovunque | Brutto, non riconoscibile come waveform; collisioni con caratteri normali |
| Sparkline Unicode `▁▂▃▄▅▆▇` (7 lvl, manca `█`) | Praticamente identico a block-element | 7 lvl invece di 8: minore granularità senza vantaggi |
| Render a colore + spazio (heatmap) | Estetica gradevole | Non leggibile in temi monocromatici; richiede ANSI 24-bit; copia/paste nullo |

### Glyph count (N)

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **(scelta) N=10 fisso** | Layout prevedibile; resampling semplice; dimensione confortevole nel bubble | Voice molto brevi (1-2 sec) appaiono "stiracchiati"; voice molto lunghi (5min) collassano dinamica |
| N proporzionale alla durata (`N = duration/sec`) | Più informazione visiva per voice lunghi | Layout variabile; bubble overflow; difficile testare; spreca spazio per voice corti |
| N=ceil(width_disponibile/3) (responsive) | Si adatta al layout | Complica `Summary()` (deve sapere la larghezza disponibile → API leakage); ricalcolo a ogni resize |
| N=8 fisso | Più compatto | Granularità inviluppo ridotta |

### WebPage

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **(scelta) Out of scope, rimando a step futuro** | Non distrae lo Step 24; URL già visibile nel testo | Esperienza utente "spartana" per messaggi link-only |
| Render come `MediaDocument` con titolo come filename | Riusa il template generico | Non rappresenta correttamente la natura del WebPage; titolo + URL non sono filename + size |
| Render in-line breve (`🌐 example.com`) | Minimo sforzo, segnala il link | Aggiunge un kind virtuale che non è in `MediaType` enum |

### Caching

| Alternativa | Pro | Contro |
|-------------|-----|--------|
| **(scelta) No cache** | Zero complessità; render veloce comunque | Ricomputo a ogni render |
| LRU cache `messageID → string` (summary precomputata) | Evita ricomputo | Invalidation in caso di edit message; memoria extra; tech debt non giustificato |
| Memoization su `*MessageMedia` (campo `cachedSummary string`) | Lazy, una volta sola | Mutazione del domain type a render-time → viola immutabilità |

## Conseguenze

### Positive

- **Implementazione minimale**: `internal/model/media.go` con `Icon()` e
  `Summary()` rispetta il limite 120 LOC. La logica braille va in
  `internal/ui/render/waveform.go` (~50 LOC stimate, coperta da unit
  test che mirror le invarianti TLA+).
- **Estendibilità**: aggiungere un nuovo `MediaType` (es. `MediaGif`,
  `MediaRoundVideo`) richiede un case nello switch + un caso nel
  dispatcher gotd/td. Pattern noto.
- **Coerenza render**: tutti i media condividono il template
  `Icon + Summary` single-line, layout uniforme nei bubble.
- **Testabilità**: la funzione braille è pura → unit test deterministici;
  invarianti TLA+ check-abili con TLC.
- **Cross-step**: il pattern `tagged-struct + switch` è già stato
  validato (Step 16); nessun nuovo paradigma introdotto.

### Negative

- **Campi opzionali nello struct**: `MessageMedia` ha 13 campi, di cui
  solo 1-3 popolati per kind. Il database mentale "quali campi valgono
  per quale Type" va memorizzato. Mitigato da:
  - Doc commenti accanto a ogni campo che indicano il `Type` di
    competenza.
  - Helper costruttori `NewPhotoMedia(...)`, `NewVoiceMedia(...)` se
    serviranno in test (out-of-scope per ora).
- **Dispatch giant switch**: `Icon()` e `Summary()` saranno due switch
  paralleli. Per 9 tipi è gestibile; oltre 15 tipi diventerebbe
  opportuno tirar fuori una function-table.
- **Granularità waveform 8-livelli**: chi ha voice molto "piatti" vede
  poca dinamica. Cosmetic only, non altera correctness.
- **N=10 fisso non scala con bubble width**: in compact mode (Step 30)
  il bubble è più stretto; 10 glifi block-element sono ~10 colonne, OK
  fino a width≈40. Se serviranno bubble più piccoli, si farà ADR
  estensione.

### Rischi

- **Voice senza waveform**: alcuni client (es. Telegram Desktop su voce
  inoltrate) emettono `Waveform` vuoto/`nil`. Mitigato dalla
  `EMPTY_FALLBACK`: render `🎤 ────────── 0:42`. Verificato in
  TLA+ spec.
- **Sticker pack senza nome**: `tg.DocumentAttributeSticker.StickerSet`
  può essere `inputStickerSetEmpty`. Render fallback: solo emoji, no
  pack name. Comportamento accettabile (es. sticker custom).
- **Mime ambiguous**: un .mp3 con attributo voice DEVE finire in
  Voice (priorità attributo > mime). Test case esplicito da aggiungere
  a impl time.
- **Charset block-element non monospace su font esotiche**: in pratica
  tutti i terminali moderni li gestiscono come monospace. Rischio
  residuo: utenti con font custom rotti. Trascurabile.
- **Fallback `─` (em-dash)**: U+2500 box-drawing. Anch'esso monospace
  universale. Nessun rischio aggiuntivo rispetto al charset principale.

## Scope

Questa ADR si applica a:

- **Step 24 — Media messages** (prima applicazione).
- Step futuri che estenderanno il rendering media:
  - Step 25 (reactions) NON tocca questa ADR (reactions sono separati).
  - Eventuale step "media download + viewer" — se accadrà, sarà ADR
    dedicata che riusa la tassonomia stabilita qui.
  - Eventuale step "WebPage previews" — sostituirà la decisione §4.

## Cross-links

- [`phase-2-behavioral/media-rendering.md`](../phase-2-behavioral/media-rendering.md) §Taxonomy + Decision Tree
- [`phase-3-interactions/media-flow.md`](../phase-3-interactions/media-flow.md) §Braille Waveform Mapping
- [`phase-4-concurrency/media_waveform.tla`](../phase-4-concurrency/media_waveform.tla) §Invariants TOTAL / LENGTH / MONOTONIC / SILENCE / SATURATION / EMPTY_FALLBACK
- [`phase-5-data/domain-types.md`](../phase-5-data/domain-types.md) §MessageMedia
- [`phase-5-data/entity-mapping.md`](../phase-5-data/entity-mapping.md) §Media Mapping (cascata document → MediaType)
- Pipeline Step 24
