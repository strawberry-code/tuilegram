package views

import (
	"fmt"

	"github.com/strawberry-code/tuilegram/internal/ui/components"
)

// renderEditOverlay renders the edit-message modal via the unified Modal
// primitive (ADR-009: edit overlay reuses same rounded-border Crush style).
func (m ConversationModel) renderEditOverlay() string {
	modal := components.Modal{
		Title: "Edit message",
		Body:  m.textarea.View(),
		Hints: "enter save • esc cancel",
		Width: 64,
		Tone:  components.ToneDefault,
	}
	return modal.Render(m.Width, m.Height)
}

// renderDeleteOverlay renders the N-aware delete confirmation modal.
// N==1 → "Delete this message?"  N>1 → "Delete N messages?"
// Uses ToneWarning (red border) to signal destructive action (ADR-009).
func (m ConversationModel) renderDeleteOverlay() string {
	n := len(m.deleteMsgIDs)
	var title string
	if n <= 1 {
		title = "Delete this message?"
	} else {
		title = fmt.Sprintf("Delete %d messages?", n)
	}
	modal := components.Modal{
		Title: title,
		Body:  "",
		Hints: "[y] yes • [n] no • esc cancel",
		Width: 52,
		Tone:  components.ToneWarning,
	}
	return modal.Render(m.Width, m.Height)
}
