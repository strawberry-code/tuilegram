package views

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

func (m ConversationModel) Update(msg tea.Msg) (ConversationModel, tea.Cmd) {
	// Forward picker absorbs all input when active (modal invariant, ADR-007).
	if m.forwardPicker.Active() {
		var cmd tea.Cmd
		m.forwardPicker, cmd = m.forwardPicker.Update(msg)
		return m, cmd
	}
	switch msg := msg.(type) {
	case telegram.PinnedMsgLoadedMsg:
		return m.handlePinnedMsgLoaded(msg)
	case SearchInChatOpenMsg:
		return m.handleSearchOpen()
	case SearchInChatCloseMsg:
		return m.handleSearchClose()
	case SearchInChatNextMsg:
		return m.handleSearchNext()
	case SearchInChatPrevMsg:
		return m.handleSearchPrev()
	case telegram.MessagesLoadedMsg:
		return m.handleMessagesLoaded(msg)
	case telegram.MessagesErrMsg:
		m.loading = false
	case telegram.ForwardPickerReadyMsg:
		m.forwardPicker = m.forwardPicker.Open(msg.Chats)
		return m, nil
	case telegram.MessageSentMsg:
		m.markLastSent(model.StatusDelivered)
	case telegram.ReactionsUpdatedMsg:
		m = m.handleReactionsUpdated(msg)
	case telegram.NewMessageMsg:
		m = m.handleNewMessage(msg)
	case telegram.SelectToggleMsg:
		m = m.toggleSelectionByID(msg.MsgID)
		m.viewport.SetContent(m.renderMessages())
		return m, nil
	case telegram.SelectClearMsg:
		m = m.clearSelection()
		m.viewport.SetContent(m.renderMessages())
		return m, nil
	case spinner.TickMsg:
		if m.loading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	// Propagate non-key msgs to textarea (cursor blink, etc.) when focused.
	if m.inputFocus {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
	// Propagate to searchInput for cursor blink when bar is active.
	if m.searchBar.Active {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m ConversationModel) handleMessagesLoaded(msg telegram.MessagesLoadedMsg) (ConversationModel, tea.Cmd) {
	m.loading = false
	m.messages = msg.Messages
	if len(m.messages) > 0 {
		m.cursor = len(m.messages) - 1
	}
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	m.inputFocus = true
	m.sendBtn.Active = true
	return m, m.textarea.Focus()
}

func (m ConversationModel) handleInputKey(msg tea.KeyMsg) (ConversationModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.inputFocus, m.sendBtn.Active, m.editMode = false, false, false
		m.replyTo = nil
		m.textarea.Reset()
		m.textarea.Blur()
		return m, nil
	case "enter":
		return m.submitText()
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

// toggleSelectionByID è la variante per ID esplicito (da SelectToggleMsg).
func (m ConversationModel) toggleSelectionByID(id int) ConversationModel {
	if _, ok := m.selection[id]; ok {
		delete(m.selection, id)
	} else {
		m.selection[id] = struct{}{}
	}
	m.multiSelect = len(m.selection) > 0
	return m
}
