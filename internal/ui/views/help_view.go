package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/components"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

func renderHelpModal(m HelpModel) string {
	return helpModalFor(m).Render(m.Width, m.Height)
}

// overlayHelp renders help modal composited on bg (Crush-style).
func overlayHelp(bg string, m HelpModel, width int) string {
	return helpModalFor(m).RenderOverlay(bg, width)
}

func helpModalFor(m HelpModel) components.Modal {
	return components.Modal{
		Title: "Keybindings",
		Body:  buildHelpBody(m.scrollOffset),
		Hints: "↑↓ scroll  ·  esc / ? close",
	}
}

// buildHelpBody generates the full keybinding reference, applying a scroll offset.
func buildHelpBody(offset int) string {
	lines := allHelpLines()
	if offset < 0 {
		offset = 0
	}
	if offset >= len(lines) {
		offset = len(lines) - 1
	}
	return strings.Join(lines[offset:], "\n")
}

// allHelpLines returns every help line in section order.
func allHelpLines() []string {
	sectionStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary())
	descStyle := lipgloss.NewStyle().Foreground(styles.ColorText())

	type row = [2]string
	sections := []struct {
		title string
		rows  []row
	}{
		{"Global", []row{
			{"Ctrl+P", "Command palette"}, {"?", "Help overlay"},
			{"/", "Global search"}, {"Ctrl+Q", "Quit"},
		}},
		{"Navigation", []row{
			{"gg / g→g", "Scroll to top"}, {"G / g→G", "Scroll to bottom"},
			{"g→u", "Jump to next unread"}, {"g→i", "Chat info"},
			{"zz / z→z", "Center message"}, {"h / Esc", "Back to chat list"},
		}},
		{"Conversation", []row{
			{"j / k", "Move cursor"}, {"Ctrl+D / Ctrl+U", "Half-page scroll"},
			{"r", "Reply"}, {"e", "Edit message"},
			{"D", "Delete message"}, {"f", "Forward message"},
			{"Space", "Select/deselect"}, {"i / Tab", "Focus input"},
			{"Ctrl+F", "Search in chat"},
		}},
		{"Overlays", []row{
			{"Esc", "Close active overlay"}, {"Enter", "Confirm / submit"},
			{"j / k", "Navigate list"},
		}},
	}
	var lines []string
	for _, s := range sections {
		lines = append(lines, sectionStyle.Render("── "+s.title+" ──"))
		for _, r := range s.rows {
			line := keyStyle.Render(r[0]) + descStyle.Render("  "+r[1])
			lines = append(lines, line)
		}
	}
	return lines
}
