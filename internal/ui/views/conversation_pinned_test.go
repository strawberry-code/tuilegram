package views

import (
	"testing"
	"time"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// TestPinnedBar_NilMsg verifica che pinnedBar() ritorni stringa vuota senza pin.
func TestPinnedBar_NilMsg(t *testing.T) {
	m := NewConversationModel()
	m.Width = 80
	if got := m.pinnedBar(); got != "" {
		t.Errorf("pinnedBar() should be empty when pinnedMsg == nil, got %q", got)
	}
}

// TestPinnedBar_WithMsg verifica che pinnedBar() contenga il testo del pin.
func TestPinnedBar_WithMsg(t *testing.T) {
	m := NewConversationModel()
	m.Width = 80
	msg := model.Message{ID: 1, Text: "Important announcement", Date: time.Now()}
	m.pinnedMsg = &msg
	bar := m.pinnedBar()
	if bar == "" {
		t.Fatal("pinnedBar() should not be empty when pinnedMsg != nil")
	}
}

// TestPinnedBar_Truncation verifica che testi lunghi vengano troncati.
func TestPinnedBar_Truncation(t *testing.T) {
	m := NewConversationModel()
	m.Width = 20 // molto stretto
	long := "This is a very long pinned message that exceeds the width"
	msg := model.Message{ID: 1, Text: long, Date: time.Now()}
	m.pinnedMsg = &msg
	bar := m.pinnedBar()
	if bar == "" {
		t.Fatal("pinnedBar() should not be empty")
	}
}

// TestSetSize_PinnedBarHeight verifica PINNED_OFFSET_RESERVED.
// Con pinnedMsg != nil il viewport deve essere 2 righe più basso.
func TestSetSize_PinnedBarHeight(t *testing.T) {
	m := NewConversationModel()
	m.SetSize(80, 40)
	heightWithout := m.viewport.Height

	msg := model.Message{ID: 1, Text: "pin", Date: time.Now()}
	m.pinnedMsg = &msg
	m.SetSize(80, 40)
	heightWith := m.viewport.Height

	diff := heightWithout - heightWith
	if diff != pinnedBarHeight {
		t.Errorf("viewport height diff = %d, want %d (pinnedBarHeight)", diff, pinnedBarHeight)
	}
}

// TestPinnedStale_Drop verifica PINNED_STALE_DROP: msg con chatID diverso è droppato.
func TestPinnedStale_Drop(t *testing.T) {
	m := NewConversationModel()
	m.chat = model.Chat{ID: model.ChatID{PeerType: model.PeerUser, ID: 100}}
	m.Width = 80
	m.Height = 40

	staleMsg := model.Message{ID: 1, Text: "stale pin", Date: time.Now()}
	// ChatID diverso dalla chat attiva.
	loaded := telegram.PinnedMsgLoadedMsg{
		ChatID: model.ChatID{PeerType: model.PeerUser, ID: 999},
		Msg:    &staleMsg,
	}
	updated, _ := m.handlePinnedMsgLoaded(loaded)
	if updated.pinnedMsg != nil {
		t.Error("PINNED_STALE_DROP violated: stale msg should be dropped")
	}
}

// TestPinnedSingle_PerChat verifica che OpenChat azzeri pinnedMsg (PINNED_SINGLE_PER_CHAT).
func TestPinnedSingle_PerChat(t *testing.T) {
	m := NewConversationModel()
	msg := model.Message{ID: 1, Text: "old pin", Date: time.Now()}
	m.pinnedMsg = &msg
	chat := model.Chat{ID: model.ChatID{PeerType: model.PeerUser, ID: 200}}
	m.OpenChat(chat)
	if m.pinnedMsg != nil {
		t.Error("PINNED_SINGLE_PER_CHAT violated: OpenChat should reset pinnedMsg to nil")
	}
}
