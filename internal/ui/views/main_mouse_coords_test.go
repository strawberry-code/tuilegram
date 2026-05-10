package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"testing"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// main_mouse_coords_test.go — coordinate-to-index helpers + isWheelMsg tests.

// mousePressAt constructs a left-click Press MouseMsg at (x, y).
// Shared helper used by all mouse dispatch tests.
func mousePressAt(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
}

// chatStub builds a minimal model.Chat for test setup.
func chatStub(i int) model.Chat {
	return model.Chat{ID: model.ChatID{ID: int64(i + 1)}, Title: "Chat"}
}

// TestChatListRowIndex verifies coordinate-to-index mapping.
// Step 34: per-row borders rimossi → 1 row per item.
// Layout: row 0 = header; row 1 = item 0; row 2 = item 1; etc.
func TestChatListRowIndex(t *testing.T) {
	cl := NewChatListModel()
	cl.Height = 20
	for i := 0; i < 5; i++ {
		cl.chats = append(cl.chats, chatStub(i))
	}
	if idx := chatListRowIndex(cl, 0); idx != -1 {
		t.Errorf("header row must return -1, got %d", idx)
	}
	if idx := chatListRowIndex(cl, 1); idx != 0 {
		t.Errorf("row 1 must map to chat 0, got %d", idx)
	}
	if idx := chatListRowIndex(cl, 4); idx != 3 {
		t.Errorf("row 4 must map to chat 3, got %d", idx)
	}
	if idx := chatListRowIndex(cl, 100); idx != -1 {
		t.Errorf("out-of-range row must return -1, got %d", idx)
	}
}

// TestChatListRowIndexOffset verifies offset is respected.
func TestChatListRowIndexOffset(t *testing.T) {
	cl := NewChatListModel()
	cl.Height = 20
	for i := 0; i < 5; i++ {
		cl.chats = append(cl.chats, chatStub(i))
	}
	cl.offset = 2
	// row 1 → relIdx 0 → absIdx = 2+0 = 2
	if idx := chatListRowIndex(cl, 1); idx != 2 {
		t.Errorf("with offset=2, row 1 must map to chat 2, got %d", idx)
	}
}

// TestFolderRowIndex: header → -1; first folder row → 0; out of range → -1.
func TestFolderRowIndex(t *testing.T) {
	fm := NewFolderModel()
	fm.Height = 10
	if idx := folderRowIndex(fm, 0); idx != -1 {
		t.Errorf("header row must return -1, got %d", idx)
	}
	if idx := folderRowIndex(fm, 1); idx != 0 {
		t.Errorf("row 1 must map to folder 0 (sentinel), got %d", idx)
	}
	if idx := folderRowIndex(fm, 99); idx != -1 {
		t.Errorf("out-of-range row must return -1, got %d", idx)
	}
}

// TestScrollMouseFolder verifies wheel moves folder cursor.
func TestScrollMouseFolder(t *testing.T) {
	fm := NewFolderModel()
	fm.allFolders = append(fm.allFolders, model.ChatFolder{ID: 1, Title: "Work"})
	fm.cursor = 0

	down := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	fm.scrollMouse(down)
	if fm.cursor != 1 {
		t.Errorf("WheelDown must increment cursor to 1, got %d", fm.cursor)
	}

	up := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	fm.scrollMouse(up)
	if fm.cursor != 0 {
		t.Errorf("WheelUp must decrement cursor back to 0, got %d", fm.cursor)
	}
}

// TestIsWheelMsgWheelButtons: WheelUp and WheelDown are detected as wheel events.
func TestIsWheelMsgWheelButtons(t *testing.T) {
	up := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	if !isWheelMsg(up) {
		t.Error("WheelUp must be detected as wheel")
	}
	down := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	if !isWheelMsg(down) {
		t.Error("WheelDown must be detected as wheel")
	}
}
