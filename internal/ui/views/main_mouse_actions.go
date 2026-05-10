package views

// main_mouse_actions.go — semantic click action handlers for the mouse router.
// Step 32 (ADR-020 §D4, §D6, §D10).
//
// KEYBOARD_PARITY invariant: every handler here calls the same helper as the
// keyboard equivalent path (not a duplicate). Cross-references noted inline.

import tea "github.com/charmbracelet/bubbletea"

// handleClickSend handles a click on the SEND button (ADR-020 §D10).
// Delegates to conversation.submitText() — same path as keyboard Enter.
// KEYBOARD_PARITY invariant (ADR-020 §D8): no duplicated logic.
// SENDBUTTON_INACTIVE_NO_OP: no-op when sendBtn.Active == false (empty text).
func (m MainModel) handleClickSend() (MainModel, tea.Cmd) {
	if !m.conversation.sendBtn.Active {
		return m, nil // SENDBUTTON_INACTIVE_NO_OP (ADR-020 §D10)
	}
	m = m.shiftFocusToConversation() // D6 focus shift
	var cmd tea.Cmd
	m.conversation, cmd = m.conversation.submitText()
	return m, cmd
}

// handleClickTextarea handles a click on the textarea area (not the SEND button).
// Keyboard equivalent: "i" or "tab" in handleNavKey → inputFocus := true.
// Action: focus shift + activate input mode. Does NOT submit (ADR-020 §D10).
func (m MainModel) handleClickTextarea() (MainModel, tea.Cmd) {
	m = m.shiftFocusToConversation()
	if m.conversation.active && !m.conversation.inputFocus {
		m.conversation.inputFocus = true
		m.conversation.sendBtn.Active = true
		return m, m.conversation.textarea.Focus()
	}
	return m, nil
}

// handleClickConversation handles a click on the conversation viewport or header.
// Keyboard equivalent: Tab to FocusConversation (handleFocusKey).
// Action: focus shift only — no semantic action on viewport click (ADR-020 §D6).
func (m MainModel) handleClickConversation() (MainModel, tea.Cmd) {
	if m.focus != FocusConversation {
		m.focus = FocusConversation
	}
	return m, nil
}

// handleClickChatList handles a click on a chatList row.
// Keyboard equivalent: j/k to move cursor + Enter to open (handleFocusKey).
// Action: select the clicked row + open chat atomically (ADR-020 §D4).
// CLICK_FOCUS_SHIFT: if focus was elsewhere, shift first (ADR-020 §D6).
func (m MainModel) handleClickChatList(absY int) (MainModel, tea.Cmd) {
	// Focus shift (D6): move focus to chatList panel first.
	if m.focus != FocusChatList {
		m.focus = FocusChatList
	}
	// Map absolute Y → chatList row index.
	idx := chatListRowIndex(m.chatList, absY)
	if idx < 0 {
		return m, nil // click on header/gap row
	}
	// Move cursor to the clicked chat.
	m.chatList.selected = idx
	m.chatList.ensureVisible()
	// Open the chat atomically (same path as Enter, ADR-020 §D4).
	if m.layoutMode == LayoutCompact {
		return m.handleCompactEnter()
	}
	chat, ok := m.chatList.SelectedChat()
	if !ok {
		return m, nil
	}
	spinnerCmd := m.conversation.OpenChat(chat)
	m.focus = FocusConversation
	m.recomputeBboxes() // conversation now active → fill conv bboxes
	return m, tea.Batch(spinnerCmd, m.LoadMessagesCmd(chat))
}

// handleClickFolder handles a click on a folder sidebar row.
// Keyboard equivalent: j/k + Enter in sidebar (folder_update.go handleKey).
// Action: select the clicked folder, emit FolderSelectMsg.
func (m MainModel) handleClickFolder(absY int) (MainModel, tea.Cmd) {
	// Focus shift (D6).
	if m.focus != FocusFolders {
		m.focus = FocusFolders
		m.folderModel.GiveFocusBack()
	}
	idx := folderRowIndex(m.folderModel, absY)
	if idx < 0 || idx >= len(m.folderModel.allFolders) {
		return m, nil
	}
	m.folderModel.cursor = idx
	f := m.folderModel.cursorFolder()
	m.folderModel.selectedID = f.ID
	m.applyFolderFilter()
	return m, func() tea.Msg { return FolderSelectMsg{FolderID: f.ID} }
}

// shiftFocusToConversation applies D6 focus shift + Compact panel switch.
func (m MainModel) shiftFocusToConversation() MainModel {
	m.focus = FocusConversation
	if m.layoutMode == LayoutCompact && m.compactVisible != CompactConversation {
		m.compactVisible = CompactConversation
		m.applyCompactSizes()
		m.recomputeBboxes()
	}
	return m
}
