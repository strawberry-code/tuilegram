package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// renderInfoBody builds the pre-rendered card content for the viewport.
func renderInfoBody(card ChatInfoCard) string {
	var sb strings.Builder
	sb.WriteString(renderIdentitySection(card))
	if contact := renderContactSection(card); contact != "" {
		sb.WriteString(divider())
		sb.WriteString(contact)
	}
	if profile := renderProfileSection(card); profile != "" {
		sb.WriteString(divider())
		sb.WriteString(profile)
	}
	sb.WriteString(divider())
	sb.WriteString(renderCountersSection(card))
	return sb.String()
}

func renderIdentitySection(card ChatInfoCard) string {
	val := lipgloss.NewStyle().Foreground(styles.ColorText())
	online := lipgloss.NewStyle().Foreground(styles.ColorSuccess())
	name := val.Bold(true).Render(card.Name)
	var lines []string
	lines = append(lines, name)
	if card.Username != "" {
		lines = append(lines, val.Render("@"+card.Username))
	}
	if card.OnlineStatus != "" {
		lines = append(lines, online.Render(card.OnlineStatus))
	}
	return strings.Join(lines, "\n") + "\n"
}

// renderContactSection returns "" for non-private chats (OMIT rule, ADR-017 §D5).
func renderContactSection(card ChatInfoCard) string {
	if card.ChatType != model.ChatPrivate || card.Phone == "" {
		return ""
	}
	label := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	val := lipgloss.NewStyle().Foreground(styles.ColorText())
	return label.Render("Phone: ") + val.Render(card.Phone) + "\n"
}

// renderProfileSection returns "" when bio is not applicable for the chat type.
func renderProfileSection(card ChatInfoCard) string {
	bio := card.Bio
	if bio == "" {
		switch card.ChatType {
		case model.ChatPrivate, model.ChatBot:
			bio = "—"
		default:
			return ""
		}
	}
	label := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	val := lipgloss.NewStyle().Foreground(styles.ColorText())
	return label.Render("Bio:\n") + val.Render(bio) + "\n"
}

func renderCountersSection(card ChatInfoCard) string {
	label := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	val := lipgloss.NewStyle().Foreground(styles.ColorText())
	return label.Render("Shared Media ") + val.Render(counterStr(card.SharedMediaCount)) + "\n" +
		label.Render("Shared Files ") + val.Render(counterStr(card.SharedFilesCount)) + "\n" +
		label.Render("Shared Links ") + val.Render(counterStr(card.SharedLinksCount)) + "\n"
}

func counterStr(n int) string {
	if n < 0 {
		return "[?]"
	}
	return "[" + itoa(n) + "]"
}

func divider() string {
	return lipgloss.NewStyle().Foreground(styles.ColorBorder()).Render("────────────────") + "\n"
}

// itoa converts a non-negative int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
