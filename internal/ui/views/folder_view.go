package views

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

const folderSidebarWidth = 14

// View renders the folder sidebar. Returns "" when not visible.
func (m FolderModel) View() string {
	if !m.visible || m.Width == 0 {
		return ""
	}
	header := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorText()).PaddingLeft(1).Render("FOLDERS")
	rows := m.renderItems()
	content := lipgloss.JoinVertical(lipgloss.Left, header, rows)
	return lipgloss.NewStyle().
		Width(m.Width).Height(m.Height).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(styles.ColorBorder()).
		Render(content)
}

// renderItems renders all folder rows with cursor/selected/dim styling.
func (m FolderModel) renderItems() string {
	var rows string
	for i, f := range m.allFolders {
		rows += m.renderFolder(i, f) + "\n"
	}
	if len(rows) == 0 {
		rows = lipgloss.NewStyle().Foreground(styles.ColorTextDim()).PaddingLeft(2).Render("Loading…")
	}
	return rows
}

// renderFolder renders one folder row.
// Cursor (keyboard highlight) beats selected (filter applied) beats base style.
func (m FolderModel) renderFolder(idx int, f model.ChatFolder) string {
	label := fldTruncate(f.Title, m.Width-3)

	isCursor := m.focus == folderFocusBrowsing && idx == m.cursor
	isSelected := f.ID == m.selectedID

	switch {
	case isCursor:
		return lipgloss.NewStyle().Foreground(styles.ColorSurface()).Background(styles.ColorPrimary()).PaddingLeft(2).Render(label)
	case isSelected:
		return lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true).PaddingLeft(2).Render("▸ " + label)
	default:
		return lipgloss.NewStyle().Foreground(styles.ColorText()).PaddingLeft(2).Render(label)
	}
}

// fldTruncate shortens s to maxLen runes, appending "…" if needed.
func fldTruncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}
