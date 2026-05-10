package views

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// matchCurrentStyle e matchOtherStyle sono funzioni per leggere il tema attivo
// ad ogni call da View() (supporta hot-reload — ADR-019 D9).
func matchCurrentStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(styles.ColorPrimary()).
		Foreground(styles.ColorText()).
		Bold(true)
}

func matchOtherStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(styles.ColorSearchSecondary()). // viola tenue — search non-current
		Foreground(styles.ColorText())
}

// applyHighlight decorates text inserting lipgloss spans on match positions.
// spans must be sorted ascending (allOccurrences guarantees this).
// isCurrent distinguishes the active match (stronger highlight) from others.
func applyHighlight(text string, spans []TextSpan, isCurrent bool) string {
	if len(spans) == 0 {
		return text
	}
	style := matchOtherStyle()
	if isCurrent {
		style = matchCurrentStyle()
	}
	var result []byte
	prev := 0
	tb := []byte(text)
	for _, sp := range spans {
		if sp.Start > len(tb) {
			break
		}
		end := sp.End
		if end > len(tb) {
			end = len(tb)
		}
		result = append(result, tb[prev:sp.Start]...)
		result = append(result, []byte(style.Render(string(tb[sp.Start:end])))...)
		prev = end
	}
	result = append(result, tb[prev:]...)
	return string(result)
}

// highlightText restituisce il testo del messaggio con gli span evidenziati.
// Se la search non è attiva o il msg non ha match, ritorna il testo originale.
func (m ConversationModel) highlightText(msgID int, text string) string {
	if !m.searchBar.Active || m.searchBar.Query == "" {
		return text
	}
	spans := m.searchBar.spansFor(msgID)
	if len(spans) == 0 {
		return text
	}
	isCurrent := m.searchBar.currentMsgID() == msgID
	return applyHighlight(text, spans, isCurrent)
}
