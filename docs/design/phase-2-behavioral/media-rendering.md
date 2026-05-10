# Media Rendering — Taxonomy & Dispatch (Step 24)

Modello comportamentale del **rendering inline dei messaggi media** introdotto
nello Step 24. Il messaggio porta un payload `MessageMedia` opzionale: in
`Step 24` ci limitiamo al **render testuale single-line** della summary
(`Icon() + " " + Summary()`) inline nel bubble esistente.

**Scope Step 24**:
- Photo, Video, Voice, Document, Sticker.
- Display-only: nessun download, nessun viewer, nessun upload.
- Branching dispatch da `tg.MessageMediaClass` → `MediaKind` interno.
- Voice: rendering del waveform a caratteri braille verticali (`▁▂▃▄▅▆▇█`).

**Fuori scope Step 24** (rimandati a step futuri o esplicitamente non
trattati):
- Download su disco di photo/video/document.
- Media viewer (apertura immagini in pannello dedicato, anteprima video).
- Upload di media in invio.
- `tg.MessageMediaWebPage` (link previews) — vedi
  [ADR-011](../phase-6-decisions/ADR-011-media-rendering-taxonomy.md) §WebPage.
- `tg.MessageMediaGeo`, `tg.MessageMediaVenue`, `tg.MessageMediaContact`,
  `tg.MessageMediaPoll` — già presenti come `MediaType` nel domain
  ([phase-5-data/domain-types.md](../phase-5-data/domain-types.md)) ma non
  oggetto di rendering polimorfico in questo step (fallback a `Document`
  generico se ricevuti).
- Reazioni e system messages → Step 25.

## Taxonomy — Class Diagram

`MessageMedia` resta uno **struct unico** con campo discriminante `Type
MediaType` (così come già definito in
[`phase-5-data/domain-types.md`](../phase-5-data/domain-types.md)). I metodi
polimorfici `Icon()` e `Summary()` sono implementati con `switch m.Type` —
vedi [ADR-011](../phase-6-decisions/ADR-011-media-rendering-taxonomy.md) per
la giustificazione (Go non ha sum types; tagged-struct è il pattern
canonico nella codebase, già usato per `DeliveryStatus.Symbol()`).

```mermaid
classDiagram
    class MessageMedia {
        +MediaType Type
        +string FileName
        +int64 Size
        +string MimeType
        +time.Duration Duration
        +[]byte Waveform
        +string StickerEmoji
        +string StickerPack
        +Icon() string
        +Summary() string
    }

    class MediaType {
        <<enumeration>>
        MediaPhoto
        MediaVideo
        MediaAudio
        MediaVoice
        MediaDocument
        MediaSticker
        MediaLocation
        MediaContact
        MediaPoll
    }

    MessageMedia --> MediaType : Type

    class tg_MessageMediaPhoto {
        +tg.Photo Photo
    }
    class tg_MessageMediaDocument {
        +tg.Document Document
    }
    class tg_MessageMediaWebPage {
        +tg.WebPage WebPage
    }

    tg_MessageMediaPhoto ..> MessageMedia : "→ MediaPhoto"
    tg_MessageMediaDocument ..> MessageMedia : "→ MediaVideo|Audio|Voice|Sticker|Document<br/>(branch on mime/attributes)"
    tg_MessageMediaWebPage ..> MessageMedia : "OUT of scope (Step 24)<br/>fallback: render as text only"
```

**Nota**: l'**alternativa** "interface `MessageMedia` + un tipo concreto per
kind" (`PhotoMedia`, `VoiceMedia`, ...) è stata valutata e scartata. Vedi
[ADR-011 §Alternative considerate](../phase-6-decisions/ADR-011-media-rendering-taxonomy.md).

## Dispatch — Decision Tree (gotd/td → MediaKind)

Il branching avviene in `internal/telegram/convert/media.go` (futura
implementazione). Lo statechart sotto modella la **macchina di decisione**
applicata a un singolo `tg.MessageMediaClass` in arrivo:

