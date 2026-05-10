package views

import (
	"strconv"
	"strings"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// renderReactionsRow produce la riga emoji sotto il bubble (Step 25).
// Formato: "👍 3  ❤️ 2  😂 1" — spazio tra emoji e count, due spazi tra entry.
// ChosenByMe evidenziato con ReactionChosen style.
// Ritorna "" se la slice è vuota (caller non aggiunge newline superflua).
func renderReactionsRow(reactions []model.Reaction) string {
	if len(reactions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(reactions))
	for _, r := range reactions {
		entry := r.Emoji + " " + strconv.Itoa(r.Count)
		if r.ChosenByMe {
			parts = append(parts, styles.ReactionChosenStyle().Render(entry))
		} else {
			parts = append(parts, styles.ReactionStyle().Render(entry))
		}
	}
	return strings.Join(parts, "  ")
}
