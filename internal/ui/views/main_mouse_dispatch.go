package views

// main_mouse_dispatch.go — central mouse event router for MainModel.
// Step 32 (ADR-020 §D1): handleMouseMsg is the single entry point called
// from main_update.go. Implements the dispatch tree from mouse-routing.md §B.
//
// Invariants enforced here:
//   NO_PHANTOM_DRAG  — Motion/Release events are dropped immediately.
//   OVERLAY_FIRST    — overlay hit-test precedes base-layer hit-test.
//   WHEEL_BY_POSITION — wheel routes by cursor position, not by activePanel.
//   NO_HIDDEN_CLICK  — Compact hidden-panel bboxes are zero → no match.
//   KEYBOARD_PARITY  — every click handler calls the same helper as keyboard.

import tea "github.com/charmbracelet/bubbletea"

// handleMouseMsg is the central router. Called from MainModel.Update.
// Only Press + wheel events produce state mutations (ADR-020 §D7).
func (m MainModel) handleMouseMsg(msg tea.MouseMsg) (MainModel, tea.Cmd) {
	// NO_PHANTOM_DRAG: discard Motion and Release immediately.
	if msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionRelease {
		return m, nil
	}

	// Wheel events: route by cursor position (D3 WHEEL_BY_POSITION).
	// IsWheel() is defined on MouseEvent, not MouseMsg; check Button directly.
	if isWheelMsg(msg) {
		return m.dispatchWheel(msg)
	}

	// Left-click only; discard Right, Middle, and other buttons (D-scope).
	if msg.Button != tea.MouseButtonLeft {
		return m, nil
	}

	return m.dispatchClick(msg)
}

// dispatchWheel routes a wheel event to the widget under the cursor.
// Consults bboxes in order: convViewport → chatList → folderSidebar.
// Cursor outside all wheel-aware widgets: no-op (NoOpWheel, ADR-020 §D3).
func (m MainModel) dispatchWheel(msg tea.MouseMsg) (MainModel, tea.Cmd) {
	x, y := msg.X, msg.Y
	if m.bboxes.convViewport.Contains(x, y) {
		// Forward raw MouseMsg to bubbles/viewport via conversation.Update.
		var cmd tea.Cmd
		m.conversation, cmd = m.conversation.Update(msg)
		return m, cmd
	}
	if m.bboxes.chatList.Contains(x, y) {
		m.chatList.handleMouse(msg)
		return m, nil
	}
	if m.bboxes.folderSidebar.Contains(x, y) {
		m.folderModel.scrollMouse(msg)
		return m, nil
	}
	return m, nil // NoOpWheel: cursor in gap/status bar/hidden
}

// dispatchClick routes a left-click press event.
// Priority: overlay-first (OVERLAY_FIRST) → base-layer widget hit-test.
func (m MainModel) dispatchClick(msg tea.MouseMsg) (MainModel, tea.Cmd) {
	// OVERLAY_FIRST: if any overlay is active, resolve within overlay logic.
	if m.anyOverlayActive() {
		return m.dispatchClickOverlay(msg)
	}
	return m.dispatchClickBase(msg)
}

// dispatchClickBase performs hit-test against the base-layer widget bboxes.
// z-order: sendButton ⊂ inputArea (innermost first), then panels.
func (m MainModel) dispatchClickBase(msg tea.MouseMsg) (MainModel, tea.Cmd) {
	x, y := msg.X, msg.Y

	// sendButton must be tested before inputArea (sub-rect, innermost first).
	if m.bboxes.sendButton.Contains(x, y) {
		return m.handleClickSend()
	}
	if m.bboxes.inputArea.Contains(x, y) {
		return m.handleClickTextarea()
	}
	if m.bboxes.convViewport.Contains(x, y) || m.bboxes.convHeader.Contains(x, y) {
		return m.handleClickConversation()
	}
	if m.bboxes.chatList.Contains(x, y) {
		return m.handleClickChatList(msg.Y)
	}
	if m.bboxes.folderSidebar.Contains(x, y) {
		return m.handleClickFolder(msg.Y)
	}
	return m, nil // NoOpClick: gap, status bar, or hidden panel
}
