package views

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// renderHeader restituisce la riga header della conversazione.
// In MultiSelect mostra la barra di selezione; in Typing.Active aggiunge
// "— typing..." alla riga di status (statechart typing-indicator.md §Render UI).
// Step 34: in DM (1-on-1) il lato destro mostra "You" — l'header agisce da
// banner partecipanti, sostituendo il prefisso "Sender:" su ogni messaggio.
func (m ConversationModel) renderHeader() string {
	if m.multiSelect {
		return m.renderMultiSelectBar()
	}
	name := lipgloss.NewStyle().Bold(true).Foreground(styles.ColorText()).Render(m.chat.Title)
	var statusText string
	if m.Typing {
		statusText = lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Italic(true).Render("— typing...")
	} else {
		statusText = lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render(renderChatStatus(m.chat))
	}
	left := name + "  " + statusText
	if isDM(m.chat) {
		you := lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true).Render("You")
		gap := m.Width - 2 - lipgloss.Width(left) - lipgloss.Width(you)
		if gap < 1 {
			gap = 1
		}
		left = left + lipgloss.NewStyle().Width(gap).Render("") + you
	}
	return headerBorderStyle().Width(m.Width - 2).Render(left)
}

// isDM è true per chat 1-on-1 (utente o bot).
func isDM(c model.Chat) bool {
	return c.Type == model.ChatPrivate || c.Type == model.ChatBot
}

// headerBorderStyle è la barra header standard (bottone, no multi-select).
func headerBorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorBorder())
}

// renderMultiSelectBar mostra l'info bar con conteggio selezione e shortcut.
// Visibile solo in MultiSelect (MODE_COHERENCE: len(selection) > 0).
func (m ConversationModel) renderMultiSelectBar() string {
	n := len(m.selection)
	text := fmt.Sprintf("%d selected  f=forward  D=delete  Esc=cancel", n)
	return multiSelectBarStyle().Width(m.Width - 2).Render(text)
}

func multiSelectBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(styles.ColorSuccess()).Bold(true).
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorSuccess())
}
