package views

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// visibleItems restituisce quanti items sono visibili nel viewport.
// Step 34 (ADR-022): per-row borders rimossi → ogni item = 1 riga base.
// header=1 riga. Con peer typing item cresce a 2 righe ma questa stima
// usa il caso base per ensureVisible/ctrl+d/u — precisione esatta non
// richiesta. Mouse mapping usa accumulo per-item in chatListRowIndex.
func (m ChatListModel) visibleItems() int {
	available := m.Height - 1
	if available <= 0 {
		return 0
	}
	return available
}

func (m *ChatListModel) ensureVisible() {
	visible := m.visibleItems()
	if visible == 0 {
		return
	}
	if m.selected < m.offset {
		m.offset = m.selected
	}
	if m.selected >= m.offset+visible {
		m.offset = m.selected - visible + 1
	}
}

// JumpToUnread moves selection to the first chat with unread messages.
// No-op if no unread chats exist (Step 28: g→u chord + jumpUnreadMsg).
func (m *ChatListModel) JumpToUnread() {
	for i, c := range m.chats {
		if c.HasUnread() {
			m.selected = i
			m.ensureVisible()
			return
		}
	}
}

// ChatByID finds a chat by its ChatID. Used by search jump-to-message.
// Returns false when the chat is not present in the loaded list.
func (m ChatListModel) ChatByID(id model.ChatID) (model.Chat, bool) {
	for _, c := range m.chats {
		if c.ID == id {
			return c, true
		}
	}
	return model.Chat{}, false
}

// SelectedChat restituisce la chat selezionata e true se disponibile.
func (m ChatListModel) SelectedChat() (model.Chat, bool) {
	if len(m.chats) == 0 {
		return model.Chat{}, false
	}
	return m.chats[m.selected], true
}

func (m *ChatListModel) handleMouse(msg tea.MouseMsg) {
	if msg.Button == tea.MouseButtonWheelDown {
		if m.selected < len(m.chats)-1 {
			m.selected++
			m.ensureVisible()
		}
	} else if msg.Button == tea.MouseButtonWheelUp {
		if m.selected > 0 {
			m.selected--
			m.ensureVisible()
		}
	}
}

func (m *ChatListModel) handleKey(msg tea.KeyMsg) {
	switch msg.String() {
	case "j", "down":
		if m.selected < len(m.chats)-1 {
			m.selected++
			m.ensureVisible()
		}
	case "k", "up":
		if m.selected > 0 {
			m.selected--
			m.ensureVisible()
		}
	case "g":
		m.selected = 0
		m.offset = 0
	case "G":
		m.selected = max(0, len(m.chats)-1)
		m.ensureVisible()
	case "ctrl+d":
		half := m.visibleItems() / 2
		m.selected = min(m.selected+half, len(m.chats)-1)
		m.ensureVisible()
	case "ctrl+u":
		half := m.visibleItems() / 2
		m.selected = max(m.selected-half, 0)
		m.ensureVisible()
	}
}
