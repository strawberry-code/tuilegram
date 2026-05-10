package components

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// pickerUpdate is the Update logic for ForwardPickerModel, factored out for LOC.
func pickerUpdate(m ForwardPickerModel, msg tea.Msg) (ForwardPickerModel, tea.Cmd) {
	if !m.Active() {
		return m, nil
	}

	// ADR-007: ignore all input while RPC is in flight.
	if m.Forwarding() {
		return m, nil
	}

	keyMsg, isKey := msg.(tea.KeyMsg)
	if !isKey {
		// Forward non-key msgs to the text input (e.g. blink tick).
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	switch keyMsg.String() {
	case "esc":
		return m.Close(), func() tea.Msg { return telegram.OverlayCloseMsg{} }

	case "enter":
		if chat, ok := m.Selected(); ok {
			m = m.BeginForwarding()
			// Caller (main_update) fills Source+Messages from snapshot.
			return m, func() tea.Msg { return telegram.ForwardSubmitMsg{Target: chat} }
		}
		return m, nil

	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
		return m, nil

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	default:
		// Printable char or backspace — delegate to textinput then re-rank.
		prev := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != prev {
			q := m.input.Value()
			m.filtered = FuzzyRank(m.allChats, q)
			m.cursor = 0
			m.lastErr = nil
		}
		return m, cmd
	}
}
