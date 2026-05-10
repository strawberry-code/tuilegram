package views

import (
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// receiptStr restituisce il simbolo di delivery status per i messaggi outgoing.
func receiptStr(status model.DeliveryStatus) string {
	dim := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	blue := lipgloss.NewStyle().Foreground(styles.ColorIncoming())
	switch status {
	case model.StatusSent:
		return dim.Render("✓")
	case model.StatusDelivered:
		return dim.Render("✓✓")
	case model.StatusRead:
		return blue.Render("✓✓")
	}
	return ""
}

// sameGroup verifica se due messaggi appartengono allo stesso gruppo
// (stesso sender, entro 5 minuti).
func sameGroup(a, b model.Message) bool {
	if a.IsOutgoing != b.IsOutgoing {
		return false
	}
	if !a.IsOutgoing && a.SenderName != b.SenderName {
		return false
	}
	return b.Date.Sub(a.Date) < 5*time.Minute
}

// differentDay verifica se due timestamp appartengono a giorni diversi.
func differentDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay != by || am != bm || ad != bd
}

// renderChatStatus restituisce il testo di stato in base al tipo di chat.
func renderChatStatus(chat model.Chat) string {
	switch chat.Type {
	case model.ChatPrivate, model.ChatBot:
		if chat.IsOnline {
			return "● online"
		}
		return "○ offline"
	case model.ChatGroup:
		return "group"
	case model.ChatChannel:
		return "channel"
	case model.ChatSavedMessages:
		return "saved messages"
	}
	return ""
}

// renderDateSeparator crea il separatore di data centrato.
func renderDateSeparator(t time.Time, w int) string {
	label := " " + t.Format("Jan 2, 2006") + " "
	labelLen := len([]rune(label))
	padding := (w - labelLen) / 2
	if padding < 1 {
		padding = 1
	}
	dashes := strings.Repeat("─", padding)
	return lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render(dashes + label + dashes)
}
