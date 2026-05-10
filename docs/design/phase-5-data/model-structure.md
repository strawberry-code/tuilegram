# Model Structure

Struttura del `tea.Model` root e di tutti i sub-model.

## App Model (root)

```go
type AppModel struct {
    // State machine
    state     AppState
    prevState AppState  // per Esc / back navigation

    // Sub-models
    auth       AuthModel
    chatList   ChatListModel
    conversation ConversationModel
    statusBar  StatusBarModel
    folders    FolderModel

    // Overlays (nil quando chiusi)
    activeOverlay OverlayType
    search       *SearchModel
    cmdPalette   *CommandPaletteModel
    whichKey     *WhichKeyModel
    confirmDlg   *ConfirmModel
    chatInfo     *ChatInfoModel
    editOverlay  *EditOverlayModel
    forwardPicker *ForwardPickerModel
    helpOverlay  *HelpModel

    // Layout
    width  int
    height int
    layout LayoutMode

    // Focus
    focusedPanel Panel

    // Telegram bridge
    telegram *TelegramBridge  // reference per creare tea.Cmd

    // Data
    chats       []Chat       // lista ordinata
    chatIndex   map[ChatID]*Chat
    openChatID  *ChatID      // chat attualmente aperta (nil se nessuna)
    selfUser    *User        // utente loggato

    // Connection
    connectionStatus ConnectionStatus
}

type AppState int
const (
    StateInitializing AppState = iota
    StateAuth
    StateLoading
    StateMain
)

type LayoutMode int
const (
    LayoutTwoPanel   LayoutMode = iota  // width >= 100
    LayoutSinglePanel                    // width < 100
)

type ConnectionStatus int
const (
    StatusConnected    ConnectionStatus = iota
    StatusDisconnected
    StatusReconnecting
)
```

## Auth Model

```go
type AuthModel struct {
    step     AuthStep
    phone    textinput.Model
    otp      OTPInputModel      // custom component
    password textinput.Model
    error    string              // ultimo errore da mostrare
    loading  bool                // true durante verifica
    spinner  spinner.Model

    // Figlet title (pre-rendered)
    titleArt string
}

type AuthStep int
const (
    AuthStepPhone AuthStep = iota
    AuthStepCode
    AuthStepPassword
    AuthStepVerifying
)
```

## ChatList Model

```go
type ChatListModel struct {
    items    []ChatListItem
    selected int               // indice dell'item selezionato
    viewport viewport.Model    // scroll della lista
    width    int
    height   int
}

type ChatListItem struct {
    Chat      *Chat
    IsActive  bool     // true se è la chat aperta
    IsTyping  bool     // true se qualcuno sta scrivendo
    TypingUser string  // nome di chi sta scrivendo
}
```

## Conversation Model

```go
type ConversationModel struct {
    // Header
    chatTitle    string
    chatType     ChatType
    onlineStatus string     // "● online", "last seen ...", "12 members"
    isTyping     bool
    typingUser   string

    // Pinned
    pinnedMessage *Message

    // Messages
    messages  []Message
    viewport  viewport.Model   // scroll dei messaggi
    cursor    int              // indice del messaggio sotto il cursore (-1 se nessuno)

    // Multi-select
    selected  map[int]bool     // messageID → selected
    selectMode bool

    // Input
    input     textarea.Model   // multiline, max 5 righe
    replyTo   *Message         // messaggio a cui si sta rispondendo (nil se nessuno)
    sendBtn   ButtonModel      // custom SEND button

    // Layout
    width    int
    height   int
}
```

## Overlay Models

