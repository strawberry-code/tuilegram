package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleKey dispatches nav/action keys.
// Priority: ctrl+f → search open; search active → searchKey;
// deleteMode → deleteKey; inputFocus → inputKey; else navKey.
func (m ConversationModel) handleKey(msg tea.KeyMsg) (ConversationModel, tea.Cmd) {
	if msg.String() == "ctrl+f" {
		return m.handleSearchOpen()
	}
	if m.searchBar.Active {
		return m.handleSearchKey(msg)
	}
	if m.deleteMode {
		return m.handleDeleteKey(msg)
	}
	if m.inputFocus {
		return m.handleInputKey(msg)
	}
	return m.handleNavKey(msg)
}

// handleSearchKey gestisce i tasti con la barra search aperta.
// Esc=close; Enter/n=next; shift+tab/N=prev; altri char → textinput query.
func (m ConversationModel) handleSearchKey(msg tea.KeyMsg) (ConversationModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.handleSearchClose()
	case "enter", "n":
		return m.handleSearchNext()
	case "shift+tab", "N":
		return m.handleSearchPrev()
	case "ctrl+f":
		return m, nil // barra già aperta: no-op
	}
	prevQuery := m.searchBar.Query
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	newQuery := m.searchInput.Value()
	if newQuery == prevQuery {
		return m, cmd
	}
	updated, searchCmd := m.handleSearchQueryChanged(newQuery)
	return updated, tea.Batch(cmd, searchCmd)
}

func (m ConversationModel) handleNavKey(msg tea.KeyMsg) (ConversationModel, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.messages)-1 {
			m.cursor++
			m.viewport.SetContent(m.renderMessages())
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.viewport.SetContent(m.renderMessages())
		}
	case "ctrl+d":
		m.viewport.HalfPageDown()
	case "ctrl+u":
		m.viewport.HalfPageUp()
	case " ":
		m = m.toggleSelection()
		m.viewport.SetContent(m.renderMessages())
	case "r":
		if !m.multiSelect {
			return m.beginReply()
		}
	case "e":
		if !m.multiSelect {
			return m.beginEdit()
		}
	case "D":
		return m.beginBatchDelete()
	case "f":
		return m.beginBatchForward()
	case "esc":
		if m.multiSelect {
			m = m.clearSelection()
			m.viewport.SetContent(m.renderMessages())
		}
	case "i", "tab":
		m.inputFocus = true
		m.sendBtn.Active = true
		return m, m.textarea.Focus()
	}
	return m, nil
}

func (m ConversationModel) beginReply() (ConversationModel, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.messages) {
		return m, nil
	}
	msg := m.messages[m.cursor]
	m.replyTo = &msg
	m.inputFocus = true
	m.sendBtn.Active = true
	return m, m.textarea.Focus()
}

func (m ConversationModel) beginEdit() (ConversationModel, tea.Cmd) {
	if m.cursor >= 0 && m.cursor < len(m.messages) && m.messages[m.cursor].IsOutgoing {
		m.editMsgIdx = m.cursor
		return m.activateEdit()
	}
	return m, nil
}
