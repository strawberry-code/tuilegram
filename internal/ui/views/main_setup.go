package views

import (
	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// main_setup.go — bridge wiring, chat population, folder filtering, overlay guard.
// Separated from main.go to respect the 120-LOC limit.

// SetBridge assigns the Telegram bridge to MainModel and sub-models that use it.
func (m *MainModel) SetBridge(b *telegram.Bridge) {
	m.Bridge = b
	m.search.Bridge = b
}

// SetChats stores the unfiltered dialog list and applies the current folder filter.
// Called on DialogsLoadedMsg and whenever allChats must be refreshed.
func (m *MainModel) SetChats(chats []model.Chat) {
	m.allChats = chats
	m.applyFolderFilter()
}

// applyFolderFilter recomputes chatList from allChats using the active folder.
// Always synchronous — O(n) scan, no RPC (invariant FILTER_SYNC, ADR-016 §C).
// activeChatID is intentionally NOT touched (ADR-016 §D4, ACTIVE_CHAT_INVARIANT).
func (m *MainModel) applyFolderFilter() {
	selID := m.folderModel.SelectedID()
	// Step 34: sync chatList tab bar (folders + active id).
	m.chatList.SetFolders(m.folderModel.allFolders, selID)
	if selID == 0 {
		// "All Chats" sentinel: no filter.
		m.chatList.SetChats(m.allChats)
		return
	}
	var folder model.ChatFolder
	for _, f := range m.folderModel.allFolders {
		if f.ID == selID {
			folder = f
			break
		}
	}
	filtered := make([]model.Chat, 0, len(m.allChats))
	for _, c := range m.allChats {
		if folder.ContainsChat(c.ID) {
			filtered = append(filtered, c)
		}
	}
	m.chatList.SetChats(filtered)
}

// anyOverlayActive reports whether any modal overlay is currently shown.
// The folder sidebar is NOT counted — it is a panel, not an overlay (ADR-016 §D2).
func (m MainModel) anyOverlayActive() bool {
	return m.search.Active ||
		m.cmdPalette.Active ||
		m.help.Active ||
		m.whichKey.IsVisible() ||
		m.conversation.forwardPicker.Active() ||
		m.chatInfo.Active
}
