package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/components"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

const paletteMaxVisible = 10

// renderPaletteModal builds the Modal body and delegates centering to Modal.Render.
func renderPaletteModal(m CmdPaletteModel) string {
	return paletteModalFor(m).Render(m.Width, m.Height)
}

// overlayPalette renders palette modal composited on bg (Crush-style).
func overlayPalette(bg string, m CmdPaletteModel, width int) string {
	return paletteModalFor(m).RenderOverlay(bg, width)
}

func paletteModalFor(m CmdPaletteModel) components.Modal {
	title := "Commands"
	if m.query != "" {
		title = "Commands · " + m.query
	}
	body := lipgloss.JoinVertical(lipgloss.Left,
		m.input.View(),
		"",
		renderPaletteList(m),
	)
	return components.Modal{
		Title: title,
		Body:  body,
		Hints: "↵ run  ·  ↑↓ navigate  ·  esc close",
	}
}

// renderPaletteList renders the filtered command list with cursor highlight.
// Empty filtered list → dim placeholder.
func renderPaletteList(m CmdPaletteModel) string {
	dimStyle := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	normalStyle := lipgloss.NewStyle().Foreground(styles.ColorText())
	cursorStyle := lipgloss.NewStyle().Foreground(styles.ColorSurface()).Background(styles.ColorPrimary()).Bold(true)
	if len(m.filtered) == 0 {
		return dimStyle.Render("No commands match")
	}
	start, end := visibleWindow(m.cursor, len(m.filtered), paletteMaxVisible)
	var sb strings.Builder
	for i := start; i < end; i++ {
		e := m.filtered[i]
		label := buildCommandLabel(e, dimStyle)
		if i == m.cursor {
			sb.WriteString(cursorStyle.Render("▸ " + label))
		} else {
			sb.WriteString(normalStyle.Render("  " + label))
		}
		if i < end-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// buildCommandLabel formats a single command row: "Title  [key]  (Section)".
func buildCommandLabel(e CommandEntry, dimStyle lipgloss.Style) string {
	label := e.Title
	if len(e.Keys) > 0 {
		hint := dimStyle.Render("  [" + strings.Join(e.Keys, "/") + "]")
		label += hint
	}
	if e.Section != "" {
		sec := lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true).Render("  " + e.Section)
		label += sec
	}
	return label
}
