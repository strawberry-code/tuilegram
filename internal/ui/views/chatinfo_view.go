package views

import (
	"github.com/strawberry-code/tuilegram/internal/ui/components"
)

// View renders the chat info overlay when Active; "" otherwise.
// Uses Modal primitive with compact placement: right (ADR-017 §D1).
// Body content is pre-rendered by renderInfoBody (chatinfo_render.go).
func (m ChatInfoModel) View() string {
	if !m.Active {
		return ""
	}
	body := m.vp.View()
	modal := components.Modal{
		Title:     "Info",
		Body:      body,
		Hints:     "↑↓ scroll · esc close",
		Width:     34,
		Placement: components.PlacementRight,
	}
	return modal.Render(m.Width, m.Height)
}
