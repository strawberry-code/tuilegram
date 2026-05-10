package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
)

const typingTTL = 5 * time.Second

// TypingState mantiene lo stato di typing per-peer (ADR-010: timestamp-based).
// Assenza dalla mappa ≡ Idle. Presenza con now-lastTypingAt < TTL ≡ Typing.Active.
type TypingState struct {
	LastTypingAt time.Time
	UserID       int64
}

// TypingTimeoutMsg è inviato da tea.Tick(5s) per scadere il TTL di typing.
// scheduledAt è il timestamp del UpdateUserTypingMsg che ha armato il tick:
// se typing[peer].lastTypingAt > scheduledAt → tick stale → no-op (ADR-010).
type TypingTimeoutMsg struct {
	Peer        model.ChatID
	ScheduledAt time.Time
}

// scheduleTypingTimeoutCmd schedula un tick che scade dopo typingTTL.
// Conforme ADR-010 (re-arm): ogni UpdateUserTyping arma un nuovo tick;
// i tick stale vengono ignorati dal handler via check su lastTypingAt.
func scheduleTypingTimeoutCmd(peer model.ChatID, at time.Time) tea.Cmd {
	return tea.Tick(typingTTL, func(_ time.Time) tea.Msg {
		return TypingTimeoutMsg{Peer: peer, ScheduledAt: at}
	})
}
