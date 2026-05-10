package components

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// View renderizza le celle OTP.
// Stili calcolati per-frame per supportare hot-reload del tema.
func (m OTPInputModel) View() string {
	cellStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary()).
		Width(3).
		Align(lipgloss.Center)

	cellActiveStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorText()).
		Width(3).
		Align(lipgloss.Center)

	cellFilledStyle := lipgloss.NewStyle().Foreground(styles.ColorText())

	cells := make([]string, m.length)
	for i := range m.length {
		content := " "
		if m.cells[i] != 0 {
			content = cellFilledStyle.Render(string(m.cells[i]))
		}
		if i == m.cursor && m.focused {
			cells[i] = cellActiveStyle.Render(content)
		} else {
			cells[i] = cellStyle.Render(content)
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, cells...)
}
