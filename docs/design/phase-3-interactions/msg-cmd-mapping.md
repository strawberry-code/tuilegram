# Message & Command Mapping

Mapping esaustivo tra input utente, messaggi interni, comandi asincroni e transizioni di stato.

## Input → Message → State Transition

### ChatList Focus

| Input | tea.Msg | State Transition | Cmd emesso |
|-------|---------|------------------|------------|
| `j` / `↓` | `tea.KeyPressMsg` | selected_index++ | — |
| `k` / `↑` | `tea.KeyPressMsg` | selected_index-- | — |
| `Enter` / `l` | `tea.KeyPressMsg` | → `ChatSelectedMsg` → focus=Conversation | `loadChatCmd` |
| `g,g` | `tea.KeyPressMsg` ×2 | selected_index = 0 | — |
| `G` | `tea.KeyPressMsg` | selected_index = len-1 | — |
| `d` | `tea.KeyPressMsg` | — | `markReadCmd` |
| `p` | `tea.KeyPressMsg` | — | `pinCmd` |
| `m` | `tea.KeyPressMsg` | toggle muted | `muteCmd` |
| `a` | `tea.KeyPressMsg` | remove from list | `archiveCmd` |
| `n` | `tea.KeyPressMsg` | → NewConversationOverlay | — |
| `Ctrl+U` | `tea.KeyPressMsg` | scroll half page up | — |
| `Ctrl+D` | `tea.KeyPressMsg` | scroll half page down | — |
| `Tab` | `tea.KeyPressMsg` | focus → Conversation (messages) | — |
| scroll wheel | `tea.MouseWheelMsg` | scroll viewport | — |
| click item | `tea.MouseClickMsg` | select + open | `loadChatCmd` |

### Conversation Focus (Messages Viewport)

| Input | tea.Msg | State Transition | Cmd emesso |
|-------|---------|------------------|------------|
| `j` / `↓` | `tea.KeyPressMsg` | cursor_index++ | — |
| `k` / `↑` | `tea.KeyPressMsg` | cursor_index-- | — |
| `h` / `Esc` | `tea.KeyPressMsg` | focus → ChatList | — |
| `Tab` / `i` | `tea.KeyPressMsg` | focus → Input | — |
| `r` | `tea.KeyPressMsg` | → `ReplyToMsg` → focus=Input | — |
| `e` | `tea.KeyPressMsg` | → `EditRequestMsg` → EditOverlay | — |
| `f` | `tea.KeyPressMsg` | → `ForwardRequestMsg` → ForwardPicker | — |
| `D` | `tea.KeyPressMsg` | → `DeleteRequestMsg` → ConfirmDialog | — |
| `y` | `tea.KeyPressMsg` | copy text to clipboard | — |
| `Space` | `tea.KeyPressMsg` | toggle select on cursor msg | — |
| `g,g` | `tea.KeyPressMsg` ×2 | scroll to top | `loadHistoryCmd` |
| `G` | `tea.KeyPressMsg` | scroll to bottom | — |
| scroll wheel | `tea.MouseWheelMsg` | scroll viewport | — |

### Input Focus

| Input | tea.Msg | State Transition | Cmd emesso |
|-------|---------|------------------|------------|
| text chars | `tea.KeyPressMsg` | append to buffer | — |
| `Enter` | `tea.KeyPressMsg` | clear buffer, → Composing | `sendMessageCmd` |
| `Shift+Enter` | `tea.KeyPressMsg` | newline in buffer | — |
| `Esc` | `tea.KeyPressMsg` | cancel reply/focus → Messages | — |
| `Ctrl+A` | `tea.KeyPressMsg` | → AttachFilePicker | — |
| `↑` (empty) | `tea.KeyPressMsg` | → EditOverlay(last own msg) | — |
| click SEND | `tea.MouseClickMsg` | same as Enter | `sendMessageCmd` |

### Global (any focus)

