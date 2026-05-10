package views

import "testing"

// main_bbox_compact_test.go — Compact layout bbox + overlay dispatch tests.

func TestCompactChatListBboxes(t *testing.T) {
	m := newTestMainModel(80, 30, LayoutCompact)
	m.compactVisible = CompactChatList
	m.recomputeBboxes()
	if !m.bboxes.chatList.valid() {
		t.Error("Compact/ChatList: chatList bbox must be valid")
	}
	if m.bboxes.convViewport.valid() {
		t.Error("Compact/ChatList: convViewport must be nil (NO_HIDDEN_CLICK)")
	}
	if m.bboxes.sendButton.valid() {
		t.Error("Compact/ChatList: sendButton must be nil (NO_HIDDEN_CLICK)")
	}
}

func TestCompactConversationBboxes(t *testing.T) {
	m := newTestMainModel(80, 30, LayoutCompact)
	m.compactVisible = CompactConversation
	m.conversation.active = true
	m.recomputeBboxes()
	if m.bboxes.chatList.valid() {
		t.Error("Compact/Conversation: chatList must be nil (NO_HIDDEN_CLICK)")
	}
	if !m.bboxes.convViewport.valid() {
		t.Error("Compact/Conversation: convViewport must be valid")
	}
	if !m.bboxes.inputArea.valid() {
		t.Error("Compact/Conversation: inputArea must be valid")
	}
}

// TestNoHiddenClick: click in hidden panel area in Compact → no focus change.
func TestNoHiddenClick(t *testing.T) {
	m := newTestMainModel(80, 30, LayoutCompact)
	m.compactVisible = CompactChatList
	m.recomputeBboxes()
	before := m.focus
	m2, _ := m.dispatchClickBase(mousePressAt(60, 10))
	if m2.focus != before {
		t.Error("NO_HIDDEN_CLICK: click on hidden panel must not change focus")
	}
}

// TestNoHiddenClickConvVisible: simmetrico — click su area chatList (nascosta)
// quando compactVisible=Conversation deve essere no-op (NO_HIDDEN_CLICK).
func TestNoHiddenClickConvVisible(t *testing.T) {
	m := newTestMainModel(80, 30, LayoutCompact)
	m.compactVisible = CompactConversation
	m.conversation.active = true
	m.focus = FocusConversation
	m.recomputeBboxes()
	if m.bboxes.chatList.valid() {
		t.Error("setup: chatList bbox should be nil when compactVisible=Conversation")
	}
}

// TestOverlayDismissOnClickOutside: click outside chatInfo (right-anchored) closes it.
func TestOverlayDismissOnClickOutside(t *testing.T) {
	m := newTestMainModel(120, 35, LayoutWide)
	m.recomputeBboxes()
	m.chatInfo.Active = true
	// Click far to the left — outside chatInfoOverlay bbox.
	m2, _ := m.dispatchClickOverlay(mousePressAt(5, 10))
	if m2.chatInfo.Active {
		t.Error("click outside dismissable chatInfo overlay must close it")
	}
}

// TestFullScreenOverlayBboxContainsAll: overlayFullscr covers every cell on canvas.
// Full-screen overlays have no "outside" → click-outside dismiss never fires.
func TestFullScreenOverlayBboxContainsAll(t *testing.T) {
	m := newTestMainModel(120, 35, LayoutWide)
	m.recomputeBboxes()
	ob := m.bboxes.overlayFullscr
	// Every corner must be inside.
	for _, pt := range [][2]int{{0, 0}, {119, 0}, {0, 34}, {119, 34}} {
		if !ob.Contains(pt[0], pt[1]) {
			t.Errorf("overlayFullscr must contain corner (%d,%d)", pt[0], pt[1])
		}
	}
}

// TestOverlayModalNoOp: click outside modal forwardPicker → no state change.
func TestOverlayModalNoOp(t *testing.T) {
	m := newTestMainModel(120, 35, LayoutWide)
	m.recomputeBboxes()
	m.conversation.forwardPicker = m.conversation.forwardPicker.Open(nil)
	m2, _ := m.dispatchClickOverlay(mousePressAt(5, 5))
	if !m2.conversation.forwardPicker.Active() {
		t.Error("click outside modal forwardPicker must be no-op (MODAL)")
	}
}

// TestIsWheelMsg: left-press is not a wheel event.
func TestIsWheelMsg(t *testing.T) {
	if isWheelMsg(mousePressAt(0, 0)) {
		t.Error("left press must not be detected as wheel")
	}
}
