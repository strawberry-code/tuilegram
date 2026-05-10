package components

// modal_placement.go — Placement enum for the Modal primitive (Step 29).
// Extends the existing Modal struct with a positional anchor field.
// ADR-017 §D1: minimal enum extension, no new component created.

import "github.com/charmbracelet/lipgloss"

// Placement controls where Modal.Render anchors the box on the canvas.
type Placement int

const (
	// PlacementCenter is the default: box centered on the canvas.
	// Used by palette, help, search, edit, forward, confirm.
	PlacementCenter Placement = iota

	// PlacementBottomRight anchors the box to the bottom-right corner.
	// Used by which-key (Step 28). Note: which-key renders its box
	// manually via renderCompactBox; this value is reserved for future
	// callers that use Modal.Render directly.
	PlacementBottomRight

	// PlacementRight anchors the box to the right edge, vertically centered.
	// Used by chat info overlay (Step 29, ADR-017 §D1).
	PlacementRight
)

// placementToLipgloss maps a Placement to the lipgloss horizontal/vertical pair.
func placementToLipgloss(p Placement) (lipgloss.Position, lipgloss.Position) {
	switch p {
	case PlacementBottomRight:
		return lipgloss.Right, lipgloss.Bottom
	case PlacementRight:
		return lipgloss.Right, lipgloss.Center
	default: // PlacementCenter
		return lipgloss.Center, lipgloss.Center
	}
}