| Input | tea.Msg | State Transition | Cmd emesso |
|-------|---------|------------------|------------|
| `/` | `tea.KeyPressMsg` | → SearchOverlay | — |
| `Ctrl+F` | `tea.KeyPressMsg` | → SearchInChatOverlay | — |
| `Ctrl+P` | `tea.KeyPressMsg` | → CommandPalette | — |
| `?` | `tea.KeyPressMsg` | → HelpOverlay | — |
| `Ctrl+Q` | `tea.KeyPressMsg` | → quit | `tea.Quit` |
| `Ctrl+L` | `tea.KeyPressMsg` | full redraw | `tea.ClearScreen` |
| `F` | `tea.KeyPressMsg` | toggle folder sidebar | — |
| `i` (from conv) | `tea.KeyPressMsg` | → ChatInfoOverlay | — |
| `WindowSizeMsg` | `tea.WindowSizeMsg` | recalc layout | — |

## Telegram Event → UI Update

| Telegram Event (via p.Send) | UI Update |
|-----------------------------|-----------|
| `ConnectedMsg` | header CHATS → `●`, clear status bar error |
| `DisconnectedMsg` | header CHATS → `✕`, show error in status bar |
| `ReconnectingMsg` | header CHATS → `○` |
| `DialogsLoadedMsg{chats}` | populate ChatList, re-sort |
| `MessagesLoadedMsg{chatID, msgs}` | populate Conversation viewport |
| `NewMessageMsg{msg}` | append to conv (if open), update ChatList item |
| `MessageEditedMsg{chatID, msgID, text}` | update message text in viewport |
| `MessageDeletedMsg{chatID, msgID}` | remove message from viewport |
| `UserStatusMsg{userID, status}` | update dot in ChatList, header in conv |
| `TypingMsg{chatID, userID}` | show "typing..." in ChatList + conv header |
| `ReadHistoryMsg{chatID, maxID}` | update receipt icons (✓✓ blu) |
| `ReactionsMsg{chatID, msgID, reactions}` | update reaction row under message |
| `PinnedMessageMsg{chatID, msg}` | show/update pinned bar under conv header |
| `ChatUpdateMsg{chat}` | update ChatList item (muted, pinned, etc.) |

## Command Result → UI Update

| Command Result | Success UI Update | Error UI Update |
|----------------|-------------------|-----------------|
| `SendResultMsg` | message status → Sent (✓) | status bar: "Send failed (r to retry)" |
| `EditResultMsg` | update message text, close overlay | status bar: "Edit failed" |
| `DeleteResultMsg` | remove message from viewport | status bar: "Delete failed" |
| `ForwardResultMsg` | close overlay, status bar: "Forwarded" | status bar: "Forward failed" |
| `SearchResultMsg` | populate search overlay results | status bar: "Search failed" |
| `LoadMoreMsg` | prepend messages to viewport | status bar: "Failed to load history" |
| `MarkReadResultMsg` | update unread count | — |
| `MuteResultMsg` | update chat item style | status bar: "Failed to mute" |
| `PinResultMsg` | update chat item | status bar: "Failed to pin" |
| `ArchiveResultMsg` | remove from chat list | status bar: "Failed to archive" |

## Update Propagation Matrix

Quale componente viene aggiornato da quale messaggio:

| Msg \ Component | ChatList | Conv Header | Messages VP | Input | Status Bar | Overlay |
|-----------------|----------|-------------|-------------|-------|------------|---------|
| NewMessageMsg | ✓ sort | — | ✓ append | — | — | — |
| UserStatusMsg | ✓ dot | ✓ status | — | — | — | — |
| TypingMsg | ✓ name | ✓ typing | — | — | — | — |
| ReadHistoryMsg | ✓ unread | — | ✓ receipts | — | — | — |
| ConnectedMsg | ✓ header | — | — | — | ✓ clear | — |
| DisconnectedMsg | ✓ header | — | — | — | ✓ error | — |
| SendResultMsg | — | — | ✓ status | — | ✓ if error | — |
| SearchResultMsg | — | — | — | — | — | ✓ results |
| WindowSizeMsg | ✓ resize | ✓ resize | ✓ resize | ✓ resize | ✓ resize | ✓ resize |
