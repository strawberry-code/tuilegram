package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// conversation_bubble_render.go — Step 34: outgoing/incoming bubble rendering.
// Estratto da conversation_render.go per LOC compliance.
// bubble Width=bubbleW + Align(Right|Left) → wrapped lines pulled correttamente.
// Per outgoing: outer Width=w Align=Right pinna bubble al margine destro.

// Step 34: ogni riga (post-wrap) viene right-allineata individualmente.
// lipgloss.Width+Align tratta il blocco come unità → padding solo a sinistra
// del blocco, non per-line. Soluzione: split su \n + PlaceHorizontal per riga.
func (m ConversationModel) renderOutgoing(
	msg model.Message, prefix, mediaLine, timeStr string, isLast bool, w int, displayText string,
) string {
	receipt := ""
	if isLast {
		receipt = " " + receiptStr(msg.Status)
	}
	body := mediaLine + outgoingTextStyle().Render(displayText) + timeStr + receipt
	suffix := strings.TrimSuffix(prefix, " ")
	if suffix != "" {
		suffix = " " + suffix
	}
	return alignLines(body+suffix, w, lipgloss.Right)
}

// Step 34: incoming → left flush per-line.
func (m ConversationModel) renderIncoming(
	msg model.Message, prefix, mediaLine, timeStr string, w int, displayText string,
) string {
	linkedText := renderTextWithLinks(displayText, msg.Links)
	body := renderForwardBlock(msg, linkedText)
	text := incomingTextStyle().Render(" " + body)
	var content string
	if isDM(m.chat) {
		content = prefix + mediaLine + text + timeStr
	} else {
		name := coloredNameStyle(m.chat, msg.SenderID).Render(msg.SenderName + ":")
		content = prefix + name + mediaLine + text + timeStr
	}
	return alignLines(content, w, lipgloss.Left)
}

// alignLines: split su \n, applica PlaceHorizontal(width, pos) a ogni riga,
// rejoin. Garantisce alignment per-line indipendente dal blocco.
func alignLines(s string, width int, pos lipgloss.Position) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = lipgloss.PlaceHorizontal(width, pos, l)
	}
	return strings.Join(lines, "\n")
}
