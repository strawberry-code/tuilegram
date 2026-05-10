package views

// main_bbox_compute.go — Wide/Compact bbox computation helpers.
// Step 32: mirrors setWideSize/applyCompactSizes arithmetic exactly so bboxes
// are always consistent with the rendered layout (ADR-020 §D2).
// Split from main_bbox.go to respect the 120-LOC limit.

// recomputeWideBboxes populates bboxes for the Wide multi-panel layout.
// Arithmetic mirrors setWideSize exactly (single source of truth principle).
func (m *MainModel) recomputeWideBboxes() {
	w, h := m.Width, m.Height
	available := h - 1 // 1 row reserved for status bar

	fldW := 0
	if m.folderModel.IsVisible() {
		fldW = folderSidebarWidth
		m.bboxes.folderSidebar = Bbox{0, 0, fldW, available}
	}

	rest := w - fldW
	clW := int(float64(rest) * chatListRatio)
	convX := fldW + clW
	convW := rest - clW

	m.bboxes.chatList = Bbox{fldW, 0, fldW + clW, available}
	m.bboxes.statusBar = Bbox{0, available, w, h}

	if m.conversation.active {
		m.fillConvBboxes(convX, convW, available)
	}
}

// recomputeCompactBboxes populates bboxes for Compact single-panel layout.
// The hidden panel's bbox stays zero (NO_HIDDEN_CLICK invariant, ADR-020 §D9).
func (m *MainModel) recomputeCompactBboxes() {
	w, h := m.Width, m.Height
	available := h - 1
	m.bboxes.statusBar = Bbox{0, available, w, h}

	switch m.compactVisible {
	case CompactChatList:
		m.bboxes.chatList = Bbox{0, 0, w, available}
	case CompactConversation:
		if m.conversation.active {
			m.fillConvBboxes(0, w, available)
		}
	}
}

// fillConvBboxes sets header, viewport, inputArea, and sendButton bboxes
// for the conversation panel at absolute column convX with width convW.
// available is the panel height (status bar row already excluded).
//
// Layout (row-relative to absolute y=0):
//
//	[0..1]           convHeader (2 rows: outer border top + header content)
//	[2..inputTop-1]  convViewport (shrinks by searchBarHeight when bar active)
//	[inputTop..av-1] inputArea (inputAreaHeight=2 rows: border-top + text)
//
// sendButton occupies the rightmost 6 cols of the text row (inputTop+2).
func (m *MainModel) fillConvBboxes(convX, convW, available int) {
	// Header: top 2 rows of the conversation block.
	m.bboxes.convHeader = Bbox{convX, 0, convX + convW, 2}

	// InputArea: last inputAreaHeight rows of the panel.
	inputTop := available - inputAreaHeight
	m.bboxes.inputArea = Bbox{convX, inputTop, convX + convW, available}

	// Viewport: rows between header and inputArea (minus search bar if active).
	vpBot := inputTop
	if m.conversation.searchBar.Active {
		vpBot -= searchBarHeight
	}
	if vpBot > 2 {
		m.bboxes.convViewport = Bbox{convX, 2, convX + convW, vpBot}
	}

	// SendButton: rightmost 6 cols on text row inside inputArea (inputAreaHeight=2):
	//   inputTop+0 = border-top
	//   inputTop+1 = textarea + spacer + SEND   ← sendButton row
	btnRow := inputTop + 1
	m.bboxes.sendButton = Bbox{convX + convW - 6, btnRow, convX + convW, btnRow + 1}
}
