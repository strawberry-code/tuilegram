package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// renderMessages formatta i messaggi con grouping, allineamento, cursor e reply quote.
// In MultiSelect mostra checkbox [✓]/[ ] prima di ogni messaggio.
// Branch IsService (Step 25): system messages → centrati dim; altri → bubble standard.
func (m ConversationModel) renderMessages() string {
	if len(m.messages) == 0 {
		return lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render("No messages")
	}
	w := m.viewport.Width
	var sb strings.Builder

	for i, msg := range m.messages {
		if i == 0 || differentDay(m.messages[i-1].Date, msg.Date) {
			sb.WriteString(renderDateSeparator(msg.Date, w))
			sb.WriteString("\n")
		}
		if msg.IsService {
			sb.WriteString(renderSystemMessage(msg.ServiceText, w))
			sb.WriteString("\n")
			continue
		}
		if msg.ReplyToID != 0 {
			sb.WriteString(renderReplyQuote(msg.ReplyToID, m.messages, w))
			sb.WriteString("\n")
		}
		isLast := i == len(m.messages)-1 || !sameGroup(msg, m.messages[i+1])
		var timeStr string
		if isLast {
			timeStr = "  " + msgTimeStyle().Render(msg.Date.Format("15:04"))
		}
		prefix := m.renderMsgPrefix(i, msg.ID)
		mediaLine := buildMediaLine(msg)
		displayText := wrapBubble(m.highlightText(msg.ID, msg.Text), w)
		if msg.IsOutgoing {
			sb.WriteString(m.renderOutgoing(msg, prefix, mediaLine, timeStr, isLast, w, displayText))
		} else {
			sb.WriteString(m.renderIncoming(msg, prefix, mediaLine, timeStr, w, displayText))
		}
		sb.WriteString("\n")
		if row := renderReactionsRow(msg.Reactions); row != "" {
			if msg.IsOutgoing {
				sb.WriteString(lipgloss.NewStyle().Width(w).Align(lipgloss.Right).Render(row))
			} else {
				sb.WriteString(row)
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// renderMsgPrefix restituisce il prefisso cursore/checkbox per il messaggio all'indice i.
func (m ConversationModel) renderMsgPrefix(i, msgID int) string {
	if m.multiSelect {
		if _, ok := m.selection[msgID]; ok {
			return selectedStyle().Render("[✓] ")
		}
		if i == m.cursor {
			return cursorStyle().Render("[ ] ")
		}
		return "[ ] "
	}
	if i == m.cursor {
		return cursorStyle().Render("▶ ")
	}
	return ""
}

func cursorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true)
}

func selectedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.ColorSuccess()).Bold(true)
}

func replyBarStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.ColorTextDim()).BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).BorderForeground(styles.ColorPrimary()).
		PaddingLeft(1)
}
