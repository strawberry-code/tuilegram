package views

// main_mouse_overlay.go — overlay-specific mouse click dispatch.
// Step 32 (ADR-020 §D5): click outside dismissable overlay → close;
// click outside modal overlay → no-op; click inside → forward to sub-model.
//
// Tassonomia (ADR-020 §D5):
//   Dismissable: cmdPalette, help, search, whichKey, chatInfo
//   Modal (no-op outside): forwardPicker
// Nota: confirmDelete non è un overlay (è inline in conversation_delete.go).
// editOverlay non esiste come overlay separato (edit modale via textarea).

import tea "github.com/charmbracelet/bubbletea"

// dispatchClickOverlay routes a left-click when an overlay is active.
// Uses OVERLAY_FIRST invariant: base-layer is never consulted.
func (m MainModel) dispatchClickOverlay(msg tea.MouseMsg) (MainModel, tea.Cmd) {
	x, y := msg.X, msg.Y

	// Identify the active overlay and its bbox.
	overlayBbox, isDismissable := m.activeOverlayBbox()

	if overlayBbox.Contains(x, y) {
		// Click inside overlay: forward raw msg to the overlay sub-model.
		// Most overlays have no internal mouse handling in Step 32 (keyboard-only).
		return m.forwardClickToOverlay(msg)
	}

	// Click outside overlay.
	if isDismissable {
		return m.closeActiveOverlay()
	}
	// Modal overlay: no-op (ADR-020 §D5 — forwardPicker).
	return m, nil
}

// activeOverlayBbox returns the bounding box and dismissable flag for the
// currently active overlay. Priority matches anyOverlayActive check order.
// Full-screen overlays return overlayFullscr; chatInfo returns chatInfoOverlay.
func (m MainModel) activeOverlayBbox() (Bbox, bool) {
	switch {
	case m.cmdPalette.Active:
		return m.bboxes.overlayFullscr, true // dismissable
	case m.help.Active:
		return m.bboxes.overlayFullscr, true // dismissable
	case m.search.Active:
		return m.bboxes.overlayFullscr, true // dismissable
	case m.whichKey.IsVisible():
		return m.bboxes.overlayFullscr, true // dismissable
	case m.chatInfo.Active:
		return m.bboxes.chatInfoOverlay, true // dismissable (right-anchored)
	case m.conversation.forwardPicker.Active():
		return m.bboxes.overlayFullscr, false // MODAL: no-op outside
	default:
		// Unreachable: anyOverlayActive() guard ensures one of the cases above hits.
		return Bbox{}, false
	}
}

// forwardClickToOverlay forwards the raw MouseMsg to the active overlay sub-model.
// In Step 32 most overlays have no internal mouse handling; this is a no-op hook
// for future extension (ADR-020 §D5 note on Step 32 scope).
func (m MainModel) forwardClickToOverlay(msg tea.MouseMsg) (MainModel, tea.Cmd) {
	// Future: forward msg to specific sub-model for in-overlay row click.
	// For now all overlay sub-models ignore MouseMsg internally.
	return m, nil
}

// closeActiveOverlay emits the correct *CloseMsg for the active dismissable overlay.
// Mirrors the keyboard Esc path exactly (KEYBOARD_PARITY invariant, ADR-020 §D8).
func (m MainModel) closeActiveOverlay() (MainModel, tea.Cmd) {
	switch {
	case m.cmdPalette.Active:
		m.cmdPalette = m.cmdPalette.Close()
	case m.help.Active:
		m.help = m.help.Close()
	case m.search.Active:
		m.search = m.search.Close()
	case m.whichKey.IsVisible():
		m.whichKey = m.whichKey.Cancel("")
	case m.chatInfo.Active:
		m.chatInfo = m.chatInfo.Close()
	}
	return m, nil
}