```go
type SearchModel struct {
    input   textinput.Model
    results []SearchHit
    selected int
    scope   SearchScope  // AllChats o ThisChat
    loading bool
    spinner spinner.Model
}

type SearchScope int
const (
    SearchAllChats SearchScope = iota
    SearchThisChat
)

type CommandPaletteModel struct {
    input    textinput.Model
    commands []Command
    filtered []Command
    selected int
}

type Command struct {
    Name        string
    Description string
    Action      func() tea.Msg
    Keybinding  string  // hint visivo
}

type WhichKeyModel struct {
    prefix  string
    options []WhichKeyOption
    timer   time.Duration
}

type WhichKeyOption struct {
    Key         string
    Description string
}

type ConfirmModel struct {
    prompt   string
    onYes    func() tea.Msg
    selected bool  // true = Yes focused
}

type ChatInfoModel struct {
    user     *User
    chat     *Chat
    viewport viewport.Model
}

type EditOverlayModel struct {
    messageID int
    chatID    ChatID
    input     textarea.Model
    original  string
}

type ForwardPickerModel struct {
    messages []Message   // messaggi da forwardare
    input    textinput.Model
    chats    []Chat      // lista filtrata
    selected int
}

type HelpModel struct {
    viewport viewport.Model
    content  string  // pre-rendered help text
}
```

## Custom Components

```go
// OTPInputModel — componente custom per input 2FA a celle individuali
type OTPInputModel struct {
    cells    []rune
    length   int      // numero di celle
    cursor   int      // cella attiva
    focused  bool
}

// ButtonModel — bottone SEND cliccabile
type ButtonModel struct {
    label   string
    focused bool
    pressed bool
    width   int
    height  int
}

// StatusBarModel
type StatusBarModel struct {
    shortcuts string   // keybinding contestuali (parte sinistra)
    message   string   // errore o info (parte destra)
    messageAt time.Time // quando è stato impostato il messaggio
    width     int
}

// FolderModel
type FolderModel struct {
    folders  []ChatFolder
    selected int
    visible  bool
    width    int
    height   int
}
```

## Gerarchia Model

```
AppModel
├── AuthModel
│   ├── textinput.Model (phone)
│   ├── OTPInputModel (code) ← custom
│   ├── textinput.Model (password)
│   └── spinner.Model
├── ChatListModel
│   ├── []ChatListItem
│   └── viewport.Model
├── ConversationModel
│   ├── viewport.Model (messages)
│   ├── textarea.Model (input)
│   ├── ButtonModel (SEND) ← custom
│   └── map[int]bool (selected)
├── StatusBarModel
├── FolderModel
└── Overlays (nullable)
    ├── SearchModel
    │   ├── textinput.Model
    │   └── spinner.Model
    ├── CommandPaletteModel
    │   └── textinput.Model
    ├── WhichKeyModel
    ├── ConfirmModel
    ├── ChatInfoModel
    │   └── viewport.Model
    ├── EditOverlayModel
    │   └── textarea.Model
    ├── ForwardPickerModel
    │   └── textinput.Model
    └── HelpModel
        └── viewport.Model
```

## Update Routing

```go
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    // 1. Global keys (Ctrl+Q, WindowSizeMsg) — always handled
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width, m.height = msg.Width, msg.Height
        m.layout = computeLayout(m.width)
        // resize all sub-models
    case tea.KeyPressMsg:
        if msg.String() == "ctrl+q" {
            return m, tea.Quit
        }
    }

    // 2. Overlay captures all input when active
    if m.activeOverlay != OverlayNone {
        return m.updateOverlay(msg)
    }

    // 3. State-specific routing
    switch m.state {
    case StateAuth:
        return m.updateAuth(msg)
    case StateLoading:
        return m.updateLoading(msg)
    case StateMain:
        return m.updateMain(msg)
    }

    return m, tea.Batch(cmds...)
}

func (m AppModel) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Telegram events — always processed regardless of focus
    switch msg.(type) {
    case NewMessageMsg, MessageEditedMsg, MessageDeletedMsg,
         UserStatusMsg, TypingMsg, ReadHistoryMsg, ReactionsMsg,
         PinnedMessageMsg, ChatUpdateMsg,
         ConnectedMsg, DisconnectedMsg, ReconnectingMsg:
        return m.handleTelegramEvent(msg)
    }

    // Global keybindings (/, Ctrl+P, ?, F, i)
    if cmd := m.handleGlobalKey(msg); cmd != nil {
        return m, cmd
    }

    // Focus-specific routing
    switch m.focusedPanel {
    case PanelChatList:
        return m.updateChatList(msg)
    case PanelMessages:
        return m.updateMessages(msg)
    case PanelInput:
        return m.updateInput(msg)
    }

    return m, nil
}
```