```mermaid
stateDiagram-v2
    [*] --> Inspect

    Inspect --> Photo : tg.MessageMediaPhoto
    Inspect --> InspectDocument : tg.MessageMediaDocument
    Inspect --> WebPageOOS : tg.MessageMediaWebPage
    Inspect --> GenericFallback : tg.MessageMediaGeo<br/>tg.MessageMediaVenue<br/>tg.MessageMediaContact<br/>tg.MessageMediaPoll<br/>(other)
    Inspect --> NoMedia : nil / tg.MessageMediaEmpty

    state InspectDocument {
        [*] --> CheckAttributes
        CheckAttributes --> Voice : has DocumentAttributeAudio<br/>{Voice: true}
        CheckAttributes --> Audio : has DocumentAttributeAudio<br/>{Voice: false}
        CheckAttributes --> Sticker : has DocumentAttributeSticker
        CheckAttributes --> Video : has DocumentAttributeVideo<br/>(no Voice attribute)
        CheckAttributes --> CheckMime : (no decisive attribute)

        state CheckMime {
            [*] --> MimeBranch
            MimeBranch --> VideoMime : mime starts with "video/"
            MimeBranch --> AudioMime : mime starts with "audio/"
            MimeBranch --> DocumentGeneric : (default)
        }
    }

    Photo --> [*]
    Voice --> [*]
    Audio --> [*]
    Sticker --> [*]
    Video --> [*]
    VideoMime --> [*]
    AudioMime --> [*]
    DocumentGeneric --> [*]
    GenericFallback --> [*]
    WebPageOOS --> [*] : (no domain.MessageMedia produced;<br/>message.Text rendered as-is)
    NoMedia --> [*] : (Media field stays nil)
```

### Priorità degli attributi document

L'ordine di check è **fisso** e privilegia gli attributi specifici sul mime
generico (un mp3 marcato come voice resta voice, anche se mime = `audio/ogg`):

1. `DocumentAttributeAudio.Voice == true` → **Voice**.
2. `DocumentAttributeAudio.Voice == false` → **Audio**.
3. `DocumentAttributeSticker` → **Sticker** (estrae `Alt` come emoji,
   `StickerSet` per pack name).
4. `DocumentAttributeVideo` (non gif sticker) → **Video**.
5. Altrimenti, branching su `MimeType`:
   - `"video/*"` → **Video**.
   - `"audio/*"` → **Audio**.
   - default → **Document** generico (`📎 file.pdf (2.4 MB)`).

Questa cascata garantisce che il dispatch sia **totale** (ogni
`tg.MessageMediaDocument` produce esattamente una `MediaType`) e
**deterministico** (stessa input → stessa output, no random ordering su
`Attributes []DocumentAttributeClass`).

## Render — Forma testuale per kind

Tutti i media sono renderizzati come **single-line inline** dentro il
bubble, sostituendo o accompagnando il testo del messaggio. La forma è:

| Kind | Template | Esempio |
|------|----------|---------|
| Photo | `📷 {filename} ({size})` | `📷 photo.jpg (1.2 MB)` |
| Video | `🎬 {filename} ({size}) · {duration}` | `🎬 movie.mp4 (45 MB) · 2:15` |
| Audio | `🎵 {filename} ({size}) · {duration}` | `🎵 song.mp3 (3.5 MB) · 3:42` |
| Voice | `🎤 {waveform-braille} {duration}` | `🎤 ▁▂▃▅▇█▆▄▃▂ 0:42` |
| Document | `📎 {filename} ({size})` | `📎 file.pdf (2.4 MB)` |
| Sticker | `{emoji} {pack-name}` | `🎉 PartyPack` |
| Other (Location/Contact/Poll/...) | `📄 {kind-label}` | `📄 Location`, `📄 Contact`, `📄 Poll` |

**Note**:
- `{size}` è formattato human-readable: `B / KB / MB / GB`, una decimale se
  `>= 10` (es. `1.2 MB`, `345 KB`).
- `{duration}` è `m:ss` se `< 1h`, `h:mm:ss` altrimenti.
- `{filename}` mancante → derivato dal mime (`photo.jpg`, `video.mp4`,
  `audio.mp3`, `voice.ogg`, `document.bin`).
- `{waveform-braille}` è una stringa di **N glifi fissi** (vedi
  [`media-flow.md` §Braille mapping](../phase-3-interactions/media-flow.md)).
- Sticker senza `Alt` → fallback a `🖼️`. Sticker senza pack → solo emoji.

## Integrazione nel render del bubble (esistente)

