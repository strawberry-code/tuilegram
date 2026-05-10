package views

import "github.com/strawberry-code/tuilegram/internal/model"

// folderFocus tracks whether the sidebar has keyboard focus (Browsing)
// or is visible but unfocused (Background).
type folderFocus int

const (
	folderFocusBrowsing   folderFocus = iota // keyboard input goes to sidebar
	folderFocusBackground                    // sidebar visible, focus elsewhere
)

// FolderModel manages the folder sidebar panel (Step 29, ADR-016).
// This is NOT an overlay; it is an inline panel (ADR-016 §D2).
// Value-receiver pattern: root owns state via returned copies.
type FolderModel struct {
	// visible controls whether the sidebar is rendered (Hidden vs Visible states).
	visible bool
	// focus tracks Browsing vs Background sub-state.
	focus folderFocus
	// cursor is the index into allFolders of the highlighted row.
	// Range: [0, len(allFolders)-1].
	cursor int
	// selectedID is the currently applied folder filter.
	// 0 = "All Chats" sentinel (no filter). Preserved across toggle (ADR-016 §D3).
	selectedID int
	// allFolders is [sentinel, ...server folders]. Populated from DialogsLoadedMsg.
	// Invariant: allFolders[0] = AllChatsFolder (SENTINEL_PRESENT in TLA+).
	allFolders []model.ChatFolder
	// Width/Height are set by the root model on WindowSizeMsg.
	Width, Height int
}

// NewFolderModel returns a closed, zero-state FolderModel with sentinel present.
func NewFolderModel() FolderModel {
	return FolderModel{
		allFolders: []model.ChatFolder{model.AllChatsFolder},
	}
}

// IsVisible reports whether the sidebar should be rendered.
func (m FolderModel) IsVisible() bool { return m.visible }

// HasFocus reports whether the sidebar currently has keyboard focus.
func (m FolderModel) HasFocus() bool {
	return m.visible && m.focus == folderFocusBrowsing
}

// SelectedID returns the currently applied folder filter ID.
func (m FolderModel) SelectedID() int { return m.selectedID }

// SetFolders replaces the server-side folder list.
// The AllChats sentinel is always prepended (invariant SENTINEL_PRESENT).
func (m *FolderModel) SetFolders(folders []model.ChatFolder) {
	m.allFolders = make([]model.ChatFolder, 0, len(folders)+1)
	m.allFolders = append(m.allFolders, model.AllChatsFolder)
	for _, f := range folders {
		if f.ID != 0 { // skip if server accidentally sends ID=0
			m.allFolders = append(m.allFolders, f)
		}
	}
}

// cursorFolder returns the folder at the current cursor position.
func (m FolderModel) cursorFolder() model.ChatFolder {
	if m.cursor < 0 || m.cursor >= len(m.allFolders) {
		return model.AllChatsFolder
	}
	return m.allFolders[m.cursor]
}

// clampCursor ensures cursor stays within [0, len(allFolders)-1].
func (m *FolderModel) clampCursor() {
	maxIdx := len(m.allFolders) - 1
	if maxIdx < 0 {
		maxIdx = 0
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor > maxIdx {
		m.cursor = maxIdx
	}
}
