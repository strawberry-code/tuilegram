package views

import tea "github.com/charmbracelet/bubbletea"

// main_focus_key.go — handleFocusKey: panel-aware key delegation.
// Extracted from main_key_handlers.go to satisfy the 120-LOC limit.
// Step 29: FocusFolders routing + Shift+Tab back to sidebar (ADR-016 §A).
// Step 30: compact Esc/h back to chat list without closing chat (ADR-018 §D4).

// handleFocusKey delegates keyboard input to whichever panel currently has
// focus. Priority order: FocusFolders → FocusConversation → FocusChatList.
func (m MainModel) handleFocusKey(msg tea.KeyMsg) (MainModel, tea.Cmd) {
	key := msg.String()

	// Folder sidebar has focus.
	if m.focus == FocusFolders {
		var cmd tea.Cmd
		m.folderModel, cmd = m.folderModel.Update(msg)
		// Tab/h/Esc from sidebar → transfer focus to chatList.
		if !m.folderModel.HasFocus() {
			m.focus = FocusChatList
		}
		return m, cmd
	}

	// Conversation focus.
	if m.focus == FocusConversation {
		noOverlay := !m.conversation.inputFocus && !m.conversation.forwardPicker.Active()
		browsing := noOverlay && !m.conversation.multiSelect && !m.conversation.deleteMode
		if browsing && (key == "h" || key == "esc") {
			// Step 30: in Compact, Esc/h = back to chat list, activeChatID preserved.
			if m.layoutMode == LayoutCompact {
				return m.handleCompactEsc()
			}
			m.conversation.Close()
			m.focus = FocusChatList
			return m, nil
		}
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		return m, cmd
	}

	// Chat list focus.
	// Shift+Tab → give focus back to sidebar if visible (ADR-016 §A).
	if key == "shift+tab" && m.folderModel.IsVisible() {
		m.folderModel.GiveFocusBack()
		m.focus = FocusFolders
		return m, nil
	}
	if key == "enter" {
		// Step 30: in Compact, Enter on chat also switches compactVisible atomically.
		if m.layoutMode == LayoutCompact {
			return m.handleCompactEnter()
		}
		if chat, ok := m.chatList.SelectedChat(); ok {
			spinnerCmd := m.conversation.OpenChat(chat)
			m.focus = FocusConversation
			// Step 33: carica anche il pinned message (se esiste) insieme ai messaggi.
			return m, tea.Batch(spinnerCmd, m.LoadMessagesCmd(chat), m.LoadPinnedMessageCmd(chat))
		}
	}
	var cmd tea.Cmd
	m.chatList, cmd = m.chatList.Update(msg)
	return m, cmd
}
