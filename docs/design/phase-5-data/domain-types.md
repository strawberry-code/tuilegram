# Domain Types

Definizione dei tipi di dominio Go con invarianti e vincoli.

## Core Types

```go
// ChatID identifica univocamente una chat Telegram.
// Telegram usa namespace separati per user, chat e channel.
type ChatID struct {
    PeerType PeerType
    ID       int64
}

type PeerType int

const (
    PeerUser    PeerType = iota
    PeerChat             // basic group
    PeerChannel          // supergroup or channel
)

// ChatType è il tipo semantico della chat (derivato da PeerType + metadata).
type ChatType int

const (
    ChatPrivate       ChatType = iota
    ChatGroup
    ChatChannel
    ChatBot
    ChatSavedMessages
)
```

## Chat

```go
type Chat struct {
    ID            ChatID
    Type          ChatType
    Title         string      // display name (nome utente per private, titolo per gruppi)
    IsMuted       bool
    IsPinned      bool
    IsArchived    bool
    UnreadCount   int
    LastMessage   *Message    // nil se nessun messaggio
    PinnedMessage *Message    // nil se nessun messaggio pinnato
    FolderID      int         // 0 = nessuna cartella
    MemberCount   int         // solo per gruppi/canali
    AccessHash    int64       // necessario per API calls

    // Per chat private
    User *User               // nil per gruppi/canali
}
```

**Invarianti:**
- `UnreadCount >= 0`
- `Type == ChatPrivate` ⟹ `User != nil`
- `Type == ChatGroup || Type == ChatChannel` ⟹ `MemberCount > 0`
- `AccessHash` deve essere non-zero per poter fare API calls su questa chat

## User

```go
type User struct {
    ID         int64
    FirstName  string
    LastName   string
    Username   string   // senza @, può essere vuoto
    Phone      string   // può essere vuoto (privacy)
    Bio        string
    IsBot      bool
    Status     OnlineStatus
    AccessHash int64
}

// DisplayName restituisce il nome da mostrare nella UI.
func (u *User) DisplayName() string {
    if u.LastName != "" {
        return u.FirstName + " " + u.LastName
    }
    return u.FirstName
}

type OnlineStatus struct {
    IsOnline bool
    LastSeen time.Time // zero value se online o sconosciuto
}
```

## Message

```go
type Message struct {
    ID         int
    ChatID     ChatID
    SenderID   int64
    SenderName string       // cache del nome (evita lookup per ogni render)
    Text       string
    Media      *MessageMedia
    ReplyTo    *ReplyInfo
    Forward    *ForwardInfo
    Reactions  []Reaction
    Status     DeliveryStatus
    Date       time.Time
    EditDate   time.Time    // zero value se non editato
    IsOutgoing bool
    IsService  bool         // true per system messages (join, leave, etc.)
    ServiceText string      // testo pre-formattato per system messages
}

type ReplyInfo struct {
    MessageID  int
    SenderName string
    Text       string   // troncato a ~50 chars per la preview
}

type ForwardInfo struct {
    FromID   int64
    FromName string     // nome o @username della fonte
    Date     time.Time
}

type Reaction struct {
    Emoji     string
    Count     int
    ChosenByMe bool
}

type DeliveryStatus int

const (
    StatusSending   DeliveryStatus = iota
    StatusSent                          // ✓
    StatusDelivered                     // ✓✓
    StatusRead                          // ✓✓ (blu)
    StatusFailed                        // ✕
)

// Symbol restituisce la rappresentazione UI dello status.
func (s DeliveryStatus) Symbol() string {
    switch s {
    case StatusSending:   return ""
    case StatusSent:      return "✓"
    case StatusDelivered: return "✓✓"
    case StatusRead:      return "✓✓" // colorato in blu dal renderer
    case StatusFailed:    return "✕"
    default:              return ""
    }
}
```

## MessageMedia

