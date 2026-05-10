package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// renderForwardBlock prepend il blocco "┃ From <X>" al corpo del messaggio.
// Invariante FORWARD_PREFIX_PER_LINE: ogni linea ha prefisso "┃ ".
// Invariante FORWARD_LABEL_FALLBACK_CHAIN: ForwardedFrom mai stringa vuota.
// Usato da renderMessages() solo se msg.IsForwarded == true (ADR-021 §DC2).
func renderForwardBlock(msg model.Message, bodyText string) string {
	if !msg.IsForwarded {
		return bodyText
	}
	dimItalic := lipgloss.NewStyle().
		Foreground(styles.ColorForwardLabel()).
		Italic(true)
	// Riga header: "┃ From @source"
	header := dimItalic.Render("┃ From " + msg.ForwardedFrom)
	// Ogni linea del body prefissata con "┃ ".
	lines := strings.Split(bodyText, "\n")
	prefixed := make([]string, len(lines))
	for i, l := range lines {
		prefixed[i] = lipgloss.NewStyle().Foreground(styles.ColorForwardLabel()).Render("┃ ") + l
	}
	return lipgloss.JoinVertical(lipgloss.Left, header, strings.Join(prefixed, "\n"))
}

// coloredNameStyle restituisce il lipgloss.Style per il nome del mittente.
// In ChatGroup: colore deterministico da senderID (ADR-021 §DE2, DE3).
// In altri tipi: ColorIncoming (default) + Bold.
// Invariante SENDER_COLOR_GROUP_ONLY: color hash solo per ChatGroup.
func coloredNameStyle(chat model.Chat, senderID int64) lipgloss.Style {
	base := lipgloss.NewStyle().Bold(true)
	if chat.Type != model.ChatGroup {
		return base.Foreground(styles.ColorIncoming())
	}
	// senderColor usa abs(id)%8 sulla palette (SENDER_COLOR_DETERMINISTIC).
	palette := styles.ColorSenderPalette()
	idx := senderID
	if idx < 0 {
		idx = -idx
	}
	return base.Foreground(palette[idx%8])
}
