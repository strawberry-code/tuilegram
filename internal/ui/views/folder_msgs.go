package views

// folder_msgs.go — tea.Msg types for the folder sidebar (Step 29).
// Extends message-taxonomy.md §Internal UI Messages.

// FolderToggleMsg is dispatched by the global 'F' key handler.
// Flips folderSidebarVisible; selectedFolderID is always preserved (ADR-016 §D3).
type FolderToggleMsg struct{}

// FolderCursorMsg is dispatched by j/k/↑/↓ while the sidebar has focus.
// Delta is +1 (down) or -1 (up). The receiver clamps to [0, len(folders)-1].
type FolderCursorMsg struct{ Delta int }

// FolderSelectMsg is dispatched by Enter/l on the highlighted folder item.
// FolderID == 0 selects the "All Chats" sentinel (no filter).
type FolderSelectMsg struct{ FolderID int }