```go
type MediaType int

const (
    MediaPhoto    MediaType = iota
    MediaVideo
    MediaAudio
    MediaVoice
    MediaDocument
    MediaSticker
    MediaLocation
    MediaContact
    MediaPoll
)

type MessageMedia struct {
    Type     MediaType
    FileName string
    Size     int64        // bytes
    MimeType string
    Duration time.Duration // per video, audio, voice
    Waveform []byte        // per voice messages (braille rendering)

    // Sticker-specific
    StickerEmoji string
    StickerPack  string

    // Location-specific
    Latitude  float64
    Longitude float64
    VenueName string

    // Contact-specific
    ContactName  string
    ContactPhone string

    // Poll-specific
    PollQuestion string
}

// Icon restituisce l'emoji da mostrare per questo tipo di media.
func (m *MessageMedia) Icon() string {
    switch m.Type {
    case MediaPhoto:    return "📷"
    case MediaVideo:    return "🎬"
    case MediaAudio:    return "🎵"
    case MediaVoice:    return "🎤"
    case MediaDocument: return "📎"
    case MediaSticker:  return m.StickerEmoji
    case MediaLocation: return "📍"
    case MediaContact:  return "👤"
    case MediaPoll:     return "📊"
    default:            return "📄"
    }
}

// Summary restituisce la stringa da mostrare nella UI.
func (m *MessageMedia) Summary() string {
    // es: "📷 photo.jpg (1.2 MB)"
    // es: "🎤 ▁▂▃▅▇█▆▄▃▂  0:42"
    // implementazione nel renderer
}
```

## ChatFolder

```go
type ChatFolder struct {
    ID    int
    Title string
    Chats []ChatID
}
```

## Search

```go
type SearchHit struct {
    ChatID    ChatID
    ChatTitle string
    MessageID int
    Text      string   // con match evidenziato
    Date      time.Time
}
```

### SearchOverlayState (Step 26)

Stato interno dell'overlay search globale. Tenuto nel root model
(`internal/ui/views/search.go`).

```go
type SearchOverlayState struct {
    Query         string       // query corrente (senza il prefix '/')
    LatestQueryID uint64       // monotonic counter (Step 26, ADR-013)
    Hits          []SearchHit  // risultati dell'ultima RPC con qID == LatestQueryID
    Cursor        int          // indice in Hits, 0..len(Hits)-1
    InFlight      bool         // true se searchCmd è in volo per LatestQueryID
    LastErr       error        // ultimo errore RPC, nil se ok
}
```

**Invarianti (verificati in `search.tla`):**
- `LatestQueryID` è strettamente crescente nell'arco di vita del processo
  (mai reset, mai decremento). Vedi `MONOTONIC_QUERYID`.
- `Cursor < len(Hits)` se `len(Hits) > 0`, altrimenti `Cursor == 0`.
- `Hits` è lo snapshot dell'ultima RPC fresh applicata (con `qID == LatestQueryID`
  al momento dell'applicazione). Risultati con `qID != LatestQueryID` sono
  scartati prima di toccare `Hits`. Vedi `STALE_RESULT_DROP`.
- `InFlight == true ⟹ esiste almeno una searchCmd goroutine pendente`
  (può essere stale; `LatestQueryID` può essere stato bumped nel frattempo
  da nuove keystroke o da Close).

### SearchInChatState (Step 27)

Stato interno della **search locale nella conversazione attiva**. Tenuto
nella `ConversationModel` (NON nel root App, perché è per-chat e si
reset all'apertura di un'altra chat). Vedi
`phase-2-behavioral/search-in-chat.md` e ADR-014.

```go
type SearchInChatState struct {
    Active     bool                // true se barra inline aperta
    Query      string              // contenuto del textinput
    Index      []IndexedMessage    // snapshot filtrato dei msg searchabili
    Matches    []SearchMatch       // hit della query corrente, ordine cronologico
    CurrentIdx int                 // indice in Matches del match focused
    ReturnTo   ConvSubstate        // {browsingMessages, multiSelect}
}

type IndexedMessage struct {
    MsgID  int
    TextLC string  // pre-lowercased per match O(1) per char
    Pos    int     // posizione nella slice messages[] originale
}

type SearchMatch struct {
    MsgID int
    Spans []TextSpan  // posizioni inizio/fine del match nel render
}

type TextSpan struct {
    Start int  // byte offset inclusivo
    End   int  // byte offset esclusivo
}

type ConvSubstate int

const (
    ConvBrowsingMessages ConvSubstate = iota
    ConvMultiSelect
)
```

**Invarianti (verificati in `search_in_chat.tla`):**
- `Active == false ⟹ Index == nil && Matches == nil && CurrentIdx == 0 && Query == ""`
  (vedi `INACTIVE_CLEAN`).
