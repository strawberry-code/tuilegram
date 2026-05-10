package views

import (
	"strings"
	"testing"
)

// TestKeymapHint_Deterministic verifica STATUSBAR_KEYMAP_DETERMINISTIC:
// stessi argomenti → stessa stringa, chiamate multiple.
func TestKeymapHint_Deterministic(t *testing.T) {
	m := NewMainModel(100)
	m.Width = 120
	m.Height = 40
	first := m.keymapHint()
	second := m.keymapHint()
	if first != second {
		t.Errorf("keymapHint() not deterministic: %q vs %q", first, second)
	}
}

// TestKeymapHint_ChatListFocus verifica hint per FocusChatList.
func TestKeymapHint_ChatListFocus(t *testing.T) {
	m := NewMainModel(100)
	m.focus = FocusChatList
	hint := m.keymapHint()
	if !strings.Contains(hint, "j/k") {
		t.Errorf("FocusChatList hint should contain j/k, got %q", hint)
	}
}

// TestKeymapHint_ConversationFocus verifica hint per FocusConversation senza input.
func TestKeymapHint_ConversationFocus(t *testing.T) {
	m := NewMainModel(100)
	m.focus = FocusConversation
	hint := m.keymapHint()
	if !strings.Contains(hint, "gx") {
		t.Errorf("FocusConversation hint should mention gx, got %q", hint)
	}
}

// TestKeymapHint_InputFocus verifica hint per input mode.
func TestKeymapHint_InputFocus(t *testing.T) {
	m := NewMainModel(100)
	m.focus = FocusConversation
	m.conversation.inputFocus = true
	hint := m.keymapHint()
	if !strings.Contains(hint, "Enter send") {
		t.Errorf("inputFocus hint should mention Enter send, got %q", hint)
	}
}

// TestKeymapHint_FoldersFocus verifica hint per FocusFolders.
func TestKeymapHint_FoldersFocus(t *testing.T) {
	m := NewMainModel(100)
	m.focus = FocusFolders
	hint := m.keymapHint()
	if !strings.Contains(hint, "F close") {
		t.Errorf("FocusFolders hint should contain 'F close', got %q", hint)
	}
}

// TestStatusBar_RightSlotPresent verifica che il right slot contenga statusMsg.
// Su terminale normale (Width=120) la stringa è sempre presente senza troncamento.
func TestStatusBar_RightSlotPresent(t *testing.T) {
	m := NewMainModel(100)
	m.Width = 120
	m.Height = 40
	m.statusMsg = "error occurred"
	bar := m.renderStatusBar()
	// Il right slot deve essere presente nella stringa.
	if !strings.Contains(bar, "error occurred") {
		t.Errorf("status bar should contain statusMsg, got %q", bar)
	}
}
