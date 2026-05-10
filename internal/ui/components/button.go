package components

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// ButtonModel rappresenta un bottone semplice.
type ButtonModel struct {
	Label  string
	Active bool
}

// NewButton crea un nuovo bottone con l'etichetta data.
func NewButton(label string) ButtonModel {
	return ButtonModel{Label: label}
}

// View restituisce la rappresentazione visiva del bottone.
// Stili calcolati al momento del render per supportare hot-reload del tema.
func (b ButtonModel) View() string {
	if b.Active {
		return lipgloss.NewStyle().
			Foreground(styles.ColorButtonFg()).
			Background(styles.ColorButtonBg()).
			Padding(0, 1).
			Render(b.Label)
	}
	return lipgloss.NewStyle().
		Foreground(styles.ColorButtonDisabledFg()).
		Padding(0, 1).
		Render(b.Label)
}
