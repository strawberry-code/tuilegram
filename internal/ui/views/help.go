package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// HelpModel is the stateless help overlay (Step 28, statechart §C).
// Content is generated from compile-time data (registry + static sections).
// Scroll resets to 0 on Close — help is stateless between sessions (ADR-015 §D1).
type HelpModel struct {
	Active        bool
	scrollOffset  int // lines scrolled from top inside the overlay
	Width, Height int
}

// Open transitions Closed → Open.ShowingAll. Resets scroll (stateless per spec).
func (m HelpModel) Open() HelpModel {
	m.Active = true
	m.scrollOffset = 0
	return m
}

// Close transitions Open → Closed. Resets scroll.
func (m HelpModel) Close() HelpModel {
	m.Active = false
	m.scrollOffset = 0
	return m
}

// Update handles keys while Active. Non-active → no-op.
func (m HelpModel) Update(msg tea.Msg) (HelpModel, tea.Cmd) {
	if !m.Active {
		return m, nil
	}
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	return m.handleKey(keyMsg)
}

// handleKey dispatches navigation and close keys per statechart §C.
// esc/? → close; j/k/pgup/pgdn → scroll; all others → no-op (consumed).
func (m HelpModel) handleKey(msg tea.KeyMsg) (HelpModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "?":
		return m.Close(), func() tea.Msg { return HelpCloseMsg{} }
	case "j", "down":
		m.scrollOffset++
	case "k", "up":
		if m.scrollOffset > 0 {
			m.scrollOffset--
		}
	case "pgdown":
		m.scrollOffset += 10
	case "pgup":
		if m.scrollOffset > 10 {
			m.scrollOffset -= 10
		} else {
			m.scrollOffset = 0
		}
	}
	return m, nil
}

// View renders the help modal when Active; "" otherwise.
func (m HelpModel) View() string {
	if !m.Active {
		return ""
	}
	return renderHelpModal(m)
}
