package views

import tea "github.com/charmbracelet/bubbletea"

// main_compact_handlers.go — Step 30: compact mode key/size handlers
// and the cross-threshold side-effect logic (ADR-018).
// Separated from main_key_handlers.go and main_update.go to keep LOC ≤ 120.

// handleLayoutPanelSwitch toggles compactVisible between ChatList and
// Conversation (LayoutPanelSwitchMsg handler, ADR-018 §D3).
// Invariant TAB_PRESERVES_LAYOUT: layoutMode is NOT mutated here.
// activeChatID (conversation.active) is invariant (ADR-016 §D4).
func (m MainModel) handleLayoutPanelSwitch() (MainModel, tea.Cmd) {
	if m.compactVisible == CompactChatList {
		m.compactVisible = CompactConversation
	} else {
		m.compactVisible = CompactChatList
	}
	// Sync sub-model sizes: compact panels occupy full width.
	m.applyCompactSizes()
	m.recomputeBboxes() // Step 32: panel switch changes which bboxes are non-nil
	return m, nil
}

// handleCompactEsc handles Esc in Compact.ShowingConversation: switches back
// to ShowingChatList while preserving activeChatID (statechart §Compact).
// Note: Esc in Compact.ShowingChatList falls through to normal chatList handling.
func (m MainModel) handleCompactEsc() (MainModel, tea.Cmd) {
	m.compactVisible = CompactChatList
	m.applyCompactSizes()
	m.recomputeBboxes() // Step 32: visible panel changed → bboxes must reflect it
	return m, nil
}

// handleCompactEnter opens the selected chat AND switches compactVisible to
// Conversation atomically (statechart §Compact: "Enter on chat →
// compactVisible := Conversation" — single Update cycle, ADR-018 §D4).
func (m MainModel) handleCompactEnter() (MainModel, tea.Cmd) {
	chat, ok := m.chatList.SelectedChat()
	if !ok {
		return m, nil
	}
	spinnerCmd := m.conversation.OpenChat(chat)
	m.focus = FocusConversation
	m.compactVisible = CompactConversation
	m.applyCompactSizes()
	m.recomputeBboxes() // Step 32: conversation panel now visible
	// Step 33: carica anche il pinned message in background.
	return m, tea.Batch(spinnerCmd, m.LoadMessagesCmd(chat), m.LoadPinnedMessageCmd(chat))
}

// applyLayoutCrossThreshold executes the side-effects mandated by ADR-018 §D4
// when LayoutMode transitions between Wide and Compact.
//
// Wide → Compact:
//   - compactVisible derived from pre-collapse focus/chat state
//   - folderSidebarVisible forced to FALSE (auto-close, ADR-016 §D5)
//   - selectedFolderID preserved; activeOverlay preserved; activeChatID preserved
//
// Compact → Wide:
//   - compactVisible discarded (will be re-derived on next collapse)
//   - folderSidebarVisible unchanged (no auto-restore, ADR-018 §D4)
//   - all other state preserved
func (m *MainModel) applyLayoutCrossThreshold(prev, next LayoutMode) {
	if prev == LayoutWide && next == LayoutCompact {
		// Collapse: derive which panel to show; auto-close sidebar.
		m.compactVisible = deriveCompactVisible(m.focus, m.conversation.active)
		m.folderModel.ForceHide() // auto-close (ADR-018 §D4; ADR-016 §D5)
		// focus stays as-is; selectedFolderID preserved (inside folderModel)
	}
	// Expand (Compact → Wide): no destructive side-effect.
	// compactVisible is meaningless in Wide; it will be re-derived at next collapse.
}

// applyCompactSizes recalculates sub-model dimensions for Compact single-pane
// layout. Full width goes to whichever panel is compactVisible; the hidden
// panel is still sized (for when it becomes visible) but not rendered.
func (m *MainModel) applyCompactSizes() {
	available := m.Height - 1 // reserve 1 row for status bar
	m.chatList.Width = m.Width
	m.chatList.Height = available
	m.conversation.SetSize(m.Width, available)
	m.chatInfo.SetSize(m.Width, available)
	// Overlay full-width in compact.
	m.search.Width = m.Width
	m.search.Height = m.Height
	m.cmdPalette.Width = m.Width
	m.cmdPalette.Height = m.Height
	m.whichKey.Width = m.Width
	m.whichKey.Height = m.Height
	m.help.Width = m.Width
	m.help.Height = m.Height
}

// handleWindowSize processes tea.WindowSizeMsg: computes newMode, applies
// cross-threshold side-effects if the mode flips, then resizes all panels.
// Emits LayoutModeChangedMsg as a fanout when a flip occurs (ADR-018 §D2;
// same-half-plane resizes are idempotent — no msg, no state change).
// THRESHOLD_DETERMINISTIC: layoutMode' = f(width), pure function, no history.
func (m MainModel) handleWindowSize(w, h int) (MainModel, tea.Cmd) {
	newMode := computeLayoutMode(w, m.compactThreshold)
	var cmd tea.Cmd
	if newMode != m.layoutMode {
		prev := m.layoutMode
		m.layoutMode = newMode
		m.applyLayoutCrossThreshold(prev, newMode)
		changed := LayoutModeChangedMsg{OldMode: prev, NewMode: newMode}
		cmd = func() tea.Msg { return changed }
	}
	m.SetSize(w, h)
	return m, cmd
}
