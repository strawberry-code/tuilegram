# Entity Mapping

Mapping tra tipi gotd/td (`tg.*`) e tipi di dominio interni.

## Mapping Strategy

I tipi gotd/td sono generati automaticamente e riflettono 1:1 lo schema MTProto. Sono verbosi, con molti campi opzionali e union types. I tipi di dominio sono semplificati e ottimizzati per il rendering TUI.

**Regola**: il mapping avviene in un package dedicato (`internal/telegram/convert`) che isola il resto del codice dai tipi gotd/td.

## User Mapping

```
tg.User → domain.User
```

| tg.User field | domain.User field | Note |
|---------------|-------------------|------|
| `ID` | `ID` | Diretto |
| `FirstName` | `FirstName` | Diretto |
| `LastName` | `LastName` | Può essere vuoto |
| `Username` | `Username` | Senza @, può essere vuoto |
| `Phone` | `Phone` | Vuoto se privacy settings lo impediscono |
| `About` | `Bio` | Richiede chiamata separata (`users.getFullUser`) |
| `Bot` | `IsBot` | Flag booleano |
| `Status` | `Status` | Vedi mapping status sotto |
| `AccessHash` | `AccessHash` | Necessario per API calls |

### Online Status Mapping

```
tg.UserStatus (union type) → domain.OnlineStatus
```

| tg variant | OnlineStatus |
|------------|-------------|
| `tg.UserStatusOnline` | `{IsOnline: true}` |
| `tg.UserStatusOffline{WasOnline}` | `{IsOnline: false, LastSeen: WasOnline}` |
| `tg.UserStatusRecently` | `{IsOnline: false, LastSeen: ~recent}` |
| `tg.UserStatusLastWeek` | `{IsOnline: false, LastSeen: ~week ago}` |
| `tg.UserStatusLastMonth` | `{IsOnline: false, LastSeen: ~month ago}` |
| `nil` / `tg.UserStatusEmpty` | `{IsOnline: false}` |

## Chat Mapping

Le chat in Telegram hanno 3 tipi distinti nella schema TL:

```
tg.Chat (basic group)     ─┐
tg.Channel (supergroup)    ├→ domain.Chat
tg.User (private chat)    ─┘
```

### Da Dialog

```
tg.Dialog + tg.Entities → domain.Chat
```

| Sorgente | domain.Chat field | Note |
|----------|-------------------|------|
| `dialog.Peer` | `ID` | ChatID costruito dal PeerType |
| entities lookup | `Type` | Derivato: User→Private/Bot, Chat→Group, Channel→Channel/Group |
| entity name | `Title` | User.DisplayName() o Chat.Title o Channel.Title |
| `dialog.UnreadCount` | `UnreadCount` | Diretto |
| `dialog.Pinned` | `IsPinned` | Diretto |
| `dialog.NotifySettings` | `IsMuted` | `MuteUntil > now` |
| `dialog.FolderID` | `FolderID` | 0 se nessuna cartella |
| top message | `LastMessage` | Dal campo `TopMessage` + lookup nel messaggio |
| entity | `User` | Solo per chat private, dal lookup in Entities |
| entity | `MemberCount` | `Channel.ParticipantsCount` o `Chat.ParticipantsCount` |
| entity | `AccessHash` | Necessario per tutte le API calls |

### ChatType derivation

```go
func deriveChatType(peer tg.PeerClass, entities tg.Entities) ChatType {
    switch p := peer.(type) {
    case *tg.PeerUser:
        user := entities.Users[p.UserID]
        if user.Bot { return ChatBot }
        if user.Self { return ChatSavedMessages }
        return ChatPrivate
    case *tg.PeerChat:
        return ChatGroup
    case *tg.PeerChannel:
        ch := entities.Channels[p.ChannelID]
        if ch.Broadcast { return ChatChannel }
        return ChatGroup  // supergroup
    }
}
```

## Message Mapping

```
tg.MessageClass → domain.Message
```

Il dispatch di tipo concreto (vedi
[`reactions-and-system.md` §System Message Classification](../phase-2-behavioral/reactions-and-system.md)):

