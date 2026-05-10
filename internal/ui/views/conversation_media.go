package views

import "github.com/strawberry-code/tuilegram/internal/model"

// buildMediaLine restituisce la stringa media da inserire prima del testo
// nel bubble render, o "" se il messaggio non ha media (Step 24).
// Additivo: non modifica logica di allineamento, grouping, reply, receipts.
func buildMediaLine(msg model.Message) string {
	if msg.Media == nil {
		return ""
	}
	return msg.Media.Summary() + " "
}
