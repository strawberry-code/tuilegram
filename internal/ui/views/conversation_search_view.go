package views

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

func searchBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(styles.ColorText()).
		Background(styles.ColorSearchInlineBg()).
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorPrimary()).
		PaddingLeft(1).PaddingRight(1)
}

// renderSearchBar produce la barra inline di ricerca da inserire nel layout
// sotto il viewport e sopra l'input area (ADR-014 §D1).
func (m ConversationModel) renderSearchBar() string {
	icon := "/ "
	input := m.searchInput.View()
	counter := m.searchBarCounter()
	hint := lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render("  Enter=next  Shift+Tab=prev  Esc=close")
	content := icon + input + "  " + counter + hint
	return searchBarStyle().Width(m.Width - 2).Render(content)
}

// searchBarCounter restituisce la stringa "N/M" oppure "No matches".
func (m ConversationModel) searchBarCounter() string {
	n := len(m.searchBar.Matches)
	if m.searchBar.Query == "" {
		return lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render("type to search")
	}
	if n == 0 {
		return lipgloss.NewStyle().Foreground(styles.ColorError()).Render("0/0  No matches")
	}
	s := fmt.Sprintf("%d/%d", m.searchBar.CurrentIdx+1, n)
	return lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true).Render(s)
}
