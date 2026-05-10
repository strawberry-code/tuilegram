package views

import tea "github.com/charmbracelet/bubbletea"

// anyTextInputActive reports whether any text-entry widget currently owns
// keyboard input. Whenever true, single printable runes MUST be routed to
// the focused widget unchanged — global single-key shortcuts (F, i, /, ?,
// which-key prefixes) are suppressed by handleGlobalKey via this guard.
//
// Add new cases here when introducing widgets that consume free-form text
// (textinput / textarea). Failure to do so re-introduces shortcut bleed-through
// into user typing — the bug class this helper exists to eliminate.
func (m MainModel) anyTextInputActive() bool {
	return m.conversation.inputFocus ||
		m.conversation.forwardPicker.Active() ||
		m.search.Active ||
		m.cmdPalette.Active
}

// isPrintableKey reports whether msg is a printable character insertion that
// must reach a focused text input verbatim. KeyRunes covers letters/digits
// /symbols; KeySpace covers the spacebar. Alt-modified keys are excluded so
// Alt-chord shortcuts still reach handlers; control/function keys are
// excluded because they are not printable.
func isPrintableKey(msg tea.KeyMsg) bool {
	if msg.Alt {
		return false
	}
	switch msg.Type {
	case tea.KeyRunes, tea.KeySpace:
		return true
	}
	return false
}
