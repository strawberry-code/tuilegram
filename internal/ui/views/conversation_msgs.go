package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

func (m *ConversationModel) markLastSent(status model.DeliveryStatus) {
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].IsOutgoing && m.messages[i].Status == model.StatusSent {
			m.messages[i].Status = status
			m.viewport.SetContent(m.renderMessages())
			break
		}
	}
}

// submitText commits whatever's in the composer: edits if editMode, else appends.
// Shared by keyboard Enter (conversation_update.go) and mouse SEND click
// (main_mouse_actions.go) — KEYBOARD_PARITY invariant (ADR-020 §D8).
func (m ConversationModel) submitText() (ConversationModel, tea.Cmd) {
	if m.editMode {
		return m.submitEdit()
	}
	text := m.textarea.Value()
	if text == "" {
		return m, nil
	}
	m.textarea.Reset()
	m.replyTo = nil
	return m.appendOptimistic(text)
}

func (m ConversationModel) appendOptimistic(text string) (ConversationModel, tea.Cmd) {
	var replyToID int
	if m.replyTo != nil {
		replyToID = m.replyTo.ID
	}
	msg := model.Message{
		SenderName: "You",
		Text:       text,
		Date:       time.Now(),
		IsOutgoing: true,
		Status:     model.StatusSent,
		ReplyToID:  replyToID,
	}
	m.replyTo = nil
	m.messages = append(m.messages, msg)
	m.cursor = len(m.messages) - 1
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
	return m, func() tea.Msg {
		return telegram.SendRequestMsg{Chat: m.chat, Text: text, ReplyToID: replyToID}
	}
}

// activateEdit entra in modalità edit per il messaggio in m.editMsgIdx.
func (m ConversationModel) activateEdit() (ConversationModel, tea.Cmd) {
	orig := m.messages[m.editMsgIdx]
	m.editMode = true
	m.inputFocus = true
	m.sendBtn.Active = false
	m.textarea.SetValue(orig.Text)
	return m, m.textarea.Focus()
}

// submitEdit applica la modifica ottimisticamente e avvia l'API call.
func (m ConversationModel) submitEdit() (ConversationModel, tea.Cmd) {
	text := m.textarea.Value()
	if text == "" {
		return m, nil
	}
	m.messages[m.editMsgIdx].Text = text
	m.editMode = false
	m.inputFocus = false
	m.textarea.Reset()
	m.textarea.Blur()
	m.viewport.SetContent(m.renderMessages())
	chat := m.chat
	msgID := m.messages[m.editMsgIdx].ID
	return m, func() tea.Msg {
		return telegram.EditRequestMsg{Chat: chat, MsgID: msgID, Text: text}
	}
}

// handleNewMessage aggiunge un messaggio in arrivo al viewport.
// Re-indicizza incrementalmente se la search bar è attiva (ADR-014 §D2).
func (m ConversationModel) handleNewMessage(msg telegram.NewMessageMsg) ConversationModel {
	if !m.active || m.chat.ID != msg.ChatID {
		return m
	}
	atBottom := m.viewport.AtBottom()
	m.messages = append(m.messages, msg.Message)
	m.reindexNewMessage(msg.Message) // search re-index: currentIdx invariato
	m.viewport.SetContent(m.renderMessages())
	if atBottom {
		m.viewport.GotoBottom()
	}
	return m
}
