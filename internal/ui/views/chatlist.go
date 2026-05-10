package views

import (
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// ConnStatus rappresenta lo stato della connessione Telegram.
type ConnStatus int

const (
	ConnConnected ConnStatus = iota
	ConnDisconnected
	ConnReconnecting
)

// ChatListModel gestisce il pannello sinistro con la lista delle chat.
type ChatListModel struct {
	chats          []model.Chat
	selected       int
	offset         int
	Width          int
	Height         int
	Conn           ConnStatus
	typingPeers    map[model.ChatID]struct{} // peer attualmente in stato Typing.Active
	folders        []model.ChatFolder        // Step 34: tab bar source (sentinel + server folders)
	activeFolderID int                       // Step 34: ID del tab attualmente selezionato
}

// NewChatListModel crea un nuovo modello per la chat list.
func NewChatListModel() ChatListModel {
	return ChatListModel{
		typingPeers: make(map[model.ChatID]struct{}),
	}
}

// SetFolders aggiorna i tab disponibili e il tab attivo (Step 34).
// Chiamato da main_folder_info_handlers su FoldersLoadedMsg / FolderSelectMsg.
func (m *ChatListModel) SetFolders(folders []model.ChatFolder, activeID int) {
	m.folders = folders
	m.activeFolderID = activeID
}

// SetTyping aggiorna il set di peer in stato Typing.Active nella chat list.
func (m *ChatListModel) SetTyping(peer model.ChatID, active bool) {
	if active {
		m.typingPeers[peer] = struct{}{}
	} else {
		delete(m.typingPeers, peer)
	}
}

// SetChats imposta la lista delle chat.
func (m *ChatListModel) SetChats(chats []model.Chat) {
	m.chats = chats
	m.selected = 0
	m.offset = 0
}

// Chats returns a snapshot of the current chat list (for forward picker).
func (m ChatListModel) Chats() []model.Chat {
	return m.chats
}

// UpdateChat aggiorna la chat con un nuovo messaggio e riordina.
func (m *ChatListModel) UpdateChat(chatID model.ChatID, isOpen bool) {
	for i := range m.chats {
		if m.chats[i].ID == chatID {
			m.chats[i].TopMessageDate = int(time.Now().Unix())
			if !isOpen {
				m.chats[i].UnreadCount++
			}
			break
		}
	}
	sort.SliceStable(m.chats, func(i, j int) bool {
		a, b := m.chats[i], m.chats[j]
		if a.IsPinned != b.IsPinned {
			return a.IsPinned
		}
		if a.HasUnread() != b.HasUnread() {
			return a.HasUnread()
		}
		return a.TopMessageDate > b.TopMessageDate
	})
}

func (m ChatListModel) Update(msg tea.Msg) (ChatListModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.handleKey(msg)
	case tea.MouseMsg:
		m.handleMouse(msg)
	}
	return m, nil
}
