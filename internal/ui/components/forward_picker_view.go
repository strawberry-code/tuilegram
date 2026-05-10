package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

const pickerMaxVisible = 8

// pickerView renders the forward picker overlay via lipgloss.Place.
func pickerView(m ForwardPickerModel, width, height int) string {
	if m.Forwarding() {
		return renderForwardingSpinner(width, height)
	}

	boxW := min(width-8, 56)
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary()).MarginBottom(1)
	hintStyle := lipgloss.NewStyle().Foreground(styles.ColorTextDim()).MarginTop(1)
	errStyle := lipgloss.NewStyle().Foreground(styles.ColorError())
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorPrimary()).Padding(1, 2)

	title := titleStyle.Render("Forward to")
	search := m.input.View()
	listStr := renderPickerList(m, boxW, hintStyle)

	var errLine string
	if m.lastErr != nil {
		errLine = errStyle.Render("Error: " + m.lastErr.Error())
	}
	hint := hintStyle.Render("j/k nav  Enter select  Esc cancel")

	parts := []string{title, search, listStr}
	if errLine != "" {
		parts = append(parts, errLine)
	}
	parts = append(parts, hint)

	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	box := boxStyle.Width(boxW).Render(content)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func renderPickerList(m ForwardPickerModel, boxW int, hintStyle lipgloss.Style) string {
	selectedStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true)
	normalStyle := lipgloss.NewStyle().Foreground(styles.ColorText())
	if len(m.filtered) == 0 {
		return hintStyle.Render("No chats found")
	}

	start := 0
	end := len(m.filtered)
	if end > pickerMaxVisible {
		half := pickerMaxVisible / 2
		start = m.cursor - half
		if start < 0 {
			start = 0
		}
		end = start + pickerMaxVisible
		if end > len(m.filtered) {
			end = len(m.filtered)
			start = max(0, end-pickerMaxVisible)
		}
	}

	var sb strings.Builder
	for i := start; i < end; i++ {
		label := truncateTitle(m.filtered[i].Title, boxW-6)
		if i == m.cursor {
			sb.WriteString(selectedStyle.Render(fmt.Sprintf("▶ %s", label)))
		} else {
			sb.WriteString(normalStyle.Render(fmt.Sprintf("  %s", label)))
		}
		if i < end-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func renderForwardingSpinner(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary()).MarginBottom(1)
	boxStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(styles.ColorPrimary()).Padding(1, 2)
	msg := titleStyle.Render("Forwarding...")
	box := boxStyle.Render(msg)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func truncateTitle(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
