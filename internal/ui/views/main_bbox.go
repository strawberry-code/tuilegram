package views

// main_bbox.go — Bbox struct + mainBboxes cache type + top-level recomputeBboxes.
// Step 32: central hit-test router (ADR-020 §D1, §D2).
// Invariant BBOXES_VALID_BEFORE_DISPATCH: recomputeBboxes called at end of
// SetSize and every layout-mutating handler (ADR-020 §D2). Idempotent.

// Bbox is an axis-aligned rectangle in terminal cell coordinates.
// Half-open intervals: x0 ≤ x < x1, y0 ≤ y < y1 (ADR-020 §D3 edge case).
type Bbox struct {
	X0, Y0, X1, Y1 int
}

// Contains reports whether cell (x, y) is inside the Bbox.
// Border columns belong to the left/top panel (half-open, deterministic).
func (b Bbox) Contains(x, y int) bool {
	return x >= b.X0 && x < b.X1 && y >= b.Y0 && y < b.Y1
}

// valid reports whether the Bbox has positive area.
func (b Bbox) valid() bool {
	return b.X1 > b.X0 && b.Y1 > b.Y0
}

// mainBboxes caches bounding boxes of all clickable widgets.
// Zero-value Bbox (valid()==false) means the widget is hidden.
// Compact: hidden panel slots stay zero (NO_HIDDEN_CLICK, ADR-020 §D9).
type mainBboxes struct {
	folderSidebar   Bbox // Wide only, when folderModel.IsVisible()
	chatList        Bbox // Wide + Compact ShowingChatList
	convViewport    Bbox // Wide + Compact ShowingConversation
	convHeader      Bbox // Wide + Compact ShowingConversation
	inputArea       Bbox // Wide + Compact ShowingConversation
	sendButton      Bbox // sub-rect inside inputArea
	statusBar       Bbox // always present
	overlayFullscr  Bbox // cmdPalette/help/search/whichKey/forward (full-screen)
	chatInfoOverlay Bbox // chatInfo right-anchored overlay
}

// recomputeBboxes recalculates bboxes from current layout state.
// Called from SetSize and every handler that mutates layout (ADR-020 §D2).
func (m *MainModel) recomputeBboxes() {
	m.bboxes = mainBboxes{} // reset all slots to zero (invalid)
	if m.Width == 0 || m.Height == 0 {
		return
	}
	if m.layoutMode == LayoutCompact {
		m.recomputeCompactBboxes()
	} else {
		m.recomputeWideBboxes()
	}
	m.recomputeOverlayBboxes()
}

// recomputeOverlayBboxes sets full-screen and chatInfo overlay bboxes.
// Full-screen overlays (cmdPalette, help, search, whichKey, forwardPicker)
// occupy the entire canvas; chatInfo is right-anchored (34 content + 2 border).
func (m *MainModel) recomputeOverlayBboxes() {
	w, h := m.Width, m.Height
	m.bboxes.overlayFullscr = Bbox{0, 0, w, h}
	const chatInfoTotalW = 36 // 34 content + 2 border cols
	ciX0 := w - chatInfoTotalW
	if ciX0 < 0 {
		ciX0 = 0
	}
	m.bboxes.chatInfoOverlay = Bbox{ciX0, 0, w, h}
}
