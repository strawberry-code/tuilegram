package views

// main_status_hint.go — keymapHint: funzione pura focus-aware (ADR-021 §DD2).
// Invariante STATUSBAR_KEYMAP_DETERMINISTIC: stessi argomenti → stessa stringa.

// keymapHint restituisce la stringa di shortcuts rilevante per il focus corrente.
// È una funzione pura: nessuno stato, nessun side-effect (ADR-021 §DD2).
// Compact mode e overlay cambiano il contesto di conseguenza.
func (m MainModel) keymapHint() string {
	// Overlay attivi: mostrano i propri shortcuts.
	if m.cmdPalette.Active {
		return "↑/↓ select · Enter run · Esc cancel"
	}
	if m.help.Active {
		return "↑/↓ scroll · Esc close"
	}
	if m.search.Active {
		return "type to search · Enter jump · Esc close"
	}
	if m.chatInfo.Active {
		return "↑/↓ scroll · Esc close"
	}
	// Multi-select in conversation.
	if m.conversation.multiSelect {
		return "Space toggle · f forward · D delete · Esc cancel"
	}
	// Compact mode: shortcuts semplificati per larghezza ridotta.
	if m.layoutMode == LayoutCompact {
		return m.compactHint()
	}
	// Wide mode: shortcuts per il pannello con focus.
	switch m.focus {
	case FocusFolders:
		return "j/k navigate · Enter select · F close"
	case FocusConversation:
		if m.conversation.inputFocus {
			return "Enter send · Shift+Enter newline · Esc cancel"
		}
		return "j/k cursor · r reply · e edit · gx open link · Esc back"
	default: // FocusChatList
		return "j/k navigate · Enter open · / search · F folders · ? help"
	}
}

// compactHint restituisce shortcuts per compact mode in base al pannello visibile.
func (m MainModel) compactHint() string {
	if m.compactVisible == CompactConversation {
		if m.conversation.inputFocus {
			return "Enter send · Esc cancel"
		}
		return "j/k · r reply · Tab list · ? help"
	}
	return "j/k nav · Enter open · Tab conv · ? help"
}
