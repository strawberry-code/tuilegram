package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleKey routes keys per statechart §A keybindings table.
// Priority: esc → close; enter → submit; j/k/↑/↓ → cursor; else → input.
func (m CmdPaletteModel) handleKey(msg tea.KeyMsg) (CmdPaletteModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.Close(), func() tea.Msg { return CmdPaletteCloseMsg{} }

	case "enter":
		return m.handleSubmit()

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
	}

	// All other printable keys + backspace → forward to textinput.
	return m.forwardToInput(msg)
}

// handleSubmit executes palette dispatch atomicity (ADR-015 §D3):
//  1. palette closes FIRST (activeOverlay := none)
//  2. CmdPaletteSubmitMsg is emitted
//  3. handler's tea.Cmd is returned for the NEXT Update cycle
//
// No-op if filtered list is empty.
func (m CmdPaletteModel) handleSubmit() (CmdPaletteModel, tea.Cmd) {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return m, nil
	}
	entry := m.filtered[m.cursor]
	m = m.Close() // step 1: close before dispatch

	submitCmd := func() tea.Msg { return CmdPaletteSubmitMsg{CmdID: entry.ID} }
	if entry.Handler == nil {
		return m, submitCmd
	}
	// Handler may itself return a tea.Cmd (e.g. open another overlay in next cycle).
	handlerCmd := entry.Handler()
	if handlerCmd == nil {
		return m, submitCmd
	}
	return m, tea.Batch(submitCmd, handlerCmd)
}

// forwardToInput delivers the key to the textinput and detects query changes.
// On change: re-runs fuzzy filter synchronously (O(|registry|*|title|) ≈ <0.1ms),
// resets cursor to 0 (invariant: ShowingAll/ResultsNonEmpty always starts at top).
func (m CmdPaletteModel) forwardToInput(msg tea.KeyMsg) (CmdPaletteModel, tea.Cmd) {
	prevQuery := m.query
	var tiCmd tea.Cmd
	m.input, tiCmd = m.input.Update(msg)
	m.query = m.input.Value()
	if m.query == prevQuery {
		return m, tiCmd
	}
	m.filtered = cmdFuzzyFilter(m.registry, m.query)
	m.cursor = 0
	return m, tiCmd
}