- `Query == "" ⟹ Matches == nil` (vedi `QUERY_EMPTY_NO_MATCHES`).
- Per ogni `m \in Matches`, esiste `i \in Index` con `i.MsgID == m.MsgID`
  (vedi `NO_PHANTOM_MATCH`).
- Per ogni `i \in Index`, il messaggio corrispondente in `messages[]`
  ha `IsService == false && Text != ""` (vedi `INDEX_CONSISTENT_WITH_MESSAGES`,
  `SYSTEM_NOT_INDEXED`).
- `0 <= CurrentIdx < len(Matches)` se `len(Matches) > 0`, altrimenti
  `CurrentIdx == 0` (vedi `CURSOR_BOUNDED`).
- Su `NewMessageMsg` (append) e `LoadMoreMsg` (prepend) mentre `Active`,
  l'identità (msgID) di `Matches[CurrentIdx]` è preservata
  (vedi `MATCH_IDENTITY_PRESERVED_*`).

### Mapping a tg.MessagesSearchGlobal (Step 26)

```
api.MessagesSearchGlobal request → []SearchHit
```

| Sorgente (request) | Valore Step 26 | Note |
|--------------------|----------------|------|
| `Q` | `state.Query` | Query string user-supplied |
| `Filter` | `tg.InputMessagesFilterEmpty{}` | Tutti i tipi di messaggio |
| `MinDate` / `MaxDate` | `0` | Nessun filtro temporale |
| `OffsetRate` / `OffsetPeer` / `OffsetID` | `0` / `EmptyPeer` / `0` | Pagina iniziale, no paging in Step 26 |
| `Limit` | `50` (raccomandato) | Bounded; nessun infinite scroll in Step 26 |
| `BroadcastsOnly` | `false` | Includi tutti i tipi di chat |

| Sorgente (response: `tg.MessagesSlice`) | `SearchHit` field | Note |
|------------------------------------------|-------------------|------|
| `Messages[i].PeerID` | `ChatID` | Mapped via convert layer |
| (lookup `Chats`/`Users` via PeerID) | `ChatTitle` | Display name della chat di provenienza |
| `Messages[i].ID` | `MessageID` | Diretto |
| `Messages[i].Message` | `Text` | Plain text; highlight applicato in render |
| `Messages[i].Date` | `Date` | Unix → `time.Time` |

## Events (tea.Msg types)

