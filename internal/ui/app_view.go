package ui

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

func (m AppModel) View() string {
	switch m.state {
	case StateInitializing:
		return m.viewInitializing()
	case StateAuth:
		return m.auth.View()
	case StateMain:
		return m.main.View()
	default:
		return ""
	}
}

func (m AppModel) viewInitializing() string {
	msg := lipgloss.NewStyle().
		Foreground(styles.ColorTextDim()).
		Render(m.spinner.View() + " Connecting to Telegram...")

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
}
