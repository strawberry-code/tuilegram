package styles

import "github.com/charmbracelet/lipgloss"

// SystemMessageStyle ritorna lo stile per i system message (join/leave/pin/ecc.).
// Funzione invece di var per leggere il tema attivo ad ogni chiamata da View().
// Costo: una lipgloss.Style allocation per call — accettabile nel render path.
func SystemMessageStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColorTextDim()).
		Italic(true)
}

// ReactionStyle ritorna lo stile di default per una entry della reactions row.
func ReactionStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColorReaction())
}

// ReactionChosenStyle ritorna lo stile per una reaction scelta dall'utente.
func ReactionChosenStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColorReactionChosen()).
		Underline(true)
}