Il rendering del bubble (introdotto in Step 12, esteso in Step 18 per
reply, Step 13 per timestamp) viene **esteso** così:

```
Esistente:
  ┌────────────────────────────────┐
  │ {ReplyBar (se presente)}        │
  │ {Text}                          │
  │                       {Time}  ✓ │
  └────────────────────────────────┘

Step 24 (con media):
  ┌────────────────────────────────┐
  │ {ReplyBar (se presente)}        │
  │ {Media.Icon()} {Media.Summary()}│   ← nuova riga, dim color se vuoi
  │ {Text}     (se non vuoto)       │
  │                       {Time}  ✓ │
  └────────────────────────────────┘
```

- Se `Message.Text != ""` e `Message.Media != nil` → entrambe le righe
  rendered (Telegram chiama "caption" il testo accompagnato a media).
- Se `Message.Text == ""` e `Message.Media != nil` → solo riga media.
- Se `Message.Media == nil` → comportamento Step 23 (nessun cambio).

**Invariante**: l'aggiunta del media **non modifica** la logica di
allineamento (incoming/outgoing, Step 12), grouping (Step 13), reply bar
(Step 18), receipts (Step 16). È un'aggiunta puramente additiva sopra il
text content.

## Stati implicati e transizioni

In Step 24 non si introduce una nuova macchina a stati nel senso UI
(non ci sono interazioni utente attivabili sul media: nessun download key,
nessun viewer). L'unica "macchina" è quella di **dispatch tipo** mostrata
sopra, che è un puro classifier `tg.MessageMediaClass → MediaType`
applicato all'arrivo del messaggio (in `convert/media.go`).

## Eventi / Messaggi (tipizzati `tea.Msg`)

Lo Step 24 **non aggiunge** nuovi `tea.Msg`. I media arrivano già contenuti
in `Message.Media` quando viene emesso `NewMessageMsg{Message}` (Step 17)
o `MessagesLoadedMsg{...}` (Step 11). Il convert layer popola il campo
`Media` durante il mapping `tg.Message → domain.Message`.

## Invarianti comportamentali

1. **Totalità del dispatch**: ogni `tg.MessageMediaClass` non-nil produce
   un `*MessageMedia` valido (eventualmente con `Type = MediaDocument`
   come fallback). Nessun crash, nessun panic su union variant non
   riconosciuta.
2. **Determinismo**: stessa input → stessa `MediaType` (no random sui
   `Attributes []`).
3. **Idempotenza render**: `m.Icon()` e `m.Summary()` sono pure (no I/O,
   no allocazioni nascoste oltre la stringa risultante). Possono essere
   chiamati N volte per render senza side-effect.
4. **Render fallback safe**: se mancano dati (`FileName == ""`, `Size ==
   0`, `Duration == 0`, `Waveform == nil`) il render produce comunque
   una stringa stabile (es. `🎤 (no waveform) 0:00` o
   `📎 document.bin`). Vedi
   [`../phase-3-interactions/media-flow.md` §Fallback rules](../phase-3-interactions/media-flow.md).
5. **WebPage non modella un media**: per `tg.MessageMediaWebPage` non si
   produce `*MessageMedia`; il `Message.Text` (che contiene già l'URL)
   viene renderizzato as-is. La preview web è demandata a step futuro
   ([ADR-011](../phase-6-decisions/ADR-011-media-rendering-taxonomy.md)
   §WebPage).

## Cross-links

- Pipeline step: [`development-pipeline.md` §Step 24](../development-pipeline.md)
- Sequence diagrams + braille mapping: [`../phase-3-interactions/media-flow.md`](../phase-3-interactions/media-flow.md)
- TLA+ braille model: [`../phase-4-concurrency/media_waveform.tla`](../phase-4-concurrency/media_waveform.tla)
- Decisione taxonomy + braille: [ADR-011](../phase-6-decisions/ADR-011-media-rendering-taxonomy.md)
- Domain types: [`../phase-5-data/domain-types.md`](../phase-5-data/domain-types.md) §MessageMedia
- Entity mapping: [`../phase-5-data/entity-mapping.md`](../phase-5-data/entity-mapping.md) §Media Mapping
- Message taxonomy (no nuovi `tea.Msg`): [`../phase-1-context/message-taxonomy.md`](../phase-1-context/message-taxonomy.md)
