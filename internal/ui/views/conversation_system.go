package views

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// renderSystemMessage produce la riga centrata e dimmata per system messages.
// Formato: "── {ServiceText} ──"  (Step 25, reactions-and-system.md §Render UI).
// Centrato sull'asse orizzontale del viewport tramite lipgloss.Place.
func renderSystemMessage(serviceText string, w int) string {
	label := "── " + serviceText + " ──"
	// Tronca con ellipsis se supera la larghezza disponibile (caso raro).
	if lipgloss.Width(label) > w {
		label = truncate("── "+serviceText+" ──", w-1) + "…"
	}
	return lipgloss.Place(w, 1, lipgloss.Center, lipgloss.Center,
		styles.SystemMessageStyle().Render(label))
}

// renderReplyQuote produce la barra di citazione per i messaggi con reply.
func renderReplyQuote(replyToID int, msgs []model.Message, w int) string {
	_ = w // larghezza riservata per futura espansione
	quoted := findMsgByID(msgs, replyToID)
	if quoted == nil {
		return replyBarStyle().Render("↩ (messaggio non disponibile)")
	}
	preview := truncate(quoted.Text, 40)
	return replyBarStyle().Render("↩ " + quoted.SenderName + ": " + preview)
}

// findMsgByID cerca un messaggio per ID nella slice dei messaggi.
func findMsgByID(msgs []model.Message, id int) *model.Message {
	for i := range msgs {
		if msgs[i].ID == id {
			return &msgs[i]
		}
	}
	return nil
}

// truncate tronca la stringa a n rune aggiungendo "…" se necessario.
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}
