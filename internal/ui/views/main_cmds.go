package views

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// LoadDialogsCmd crea il comando per caricare i dialogs.
func (m MainModel) LoadDialogsCmd() tea.Cmd {
	return func() tea.Msg {
		chats, err := m.Bridge.LoadDialogs(context.Background())
		if err != nil {
			return telegram.DialogsErrMsg{Err: err}
		}
		return telegram.DialogsLoadedMsg{Chats: chats}
	}
}

// LoadMessagesCmd crea il comando per caricare i messaggi di una chat.
func (m MainModel) LoadMessagesCmd(chat model.Chat) tea.Cmd {
	return func() tea.Msg {
		msgs, err := m.Bridge.LoadMessages(context.Background(), chat)
		if err != nil {
			return telegram.MessagesErrMsg{Err: err}
		}
		return telegram.MessagesLoadedMsg{Messages: msgs}
	}
}

// SendMessageCmd crea il comando per inviare un messaggio.
func (m MainModel) SendMessageCmd(chat model.Chat, text string, replyToID int) tea.Cmd {
	return func() tea.Msg {
		err := m.Bridge.SendMessage(context.Background(), chat, text, replyToID)
		if err != nil {
			return telegram.MessageSentErrMsg{Err: err}
		}
		return telegram.MessageSentMsg{}
	}
}

// EditMessageCmd crea il comando per modificare un messaggio.
func (m MainModel) EditMessageCmd(chat model.Chat, msgID int, text string) tea.Cmd {
	return func() tea.Msg {
		_ = m.Bridge.EditMessage(context.Background(), chat, msgID, text)
		return nil
	}
}

// DeleteMessageCmd crea il comando per cancellare un singolo messaggio (Step 20 compat).
func (m MainModel) DeleteMessageCmd(chat model.Chat, msgID int) tea.Cmd {
	return func() tea.Msg {
		_ = m.Bridge.DeleteMessage(context.Background(), chat, msgID)
		return nil
	}
}

// ForwardMessageCmd crea il comando per inoltrare messaggi via Bridge.ForwardMessage.
// BATCH_ATOMICITY: una sola RPC per N messaggi (api.MessagesForwardMessages id=[]int).
func (m MainModel) ForwardMessageCmd(target, source model.Chat, msgIDs []int) tea.Cmd {
	return func() tea.Msg {
		err := m.Bridge.ForwardMessage(context.Background(), target, source, msgIDs)
		return telegram.ForwardResultMsg{Target: target, Err: err}
	}
}

// DeleteMessagesCmd crea il comando batch per cancellare N messaggi.
// BATCH_ATOMICITY: una sola RPC (api.MessagesDeleteMessages id=[]int).
func (m MainModel) DeleteMessagesCmd(chat model.Chat, msgIDs []int) tea.Cmd {
	n := len(msgIDs)
	return func() tea.Msg {
		err := m.Bridge.DeleteMessages(context.Background(), chat, msgIDs)
		return deleteResultMsg{n: n, err: err}
	}
}

// LoadPinnedMessageCmd avvia il fetch del pinned message in background (Step 33, ADR-021 §A1).
// Emette PinnedMsgLoadedMsg; il consumer applica PINNED_STALE_DROP.
func (m MainModel) LoadPinnedMessageCmd(chat model.Chat) tea.Cmd {
	return loadPinnedMessageCmd(m.Bridge, chat)
}

// LogoutCmd: AuthLogOut + remove session.json, poi tea.Quit.
func (m MainModel) LogoutCmd() tea.Cmd {
	return tea.Sequence(
		func() tea.Msg { _ = m.Bridge.Logout(context.Background()); return nil },
		tea.Quit,
	)
}

// deleteResultMsg è il risultato interno della batch delete RPC.
type deleteResultMsg struct {
	n   int
	err error
}

// buildDeleteStatusMsg costruisce il testo del status bar post-delete.
func buildDeleteStatusMsg(n int, err error) string {
	if err != nil {
		return "Delete failed: " + err.Error()
	}
	if n == 1 {
		return "Message deleted"
	}
	return fmt.Sprintf("Deleted %d messages", n)
}
