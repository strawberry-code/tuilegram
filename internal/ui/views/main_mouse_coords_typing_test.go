package views

import (
	"testing"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// main_mouse_coords_typing_test.go — height tests (sempre 1 riga post-Step 34).
// Step 34: typing è inline ✎, mai second row → height costante.

func TestChatListRowIndexTyping(t *testing.T) {
	cl := NewChatListModel()
	cl.Height = 20
	cl.typingPeers = map[model.ChatID]struct{}{}
	for i := 0; i < 5; i++ {
		cl.chats = append(cl.chats, chatStub(i))
	}
	cl.typingPeers[cl.chats[0].ID] = struct{}{}
	// Typing è inline → no shift. Layout 1 row per item invariato.
	if idx := chatListRowIndex(cl, 1); idx != 0 {
		t.Errorf("row 1 must map to chat 0 even with typing, got %d", idx)
	}
	if idx := chatListRowIndex(cl, 2); idx != 1 {
		t.Errorf("row 2 must map to chat 1, got %d", idx)
	}
	if idx := chatListRowIndex(cl, 4); idx != 3 {
		t.Errorf("row 4 must map to chat 3, got %d", idx)
	}
}

func TestChatListItemHeight(t *testing.T) {
	cl := NewChatListModel()
	cl.typingPeers = map[model.ChatID]struct{}{}
	cl.chats = append(cl.chats, chatStub(0), chatStub(1))
	cl.typingPeers[cl.chats[1].ID] = struct{}{}

	if h := chatListItemHeight(cl, 0); h != 1 {
		t.Errorf("non-typing item: height must be 1, got %d", h)
	}
	if h := chatListItemHeight(cl, 1); h != 1 {
		t.Errorf("typing item: height must still be 1 (inline mark), got %d", h)
	}
	if h := chatListItemHeight(cl, 99); h != 0 {
		t.Errorf("out-of-range: height must be 0, got %d", h)
	}
}
