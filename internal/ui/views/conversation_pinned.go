package views

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// pinnedBarHeight è l'altezza riservata dalla pinned bar quando visibile (ADR-021 §A4).
// 1 riga contenuto + 1 riga separator = 2 righe.
const pinnedBarHeight = 2

// loadPinnedMessageCmd avvia il fetch del pinned message in background.
// Invariante PINNED_STALE_DROP: il consumer controlla chatID prima di mutare stato.
func loadPinnedMessageCmd(bridge *telegram.Bridge, chat model.Chat) tea.Cmd {
	chatID := chat.ID
	return func() tea.Msg {
		msg, err := bridge.LoadPinnedMessage(context.Background(), chat)
		return telegram.PinnedMsgLoadedMsg{ChatID: chatID, Msg: msg, Err: err}
	}
}

// handlePinnedMsgLoaded gestisce PinnedMsgLoadedMsg applicando PINNED_STALE_DROP.
// Se chatID non corrisponde alla chat attiva → no-op (msg stale, drop).
// Se corrisponde → aggiorna pinnedMsg e ricalcola il viewport.
func (m ConversationModel) handlePinnedMsgLoaded(msg telegram.PinnedMsgLoadedMsg) (ConversationModel, tea.Cmd) {
	// PINNED_STALE_DROP: ignora se la chat attiva è cambiata.
	if msg.ChatID != m.chat.ID {
		return m, nil
	}
	if msg.Err != nil || msg.Msg == nil {
		m.pinnedMsg = nil
	} else {
		m.pinnedMsg = msg.Msg
	}
	// Ricalcola viewport per rispettare PINNED_OFFSET_RESERVED.
	m.SetSize(m.Width, m.Height)
	m.viewport.SetContent(m.renderMessages())
	return m, nil
}

// pinnedBar renderizza la barra del messaggio pinnato (2 righe: content + separator).
// Ritorna stringa vuota se pinnedMsg == nil (PINNED_SINGLE_PER_CHAT).
func (m ConversationModel) pinnedBar() string {
	if m.pinnedMsg == nil {
		return ""
	}
	icon := lipgloss.NewStyle().Foreground(styles.ColorPinned()).Render("📌")
	// Tronca il testo al pannello meno l'icona e spazi.
	available := m.Width - 6
	body := m.pinnedMsg.Text
	if m.pinnedMsg.Media != nil && body == "" {
		body = "[media]"
	}
	// NIT #3: tronca prima del link-render per evitare che truncateRunes tagli
	// dentro una escape sequence OSC 8 (renderebbe inutilizzabile l'hyperlink).
	body = truncateRunes(body, available)
	// Applica link-render per coerenza con la conversation view (ADR-021 §DB2).
	body = renderTextWithLinks(body, m.pinnedMsg.Links)
	content := icon + " " + lipgloss.NewStyle().Foreground(styles.ColorText()).Render(body)
	separator := strings.Repeat("─", m.Width-2)
	sep := lipgloss.NewStyle().Foreground(styles.ColorBorder()).Render(separator)
	return lipgloss.JoinVertical(lipgloss.Left, content, sep)
}

// truncateRunes tronca una stringa a maxRunes rune con ellipsis.
func truncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes-1]) + "…"
}
