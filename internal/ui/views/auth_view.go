package views

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// Titolo ASCII art (figlet, font "ANSI Shadow").
const titleArt = "" +
	"████████╗██╗   ██╗██╗██╗     ███████╗ ██████╗ ██████╗  █████╗ ███╗   ███╗\n" +
	"╚══██╔══╝██║   ██║██║██║     ██╔════╝██╔════╝ ██╔══██╗██╔══██╗████╗ ████║\n" +
	"   ██║   ██║   ██║██║██║     █████╗  ██║  ███╗██████╔╝███████║██╔████╔██║\n" +
	"   ██║   ██║   ██║██║██║     ██╔══╝  ██║   ██║██╔══██╗██╔══██║██║╚██╔╝██║\n" +
	"   ██║   ╚██████╔╝██║███████╗███████╗╚██████╔╝██║  ██║██║  ██║██║ ╚═╝ ██║\n" +
	"   ╚═╝    ╚═════╝ ╚═╝╚══════╝╚══════╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝     ╚═╝"

func (m AuthModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}

	from, to := styles.GradientColors()
	title := styles.RenderGradient(titleArt, from, to)

	inputBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary()).
		Padding(0, 1).Width(24)

	var inputRow string
	switch m.Step {
	case AuthStepPhone:
		inputRow = m.renderRow("Enter your\nphone number:", inputBoxStyle.Render(m.phone.View()))
	case AuthStepCode:
		inputRow = m.renderRow("Enter your\n2FA code:", m.otp.View())
	case AuthStepPassword:
		inputRow = m.renderRow("Enter your\npassword:", inputBoxStyle.Render(m.password.View()))
	}

	var errLine string
	if m.Err != "" {
		errLine = lipgloss.NewStyle().Foreground(styles.ColorError()).Render("\n" + m.Err)
	}

	content := lipgloss.JoinVertical(lipgloss.Center, title, "", "", inputRow, errLine)
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
}

func (m AuthModel) renderRow(label, input string) string {
	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorText())
	hintStyle := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	return lipgloss.JoinHorizontal(lipgloss.Center,
		labelStyle.Render(label), "    ", input, hintStyle.Render(" ↵"),
	)
}
