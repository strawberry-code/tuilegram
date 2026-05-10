package views

import (
	"testing"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// main_bbox_test.go — Step 32 bbox struct + Wide layout tests.
// Companion: main_bbox_compact_test.go (Compact + overlay tests).

func newTestMainModel(w, h int, mode LayoutMode) MainModel {
	return MainModel{
		chatList:     NewChatListModel(),
		conversation: NewConversationModel(),
		folderModel:  NewFolderModel(),
		layoutMode:   mode,
		Width:        w,
		Height:       h,
		typing:       make(map[model.ChatID]TypingState),
	}
}

func TestBboxContains(t *testing.T) {
	b := Bbox{10, 5, 20, 15}
	if !b.Contains(10, 5) {
		t.Error("lower-left corner must be inside")
	}
	if b.Contains(20, 15) {
		t.Error("upper-right corner must be exclusive")
	}
	if !b.Contains(19, 14) {
		t.Error("one inside upper-right must be inside")
	}
	if b.Contains(9, 5) {
		t.Error("x=9 must be outside x0=10")
	}
}

func TestBboxValidEmpty(t *testing.T) {
	var b Bbox
	if b.valid() {
		t.Error("zero-value Bbox must be invalid")
	}
	if b.Contains(0, 0) {
		t.Error("zero-value Bbox must not contain any point")
	}
}

func TestWideLayoutBboxes(t *testing.T) {
	m := newTestMainModel(140, 40, LayoutWide)
	m.recomputeBboxes()
	if !m.bboxes.chatList.valid() {
		t.Error("Wide: chatList bbox must be valid")
	}
	if !m.bboxes.statusBar.valid() {
		t.Error("Wide: statusBar bbox must be valid")
	}
	if m.bboxes.folderSidebar.valid() {
		t.Error("Wide: folderSidebar must be invalid when not visible")
	}
}

func TestWideStatusBarRow(t *testing.T) {
	m := newTestMainModel(140, 40, LayoutWide)
	m.recomputeBboxes()
	sb := m.bboxes.statusBar
	if sb.Y0 != 39 || sb.Y1 != 40 {
		t.Errorf("statusBar must be row [39,40), got [%d,%d)", sb.Y0, sb.Y1)
	}
	if sb.X0 != 0 || sb.X1 != 140 {
		t.Errorf("statusBar must span full width, got [%d,%d)", sb.X0, sb.X1)
	}
}

func TestBboxRecomputeAfterResize(t *testing.T) {
	m := newTestMainModel(140, 40, LayoutWide)
	m.recomputeBboxes()
	w1 := m.bboxes.chatList.X1
	m.Width = 200
	m.recomputeBboxes()
	w2 := m.bboxes.chatList.X1
	if w2 <= w1 {
		t.Errorf("chatList X1 must grow with width: was %d, now %d", w1, w2)
	}
}

// TestWideChatListConversationAdjacent: BBOX_TOTAL base case — no gap between panels.
func TestWideChatListConversationAdjacent(t *testing.T) {
	m := newTestMainModel(140, 40, LayoutWide)
	m.conversation.active = true
	m.recomputeBboxes()
	cl := m.bboxes.chatList
	cv := m.bboxes.convViewport
	if !cl.valid() || !cv.valid() {
		t.Skip("convViewport not populated (needs sufficient height)")
	}
	if cl.X1 != cv.X0 {
		t.Errorf("chatList.X1=%d must equal convViewport.X0=%d", cl.X1, cv.X0)
	}
}

func TestOverlayFullscreenBbox(t *testing.T) {
	m := newTestMainModel(120, 35, LayoutWide)
	m.recomputeBboxes()
	ob := m.bboxes.overlayFullscr
	if ob.X0 != 0 || ob.Y0 != 0 || ob.X1 != 120 || ob.Y1 != 35 {
		t.Errorf("overlayFullscr must be {0,0,120,35}, got %+v", ob)
	}
}
