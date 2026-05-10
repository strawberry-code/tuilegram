package views

// main_mouse_coords.go — coordinate-to-index helpers for the mouse router.
// Step 32 (ADR-020 §D4): maps absolute terminal Y coordinates to logical
// widget row indices. O(1) — pure arithmetic, no iteration.

import tea "github.com/charmbracelet/bubbletea"

// isWheelMsg reports whether the MouseMsg is a vertical wheel event.
// IsWheel() is defined on MouseEvent, not on MouseMsg (type alias without methods).
// Only WheelUp/WheelDown are handled; lateral wheel is ignored (ADR-020 scope).
func isWheelMsg(msg tea.MouseMsg) bool {
	return msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown
}

// chatListItemHeight returns the rendered row count for chat at idx.
// Step 34 (ADR-022): per-row borders rimossi, typing inline → sempre 1 riga.
func chatListItemHeight(cl ChatListModel, idx int) int {
	if idx < 0 || idx >= len(cl.chats) {
		return 0
	}
	return 1
}

// chatListRowIndex converts an absolute terminal Y coordinate to the logical
// chat index (0-based into chatList.chats, accounting for offset and per-item
// height variation when peers are typing).
//
// ChatList layout (chatlist_view.go):
//
//	row 0            — header ("● CHATS")
//	row 1+           — items at variable height: 3 rows base, 4 if typing
//
// O(visible) — accumulates per-item height starting from offset.
// Returns -1 when the click is on the header or out of range.
func chatListRowIndex(cl ChatListModel, absY int) int {
	if cl.Height == 0 || len(cl.chats) == 0 {
		return -1
	}
	relY := absY - 1 // row 0 = header
	if relY < 0 {
		return -1
	}
	row := 0
	for i := cl.offset; i < len(cl.chats); i++ {
		h := chatListItemHeight(cl, i)
		if relY < row+h {
			return i
		}
		row += h
	}
	return -1
}

// folderRowIndex converts an absolute terminal Y coordinate to the folder
// cursor index (0-based into folderModel.allFolders).
//
// Folder sidebar layout (folder_view.go):
//
//	row 0      — header ("FOLDERS")
//	row 1+N    — one row per folder (simple text row, no border)
//
// Returns -1 when the click is on the header or out of range.
func folderRowIndex(fm FolderModel, absY int) int {
	if fm.Height == 0 || len(fm.allFolders) == 0 {
		return -1
	}
	// Folder sidebar starts at y=0 (leftmost panel in Wide layout).
	relY := absY - 1 // row 0 = header
	if relY < 0 {
		return -1 // clicked on header
	}
	if relY >= len(fm.allFolders) {
		return -1
	}
	return relY
}

// scrollMouse forwards a wheel MouseMsg to the folder sidebar scroll logic.
// Mirrors handleMouse in chatlist_nav.go: wheel up/down moves the cursor.
// Sidebar has no separate scroll offset; cursor IS the scroll position.
func (m *FolderModel) scrollMouse(msg tea.MouseMsg) {
	switch msg.Button {
	case tea.MouseButtonWheelDown:
		m.cursor++
		m.clampCursor()
	case tea.MouseButtonWheelUp:
		m.cursor--
		m.clampCursor()
	}
}
