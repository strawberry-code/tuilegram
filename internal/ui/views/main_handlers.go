package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
)

func (m MainModel) handleNewMessage(msg telegram.NewMessageMsg) (MainModel, tea.Cmd) {
	m.conversation, _ = m.conversation.Update(msg)
	isOpen := m.focus == FocusConversation && m.conversation.chat.ID == msg.ChatID
	m.chatList.UpdateChat(msg.ChatID, isOpen)
	return m, nil
}

func (m MainModel) handleForwardRequest(msg telegram.ForwardRequestMsg) (MainModel, tea.Cmd) {
	ready := telegram.ForwardPickerReadyMsg{Chats: m.chatList.Chats()}
	var cmd tea.Cmd
	m.conversation, cmd = m.conversation.Update(ready)
	return m, cmd
}

func (m MainModel) handleForwardSubmit(msg telegram.ForwardSubmitMsg) (MainModel, tea.Cmd) {
	src := m.conversation.chat
	msgs := m.conversation.forwardSource
	ids := make([]int, len(msgs))
	for i, fmsg := range msgs {
		ids[i] = fmsg.ID
	}
	m.conversation.forwardPicker = m.conversation.forwardPicker.BeginForwarding()
	return m, m.ForwardMessageCmd(msg.Target, src, ids)
}

// handleForwardResult: success clears selection (BatchActionDoneMsg semantics,
// TLA+ ForwardSuccess: selection' = {}, mode' = "browsing").
func (m MainModel) handleForwardResult(msg telegram.ForwardResultMsg) (MainModel, tea.Cmd) {
	m.conversation.forwardPicker = m.conversation.forwardPicker.EndForwarding(msg.Err)
	if msg.Err == nil {
		m.conversation = m.conversation.clearSelection()
		m.conversation.viewport.SetContent(m.conversation.renderMessages())
		m.statusMsg = "Forwarded to " + msg.Target.Title
	} else {
		m.statusMsg = "Forward failed: " + msg.Err.Error()
	}
	return m, nil
}

// handleTyping gestisce UpdateUserTypingMsg: aggiorna lastTypingAt, re-arma tick (ADR-010).
// Sincronizza anche: chatlist typing peer + conversation.Typing flag.
func (m MainModel) handleTyping(msg telegram.UpdateUserTypingMsg) (MainModel, tea.Cmd) {
	now := time.Now()
	m.typing[msg.Peer] = TypingState{LastTypingAt: now, UserID: msg.UserID}
	m.chatList.SetTyping(msg.Peer, true)
	if m.focus == FocusConversation && m.conversation.chat.ID == msg.Peer {
		m.conversation.Typing = true
	}
	return m, scheduleTypingTimeoutCmd(msg.Peer, now)
}

// handleTypingTimeout gestisce TypingTimeoutMsg: clear peer se TTL scaduto (STALE_TICK_BENIGN).
func (m MainModel) handleTypingTimeout(msg TypingTimeoutMsg) (MainModel, tea.Cmd) {
	s, ok := m.typing[msg.Peer]
	if !ok {
		return m, nil // peer già in Idle (tick orfano)
	}
	if time.Since(s.LastTypingAt) < typingTTL {
		return m, nil // stale tick: un update più recente ha esteso il TTL
	}
	delete(m.typing, msg.Peer)
	m.chatList.SetTyping(msg.Peer, false)
	if m.focus == FocusConversation && m.conversation.chat.ID == msg.Peer {
		m.conversation.Typing = false
	}
	return m, nil
}

// centerViewport approssima il vim `zz` (center current line) scrollando il
// viewport in alto di metà altezza, clampato a 0. Senza un cursore per messaggio
// è la migliore approssimazione disponibile.
func (m MainModel) centerViewport() MainModel {
	vp := &m.conversation.viewport
	target := vp.YOffset - vp.Height/2
	if target < 0 {
		target = 0
	}
	vp.SetYOffset(target)
	return m
}
