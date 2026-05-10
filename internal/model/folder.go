package model

// ChatFolder represents a Telegram DialogFilter (server-side folder).
// IncludedChats lists the explicit chat IDs belonging to this folder.
// FolderID = 0 is reserved for the "All Chats" sentinel (UI-only).
type ChatFolder struct {
	ID            int
	Title         string
	IncludedChats []ChatID
}

// AllChatsFolder is the sentinel folder always shown first in the sidebar.
// Selecting it removes any filter (shows all dialogs).
var AllChatsFolder = ChatFolder{ID: 0, Title: "All Chats"}

// IsAllChats reports whether this folder is the "All Chats" sentinel.
func (f ChatFolder) IsAllChats() bool { return f.ID == 0 }

// ContainsChat reports whether chatID is included in this folder.
// For the sentinel (ID=0), every chat is considered included.
func (f ChatFolder) ContainsChat(id ChatID) bool {
	if f.IsAllChats() {
		return true
	}
	for _, cid := range f.IncludedChats {
		if cid == id {
			return true
		}
	}
	return false
}
