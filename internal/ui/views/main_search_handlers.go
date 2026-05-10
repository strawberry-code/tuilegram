package views

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// handleJumpToMessage closes the search overlay, opens the target chat, and
// loads messages centred on the hit's messageID (LoadMessagesAround).
// ChatByID lookup failure is a silent no-op (chat not yet loaded in list).
func (m MainModel) handleJumpToMessage(msg JumpToMessageMsg) (MainModel, tea.Cmd) {
	m.search = m.search.Close()
	chat, ok := m.chatList.ChatByID(msg.ChatID)
	if !ok {
		return m, nil
	}
	spinnerCmd := m.conversation.OpenChat(chat)
	m.focus = FocusConversation
	aroundCmd := func() tea.Msg {
		msgs, err := m.Bridge.LoadMessagesAround(context.Background(), chat, msg.MessageID)
		if err != nil {
			return telegram.MessagesErrMsg{Err: err}
		}
		return telegram.MessagesLoadedMsg{Messages: msgs}
	}
	return m, tea.Batch(spinnerCmd, aroundCmd)
}

// handleKeyMsg routes keyboard input at the MainModel level.
// Priority order (highest first):
//  1. Command palette active → forward ALL keys to it (modal, no bleed-through).
//  2. Help overlay active    → forward ALL keys to it (modal, no bleed-through).
//  3. Search overlay active  → forward ALL keys to it (modal, no bleed-through).
//  4. Chat info overlay      → forward ALL keys to it; F is UX-consumed (ADR-017 §D5).
//  5. Which-key pending/visible → handle chord resolution or cancel.
//  6. Global triggers (F, i, Ctrl+P, ?, /, prefix keys g/z) — no active overlay.
//  7. Folder sidebar focus → delegate to folderModel.
//  8. Conversation focus → delegate to conversation.
//  9. Chat list focus → enter opens chat; Shift+Tab → sidebar if visible.
func (m MainModel) handleKeyMsg(msg tea.KeyMsg) (MainModel, tea.Cmd) {
	if m.cmdPalette.Active {
		var cmd tea.Cmd
		m.cmdPalette, cmd = m.cmdPalette.Update(msg)
		return m, cmd
	}
	if m.help.Active {
		var cmd tea.Cmd
		m.help, cmd = m.help.Update(msg)
		return m, cmd
	}
	if m.search.Active {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}
	// Chat info overlay consumes all keys; F is a silent no-op (ADR-017 §D5).
	if m.chatInfo.Active {
		var cmd tea.Cmd
		m.chatInfo, cmd = m.chatInfo.Update(msg)
		return m, cmd
	}
	if m.whichKey.IsPending() || m.whichKey.IsVisible() {
		return m.handleWhichKeyInput(msg)
	}
	return m.handleGlobalKey(msg)
}