```go
// --- Telegram → TUI ---

type ConnectedMsg struct{}
type DisconnectedMsg struct{ Err error }
type ReconnectingMsg struct{ Attempt int }
type AuthRequiredMsg struct{}
type AuthSuccessMsg struct{ Self User }

type DialogsLoadedMsg struct{ Chats []Chat }
type MessagesLoadedMsg struct {
    ChatID   ChatID
    Messages []Message
}

type NewMessageMsg struct{ Message Message }
type MessageEditedMsg struct {
    ChatID    ChatID
    MessageID int
    NewText   string
    EditDate  time.Time
}
type MessageDeletedMsg struct {
    ChatID     ChatID
    MessageIDs []int
}

type UserStatusMsg struct {
    UserID int64
    Status OnlineStatus
}

type TypingMsg struct {
    ChatID ChatID
    UserID int64
}

type ReadHistoryMsg struct {
    ChatID ChatID
    MaxID  int
}

// ReactionsUpdatedMsg — Telegram UpdateMessageReactions (Step 25).
// Snapshot replace: il client sostituisce m.Reactions interamente.
// Reactions è già ordered (count desc, emoji asc) dal convert layer.
// Vedi: phase-2-behavioral/reactions-and-system.md, ADR-012.
type ReactionsUpdatedMsg struct {
    ChatID    ChatID
    MessageID int
    Reactions []Reaction
}

type PinnedMessageMsg struct {
    ChatID  ChatID
    Message *Message // nil se unpinned
}

type ChatUpdateMsg struct{ Chat Chat }

// --- Command Results ---

type SendResultMsg struct {
    TempID    int   // ID temporaneo locale
    MessageID int   // ID assegnato dal server
    Err       error
}

type EditResultMsg struct{ Err error }
type DeleteResultMsg struct{ Err error }
type ForwardResultMsg struct{ Err error }

// SearchResultMsg — risultato di un searchCmd (Step 26).
// QueryID accompagna il payload per consentire al main loop di scartare
// risultati stale (qID != latestQueryID). Vedi ADR-013.
type SearchResultMsg struct {
    QueryID uint64
    Hits    []SearchHit
    Err     error
}
type LoadMoreMsg struct {
    ChatID   ChatID
    Messages []Message
    Err      error
}
type MarkReadResultMsg struct{ Err error }
type MuteResultMsg struct{ Err error }
type PinResultMsg struct{ Err error }
type ArchiveResultMsg struct{ Err error }

// --- Internal UI ---

type ChatSelectedMsg struct{ ChatID ChatID }
type FocusChangedMsg struct{ Panel Panel }
type OverlayOpenMsg struct{ Type OverlayType; Data interface{} }
type OverlayCloseMsg struct{}

// --- Search overlay (Step 26) ---
// Vedi: phase-2-behavioral/search-overlay.md, phase-4-concurrency/search.tla, ADR-013.

type SearchOpenMsg struct{}

// SearchQueryChangedMsg — emessa dal textinput dell'overlay ad ogni cambio.
// QueryID è l'ID monotone-increasing assegnato al main loop (latestQueryID).
type SearchQueryChangedMsg struct {
    Query       string
    QueryID     uint64
    ScheduledAt time.Time
}

// SearchDebounceFiredMsg — risultato di tea.Tick(300ms) schedulato da
// SearchQueryChangedMsg. Verificare QueryID == latestQueryID prima di spawnare
// searchCmd; altrimenti no-op (stale debounce).
type SearchDebounceFiredMsg struct {
    QueryID uint64
}

// SearchCursorMsg — j/k nell'overlay search.
type SearchCursorMsg struct{ Delta int }

// SearchSubmitMsg — Enter su un hit; trigger di OverlayCloseMsg + JumpToMessageMsg.
type SearchSubmitMsg struct{ Hit SearchHit }

// JumpToMessageMsg — apre la chat target (se diversa) e centra il viewport sul
// messaggio. Riusabile in futuro per pinned-bar (Step 30) e link interni.
type JumpToMessageMsg struct {
    ChatID    ChatID
    MessageID int
}
// --- Search in chat (Step 27) ---
// Vedi: phase-2-behavioral/search-in-chat.md, phase-4-concurrency/search_in_chat.tla, ADR-014.
// Convenzione di naming: prefisso `SearchInChat` per evitare collisione con
// gli eventi `Search*` di Step 26 (search globale).

// SearchInChatOpenMsg — Ctrl+F apre la barra inline nella ConversationModel.
type SearchInChatOpenMsg struct{}

// SearchInChatNextMsg — Enter/n per navigare al match successivo.
type SearchInChatNextMsg struct{}

// SearchInChatPrevMsg — Shift+Tab/N per navigare al match precedente.
type SearchInChatPrevMsg struct{}

// SearchInChatCloseMsg — Esc per chiudere la barra. Ripristina ReturnTo.
// Il re-compute di Matches è sincrono nel loop (no Cmd, no Msg separato).
type SearchInChatCloseMsg struct{}

type ReplyToMsg struct{ Message Message }
type EditRequestMsg struct{ Message Message }
type ForwardRequestMsg struct{ Messages []Message }
type DeleteRequestMsg struct{ Messages []Message }
type SelectToggleMsg struct{ MessageID int }
type FolderToggleMsg struct{}
type TickMsg struct{}

type Panel int
const (
    PanelChatList Panel = iota
    PanelMessages
    PanelInput
)

type OverlayType int
const (
    OverlaySearch OverlayType = iota
    OverlaySearchInChat
    OverlayCommandPalette
    OverlayWhichKey
    OverlayConfirm
    OverlayChatInfo
    OverlayEditMessage
    OverlayForwardPicker
    OverlayHelp
)
```

**Nota Step 27**: `OverlaySearchInChat` esiste nel taxonomy
`OverlayType` per coerenza, ma la search-in-chat NON è un vero overlay
(è un sub-state della `ConversationModel` con barra inline). Il valore
enum è preservato per simmetria di routing ma il render non passa per
il livello Overlay del root App. Vedi ADR-014.
