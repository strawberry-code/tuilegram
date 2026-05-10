package views

import tea "github.com/charmbracelet/bubbletea"

// Update processes folder-sidebar messages. Returns the mutated model and nil
// (folder operations are all sync, no tea.Cmd spawned — invariant NO_RPC_FOLDER).
func (m FolderModel) Update(msg tea.Msg) (FolderModel, tea.Cmd) {
	switch msg := msg.(type) {
	case FolderToggleMsg:
		return m.handleToggle()
	case FolderCursorMsg:
		return m.handleCursor(msg)
	case FolderSelectMsg:
		return m.handleSelect(msg)
	case tea.KeyMsg:
		if m.visible && m.focus == folderFocusBrowsing {
			return m.handleKey(msg)
		}
	}
	return m, nil
}

// handleToggle flips sidebar visibility. Focus moves to sidebar on open,
// to background on close. selectedID is ALWAYS preserved (ADR-016 §D3).
func (m FolderModel) handleToggle() (FolderModel, tea.Cmd) {
	if m.visible {
		m.visible = false
		// Focus will be restored by the root model to chatList.
	} else {
		m.visible = true
		m.focus = folderFocusBrowsing
		// Restore cursor to the currently selected folder.
		m.cursor = m.indexOfSelected()
	}
	return m, nil
}

// handleCursor moves the selection highlight by delta, clamped.
func (m FolderModel) handleCursor(msg FolderCursorMsg) (FolderModel, tea.Cmd) {
	if !m.visible || m.focus != folderFocusBrowsing {
		return m, nil
	}
	m.cursor += msg.Delta
	m.clampCursor()
	return m, nil
}

// handleSelect applies the folder filter. Called by Enter/l in the sidebar.
// Returns a FolderSelectMsg so the root model can re-filter the chat list.
func (m FolderModel) handleSelect(msg FolderSelectMsg) (FolderModel, tea.Cmd) {
	m.selectedID = msg.FolderID
	// Sync cursor to the newly selected folder.
	m.cursor = m.indexOfSelected()
	return m, nil
}

// handleKey routes sidebar-specific keys when the sidebar has focus.
// Tab/h/Esc give focus back to the chat list (sidebar stays visible).
func (m FolderModel) handleKey(msg tea.KeyMsg) (FolderModel, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		m.cursor++
		m.clampCursor()
	case "k", "up":
		m.cursor--
		m.clampCursor()
	case "enter", "l":
		f := m.cursorFolder()
		m.selectedID = f.ID
		// Emit FolderSelectMsg so root re-filters chatList synchronously.
		return m, func() tea.Msg { return FolderSelectMsg{FolderID: f.ID} }
	case "tab", "h", "esc":
		// Give focus back to the chat list; sidebar remains visible.
		m.focus = folderFocusBackground
	}
	return m, nil
}

// indexOfSelected returns the cursor index for the currently selected folder.
func (m FolderModel) indexOfSelected() int {
	for i, f := range m.allFolders {
		if f.ID == m.selectedID {
			return i
		}
	}
	return 0 // fallback to sentinel
}

// GiveFocusBack transitions Background → Browsing (Shift+Tab from chatList).
func (m *FolderModel) GiveFocusBack() {
	if m.visible {
		m.focus = folderFocusBrowsing
	}
}

// LoseFocus transitions Browsing → Background (Tab/h/Esc from sidebar).
func (m *FolderModel) LoseFocus() {
	m.focus = folderFocusBackground
}

// ForceHide closes the sidebar unconditionally (no tea.Msg emitted).
// Called on Wide→Compact cross-threshold (ADR-018 §D4 + ADR-016 §D5).
// selectedID is preserved: folder filter remains active across layout flip.
func (m *FolderModel) ForceHide() {
	m.visible = false
	m.focus = folderFocusBackground
}