- `*tg.Message` → user message ordinario (`IsService = false`).
- `*tg.MessageService` → service message (`IsService = true`,
  `ServiceText` pre-formattato dall'action). Vedi §System Message
  Mapping più sotto.
- `*tg.MessageEmpty` → skipped (no domain message emesso).

### `tg.Message` → `domain.Message` (regular)

| tg.Message field | domain.Message field | Note |
|------------------|---------------------|------|
| `ID` | `ID` | Diretto |
| `PeerID` | `ChatID` | Costruito dal PeerType |
| `FromID` | `SenderID` | Può essere PeerUser, PeerChannel, PeerChat |
| — | `SenderName` | Lookup in entities cache |
| `Message` | `Text` | Testo del messaggio |
| `Media` | `Media` | Vedi §Media Mapping |
| `ReplyTo` | `ReplyTo` | Vedi §Reply Mapping |
| `FwdFrom` | `Forward` | Vedi §Forward Mapping |
| `Reactions` | `Reactions` | Vedi §Reactions Mapping (Step 25; ordinata count desc) |
| `Out` | `IsOutgoing` | Flag booleano |
| — | `IsService` | `false` per `*tg.Message` (Step 25) |
| — | `ServiceText` | `""` per `*tg.Message` (Step 25) |
| `Date` | `Date` | Unix timestamp → time.Time |
| `EditDate` | `EditDate` | 0 se non editato |

### Delivery Status Mapping

| Condizione | DeliveryStatus |
|-----------|----------------|
| Messaggio appena creato localmente | `StatusSending` |
| `Out == true` && ricevuto da server | `StatusSent` |
| `Out == true` && `UpdateReadHistoryOutbox.MaxID >= ID` | `StatusRead` |
| Errore durante l'invio | `StatusFailed` |

**Nota**: Telegram non distingue esplicitamente "delivered" da "sent" per le chat private. Usiamo `StatusDelivered` come stato intermedio opzionale.

## Media Mapping

```
tg.MessageMedia (union type) → domain.MessageMedia
```

**Step 24 scope**: il dispatch sotto è **implementato per Photo, Video,
Audio, Voice, Sticker, Document**. `Geo / Venue / Contact / Poll` esistono
nel domain ma in Step 24 ricevono fallback a `MediaDocument` con
label generico (vedi
[`../phase-2-behavioral/media-rendering.md`](../phase-2-behavioral/media-rendering.md)
§Render). `tg.MessageMediaWebPage` non produce `*MessageMedia` (vedi
[ADR-011](../phase-6-decisions/ADR-011-media-rendering-taxonomy.md)
§WebPage).

L'**ordine di check degli attributi** del document è fisso e documentato
nello statechart di dispatch
([`media-rendering.md` §Decision Tree](../phase-2-behavioral/media-rendering.md)):
attributi specifici (`AttributeAudio.Voice`, `AttributeSticker`,
`AttributeVideo`) vincono sul mime; mime fa da fallback per document senza
attributi decisivi.

| tg variant | MediaType (Step 24) | Campi estratti |
|------------|---------------------|----------------|
| `tg.MessageMediaPhoto` | `MediaPhoto` | FileName (generato `photo.jpg`), Size (dal PhotoSize più grande) |
| `tg.MessageMediaDocument{document: tg.Document}` | (vedi cascata) | (vedi cascata) |
| — document con `AttributeAudio{Voice: true}` (priorità 1) | `MediaVoice` | Duration, Waveform |
| — document con `AttributeAudio{Voice: false}` (priorità 2) | `MediaAudio` | FileName, Size, Duration |
| — document con `AttributeSticker` (priorità 3) | `MediaSticker` | StickerEmoji (alt), StickerPack (StickerSet name) |
| — document con `AttributeVideo` (priorità 4) | `MediaVideo` | FileName, Size, Duration |
| — document con `mime_type: "video/*"` (mime fallback) | `MediaVideo` | FileName, Size |
| — document con `mime_type: "audio/*"` (mime fallback) | `MediaAudio` | FileName, Size |
| — document (default) | `MediaDocument` | FileName, Size, MimeType |
| `tg.MessageMediaGeo` | `MediaLocation` (Step 24: fallback document-style render) | Latitude, Longitude |
| `tg.MessageMediaVenue` | `MediaLocation` (Step 24: fallback) | Latitude, Longitude, VenueName (Title) |
| `tg.MessageMediaContact` | `MediaContact` (Step 24: fallback) | ContactName, ContactPhone |
| `tg.MessageMediaPoll` | `MediaPoll` (Step 24: fallback) | PollQuestion |
| `tg.MessageMediaWebPage` | (nessuno — `Media` resta `nil`) | — |
| `nil` / `tg.MessageMediaEmpty` | (nessuno) | `Message.Media = nil` |

### Voice Waveform

Il waveform di Telegram è un `[]byte` con campioni a **5 bit packed**
little-endian (range `0..31`). Per il rendering inline nel bubble,
l'implementazione lo decodifica e lo mappa a una stringa di **N=10 glifi
fissi** scelti dal block-element Unicode `▁▂▃▄▅▆▇█` (8 livelli).

La specifica formale del mapping è in
[`../phase-4-concurrency/media_waveform.tla`](../phase-4-concurrency/media_waveform.tla)
con invarianti TOTAL / LENGTH / MONOTONIC / SILENCE / SATURATION /
EMPTY_FALLBACK. Il pseudo-codice e le tabelle di mapping sono in
[`../phase-3-interactions/media-flow.md` §Braille Waveform](../phase-3-interactions/media-flow.md).

Razionale charset (block-element vs braille pattern 256-glyph) →
[ADR-011 §Charset choice](../phase-6-decisions/ADR-011-media-rendering-taxonomy.md).

## Reactions Mapping

```
tg.MessageReactions → []domain.Reaction
```

**Step 25 scope**: il convert filtra solo le `tg.ReactionEmoji` (reazioni
emoji standard); le `tg.ReactionCustomEmoji` (emoji custom premium) sono
**skipped** in Step 25 (vedi
[ADR-012](../phase-6-decisions/ADR-012-reactions-storage-and-system-detection.md) §Custom emoji).

| `tg.ReactionCount` field | `domain.Reaction` field | Note |
|--------------------------|-------------------------|------|
| `Reaction.(*ReactionEmoji).Emoticon` | `Emoji` | Stringa emoji (es. `"👍"`); skip ReactionCustomEmoji |
| `Count` | `Count` | Diretto (`int`) |
| `Chosen` | `ChosenByMe` | Diretto (`bool`) |

**Ordering invariante**: la slice prodotta è ordinata per `Count desc,
Emoji asc` (tie-break per determinismo). Questa è una pre-condition del
rendering (vedi `REACTIONS_ORDERED` in
[`../phase-4-concurrency/reactions.tla`](../phase-4-concurrency/reactions.tla)
e §Statechart in
[`../phase-2-behavioral/reactions-and-system.md`](../phase-2-behavioral/reactions-and-system.md)).

**Snapshot semantics**: `UpdateMessageReactions` invia il **set completo**
(non delta). Il client sostituisce l'intera slice
(`m.Reactions = newReactions`, replace, non merge).

## System Message Mapping

```
tg.MessageService{Action tg.MessageActionClass} → domain.Message{IsService: true, ServiceText: ...}
```

**Step 25 scope**: la conversione produce un `domain.Message` con
`IsService = true`, `Text = ""`, `Media = nil`, `Reactions = nil`, e
`ServiceText` pre-formattato dal dispatch dell'azione. Il flag
`IsService` è **immutabile** post-creation (vedi `SYSTEM_IMMUTABLE` in
[`../phase-4-concurrency/reactions.tla`](../phase-4-concurrency/reactions.tla)).

| Caso `tg.MessageClass` | domain field | Note |
|------------------------|--------------|------|
| `*tg.Message` | `IsService = false` | Path Step 11 (text/media/reactions/...) |
| `*tg.MessageService` | `IsService = true` + `ServiceText` formattato | Path Step 25 (centered dim render) |
| `*tg.MessageEmpty` | (nessun message domain emesso) | Skip in convert (placeholder) |

### Action → ServiceText cascade

Il convert (`internal/telegram/convert/service.go`) implementa un
`switch action.(type)` con i kind più comuni mappati a stringhe
user-facing. Lookup degli usernames degli attori avviene in `entities`
(stessa response). Varianti non mappate ricadono nel fallback `"Service
message"` per garantire **totalità**.

| `tg.MessageActionClass` | `ServiceText` template | Esempio |
|-------------------------|------------------------|---------|
| `MessageActionChatAddUser` | `"{name} joined"` (oppure `"{actor} added {target}"` se differiscono) | `"Alice joined"` |
| `MessageActionChatJoinedByLink` | `"{name} joined via invite link"` | `"Bob joined via invite link"` |
| `MessageActionChatJoinedByRequest` | `"{name} joined"` | `"Carol joined"` |
| `MessageActionChatDeleteUser` | `"{name} left"` (oppure `"{actor} removed {target}"`) | `"Dave left"` |
| `MessageActionChatEditTitle` | `"{name} changed title to \"{title}\""` | `"Eve changed title to \"Project X\""` |
| `MessageActionChatEditPhoto` | `"{name} updated group photo"` | `"Frank updated group photo"` |
| `MessageActionChatDeletePhoto` | `"{name} removed group photo"` | `"Grace removed group photo"` |
| `MessageActionPinMessage` | `"{name} pinned a message"` | `"Heidi pinned a message"` |
| `MessageActionChatCreate` | `"{name} created the chat"` | `"Ivy created the chat"` |
| `MessageActionChannelCreate` | `"{name} created the channel"` | `"Jack created the channel"` |
| `MessageActionPhoneCall{Reason}` | call: `"Call · {duration}"`, missed: `"Missed call"`, in/out: prefisso `"↗"`/`"↙"` opzionale | `"Call · 4:32"`, `"Missed call"` |
| `MessageActionScreenshotTaken` | `"{name} took a screenshot"` | `"Kate took a screenshot"` |
| (altre ~20 varianti) | `"Service message"` (fallback generico) | `"Service message"` |

**Invariante di totalità**: ogni `tg.MessageActionClass`, anche
sconosciuto, produce un `ServiceText` non vuoto. Mai panic, mai
stringa vuota.

## Reply Mapping

```
tg.MessageReplyHeader → domain.ReplyInfo
```

| tg field | domain field | Note |
|----------|-------------|------|
| `ReplyToMsgID` | `MessageID` | ID del messaggio citato |
| — | `SenderName` | Lookup dal messaggio originale (può richiedere fetch) |
| — | `Text` | Testo del messaggio originale, troncato a ~50 chars |

**Challenge**: il messaggio originale potrebbe non essere nel viewport attuale. Opzioni:
1. Fetch on-demand quando il reply appare (API call)
2. Cache locale dei messaggi recenti
3. Usare i dati da `tg.MessageReplyHeader` se disponibili (reply_to_msg_id + reply_to_peer_id)

## Forward Mapping

```
tg.MessageFwdHeader → domain.ForwardInfo
```

| tg field | domain field | Note |
|----------|-------------|------|
| `FromID` | `FromID` | PeerUser o PeerChannel |
| `FromName` | `FromName` | Nome se il peer è nascosto (privacy) |
| `Date` | `Date` | Data del messaggio originale |

## InputPeer Construction

Per ogni API call serve un `tg.InputPeerClass`. Il mapping:

```go
func chatIDToInputPeer(id ChatID, accessHash int64) tg.InputPeerClass {
    switch id.PeerType {
    case PeerUser:
        return &tg.InputPeerUser{UserID: id.ID, AccessHash: accessHash}
    case PeerChat:
        return &tg.InputPeerChat{ChatID: id.ID}
    case PeerChannel:
        return &tg.InputPeerChannel{ChannelID: id.ID, AccessHash: accessHash}
    }
}
```

**Critico**: l'`AccessHash` deve essere quello corretto per il peer. Se manca o è sbagliato, il server restituisce `PEER_ID_INVALID`. L'access hash si ottiene da:
- `tg.Entities` nelle risposte API
- Cache locale (peer storage)
- `contacts.resolveUsername` per @username
