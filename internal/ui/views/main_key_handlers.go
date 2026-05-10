package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleWhichKeyInput processes a key while which-key is pending or visible.
func (m MainModel) handleWhichKeyInput(msg tea.KeyMsg) (MainModel, tea.Cmd) {
	key := msg.String()
	prefix := m.whichKey.ActivePrefix()
	if key == "esc" {
		m.whichKey = m.whichKey.Cancel("")
		return m, func() tea.Msg { return WhichKeyCancelMsg{Prefix: prefix} }
	}
	conts, hasCont := m.whichKey.continuations[prefix]
	if hasCont {
		if _, ok := conts[key]; ok {
			var chordCmd tea.Cmd
			m.whichKey, chordCmd = m.whichKey.HandleContinuation(prefix, key)
			return m, chordCmd
		}
	}
	m.whichKey = m.whichKey.Cancel(key)
	reDispatch := func() tea.Msg { return msg }
	return m, reDispatch
}

// handleGlobalKey handles keys when no overlay is active.
// Input-safety guard (main_input_safety.go): any printable rune is routed to
// the focused text widget unchanged whenever a textinput/textarea owns input.
// This is the single algorithmic guarantee that typing in the composer/search
// inputs cannot be hijacked by single-key shortcuts (F, i, /, ?, g, z, ...).
func (m MainModel) handleGlobalKey(msg tea.KeyMsg) (MainModel, tea.Cmd) {
	if m.anyTextInputActive() && isPrintableKey(msg) {
		return m.handleFocusKey(msg)
	}
	key := msg.String()
	switch key {
	// Step 30: Tab is context-aware — Compact=panel switch, Wide=focus cycle (ADR-018 §D3).
	case "tab":
		if m.layoutMode == LayoutCompact {
			return m.handleLayoutPanelSwitch()
		}
		return m.handleFocusKey(msg) // Wide: delegate to existing focus-cycle logic
	case "ctrl+p":
		m.cmdPalette = m.cmdPalette.Open()
		return m, nil
	case "?":
		m.help = m.help.Open()
		return m, nil
	case "/":
		m.search = m.search.Open()
		return m, nil
	case "F": // Step 29: toggle folder sidebar (ADR-016 §D5: no-op in compact mode)
		if m.layoutMode == LayoutCompact {
			m.statusMsg = "Folders not available in compact mode (<100 cols). Resize terminal."
			return m, nil
		}
		return m.handleFolderMsg(FolderToggleMsg{})
	case "i": // Step 29: open chat info overlay
		return m.handleChatInfoOpen()
	case "1", "2", "3", "4", "5", "6", "7", "8", "9": // Step 34: tab switch by index
		idx := int(key[0]-'0') - 1
		if idx >= 0 && idx < len(m.folderModel.allFolders) {
			f := m.folderModel.allFolders[idx]
			return m.handleFolderMsg(FolderSelectMsg{FolderID: f.ID})
		}
		return m, nil
	}
	if m.whichKey.IsPrefixKey(key) {
		var tickCmd tea.Cmd
		m.whichKey, tickCmd = m.whichKey.PressPrefix(key)
		return m, tickCmd
	}
	return m.handleFocusKey(msg)
}

// handleFocusKey is defined in main_focus_key.go (extracted for LOC compliance).
