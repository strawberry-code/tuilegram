package views

// main_notify.go — Step 34 (ADR-022): notify banner dispatch + emit helpers.
// Routes NotifyMsg/tick/timeout to the embedded NotifyModel and provides
// EmitNotifyCmd shortcuts for callers (send success, errors, etc.).

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/ui/components"
)

// handleNotify dispatches a notify-related message to the embedded NotifyModel.
// Returns (cmd, true) when the message was handled; (nil, false) otherwise.
func (m MainModel) handleNotify(msg tea.Msg) (MainModel, tea.Cmd, bool) {
	if !components.IsNotifyMsg(msg) {
		return m, nil, false
	}
	var cmd tea.Cmd
	m.Notify, cmd = m.Notify.Update(msg)
	return m, cmd, true
}

// EmitNotifyCmd returns a tea.Cmd that injects a NotifyMsg into the loop.
// Use to surface short-lived feedback (e.g., "Sent", "Forwarded", "✕ Failed").
func EmitNotifyCmd(kind components.NotifyKind, text string) tea.Cmd {
	return func() tea.Msg { return components.NotifyMsg{Kind: kind, Text: text} }
}
