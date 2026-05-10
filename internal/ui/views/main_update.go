package views

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
	"github.com/strawberry-code/tuilegram/internal/theme"
	"github.com/strawberry-code/tuilegram/internal/ui/components"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

func (m MainModel) Update(msg tea.Msg) (MainModel, tea.Cmd) {
	if m2, cmd, handled := m.handleNotify(msg); handled {
		return m2, cmd
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleWindowSize(msg.Width, msg.Height)
	case theme.ChangedMsg:
		// Hot-reload: atomic swap del tema + force re-render (ADR-019 D9, theming.tla §J).
		// Invariante NO_TORN_RELOAD: msg.NewTheme è già total e merged.
		styles.SetActive(msg.NewTheme)
		return m, func() tea.Msg { return tea.WindowSizeMsg{Width: m.Width, Height: m.Height} }
	case theme.WatcherWarnMsg:
		// Tema PRESERVATO (INVALID_PRESERVES_THEME). Solo warning su stderr in Step 31.
		m.statusMsg = "Theme reload failed: " + msg.Reason
		return m, nil
	case LayoutPanelSwitchMsg:
		return m.handleLayoutPanelSwitch()
	case LayoutModeChangedMsg:
		// Fanout notification (ADR-018 §D2): primary side-effects already
		// applied in handleWindowSize. Sub-models consume here when they
		// need to reset state on cross-threshold (none currently).
		return m, nil
	case telegram.DialogsLoadedMsg:
		m.SetChats(msg.Chats) // uses applyFolderFilter (Step 29)
	case telegram.FoldersLoadedMsg:
		return m.handleFoldersLoaded(msg)
	case FolderToggleMsg, FolderCursorMsg, FolderSelectMsg:
		return m.handleFolderMsg(msg)
	case ChatInfoOpenMsg:
		return m.handleChatInfoOpen()
	case ChatInfoCloseMsg, ChatInfoCompletionMsg:
		return m.handleChatInfoMsg(msg)
	case telegram.ConnectedMsg:
		m.chatList.Conn = ConnConnected
	case telegram.DisconnectedMsg:
		m.chatList.Conn = ConnDisconnected
	case telegram.ReconnectingMsg:
		m.chatList.Conn = ConnReconnecting
	case telegram.PinnedMsgLoadedMsg:
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		return m, cmd
	case telegram.OpenLinkMsg:
		return m.handleOpenLink(msg)
	case telegram.MessagesLoadedMsg, telegram.MessagesErrMsg, spinner.TickMsg:
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		return m, cmd
	case telegram.NewMessageMsg:
		return m.handleNewMessage(msg)
	case telegram.SendRequestMsg:
		return m, m.SendMessageCmd(msg.Chat, msg.Text, msg.ReplyToID)
	case telegram.EditRequestMsg:
		return m, m.EditMessageCmd(msg.Chat, msg.MsgID, msg.Text)
	case telegram.DeleteRequestMsg:
		return m, m.DeleteMessageCmd(msg.Chat, msg.MsgID)
	case telegram.BatchDeleteRequestMsg:
		return m, m.DeleteMessagesCmd(msg.Chat, msg.MsgIDs)
	case deleteResultMsg:
		m.statusMsg = buildDeleteStatusMsg(msg.n, msg.err)
		return m, nil
	case telegram.MessageSentMsg:
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		return m, cmd
	case telegram.MessageSentErrMsg:
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		return m, tea.Batch(cmd, EmitNotifyCmd(components.NotifyError, "Send failed"))
	case telegram.ForwardRequestMsg:
		return m.handleForwardRequest(msg)
	case telegram.ForwardSubmitMsg:
		return m.handleForwardSubmit(msg)
	case telegram.ForwardResultMsg:
		return m.handleForwardResult(msg)
	case telegram.OverlayCloseMsg:
		m.conversation.forwardPicker = m.conversation.forwardPicker.Close()
		m.search = m.search.Close()
		return m, nil
	case SearchOpenMsg:
		m.search = m.search.Open()
	case SearchDebounceFiredMsg, SearchResultMsg, SearchErrMsg:
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	case JumpToMessageMsg:
		return m.handleJumpToMessage(msg)
	case telegram.UpdateUserTypingMsg:
		return m.handleTyping(msg)
	case TypingTimeoutMsg:
		return m.handleTypingTimeout(msg)
	case tea.MouseMsg:
		// Step 32: central mouse router (ADR-020 §D1). Release/Motion discarded
		// inside handleMouseMsg (NO_PHANTOM_DRAG invariant).
		return m.handleMouseMsg(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	// Step 28 overlay messages (CmdPalette, WhichKey, Help, scroll nav).
	if m2, cmd, handled := m.handleOverlayMsg(msg); handled {
		return m2, cmd
	}
	var cmd tea.Cmd
	m.chatList, cmd = m.chatList.Update(msg)
	return m, cmd
}
