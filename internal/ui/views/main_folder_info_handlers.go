package views

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// handleFoldersLoaded stores server-side folder list and re-applies filter.
func (m MainModel) handleFoldersLoaded(msg telegram.FoldersLoadedMsg) (MainModel, tea.Cmd) {
	m.folderModel.SetFolders(msg.Folders)
	m.applyFolderFilter()
	return m, nil
}

// handleFolderMsg routes FolderToggleMsg / FolderCursorMsg / FolderSelectMsg.
// FolderSelectMsg triggers a synchronous chat-list re-filter (ADR-016 §C).
func (m MainModel) handleFolderMsg(msg tea.Msg) (MainModel, tea.Cmd) {
	var cmd tea.Cmd
	m.folderModel, cmd = m.folderModel.Update(msg)

	switch msg.(type) {
	case FolderToggleMsg:
		// Sync focus: sidebar opened → FocusFolders; closed → FocusChatList.
		if m.folderModel.IsVisible() {
			m.focus = FocusFolders
		} else {
			m.focus = FocusChatList
		}
		// Recalculate panel widths after sidebar visibility change.
		m.SetSize(m.Width, m.Height)

	case FolderSelectMsg:
		// Re-filter chat list synchronously (FILTER_SYNC invariant).
		m.applyFolderFilter()
	}

	// Propagate any cmd emitted by FolderModel.Update (e.g. inner FolderSelectMsg).
	if cmd != nil {
		return m, cmd
	}
	return m, nil
}

// handleChatInfoOpen opens the chat info overlay for the active chat.
// Guards: activeOverlay == none AND conversation is active (ADR-017 §D2).
func (m MainModel) handleChatInfoOpen() (MainModel, tea.Cmd) {
	if m.anyOverlayActive() {
		return m, nil // mutex: one overlay at a time (ADR-015 §D3)
	}
	if !m.conversation.active {
		return m, nil // guard: need an open chat (INFO_REQUIRES_OPEN_CHAT)
	}
	chat := m.conversation.chat
	var needsFetch bool
	m.chatInfo, needsFetch = m.chatInfo.Open(chat)
	m.chatInfo.RefreshContent()
	if needsFetch {
		return m, fetchFullUserCmd(m.Bridge, chat.ID)
	}
	return m, nil
}

// handleChatInfoMsg routes ChatInfoCloseMsg and ChatInfoCompletionMsg.
func (m MainModel) handleChatInfoMsg(msg tea.Msg) (MainModel, tea.Cmd) {
	var cmd tea.Cmd
	m.chatInfo, cmd = m.chatInfo.Update(msg)
	return m, cmd
}
