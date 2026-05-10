package views

import "github.com/charmbracelet/lipgloss"

// View renders the main layout. Overlay priority (highest first):
//  1. Command palette — full-screen modal (Step 28)
//  2. Help overlay   — full-screen modal (Step 28)
//  3. Search overlay — full-screen modal (Step 26)
//  4. Chat info      — compact right-anchored overlay (Step 29), drawn over base
//  5. Which-key      — compact bottom-right overlay (Step 28), drawn over base
//  6. Base layout    — [folders|] chatList | conversation + statusBar
//
// Mutual exclusion enforced in handleKeyMsg via anyOverlayActive (ADR-015 §D3).
func (m MainModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	base := m.renderBase()
	// Step 34: tutte le modali via Modal.RenderOverlay → bg sempre visibile.
	if m.cmdPalette.Active {
		return overlayPalette(base, m.cmdPalette, m.Width)
	}
	if m.help.Active {
		return overlayHelp(base, m.help, m.Width)
	}
	if m.search.Active {
		return overlaySearch(base, m.search, m.Width)
	}
	if m.chatInfo.Active {
		return m.chatInfo.View()
	}
	if m.whichKey.IsVisible() {
		return overlayWhichKey(base, m.whichKey, m.Width)
	}
	return base
}

// renderBase composes the panel layout + status bar.
// Step 29: optionally prepends the folder sidebar as a third panel.
// Step 30: in Compact mode renders a single panel (compactVisible) full-width.
// Step 33: status bar dual-slot via renderStatusBar() (main_status_bar.go).
func (m MainModel) renderBase() string {
	bottom := m.renderStatusBar()
	if m.Notify.Active() {
		bottom = m.Notify.View(m.Width)
	}
	if m.layoutMode == LayoutCompact {
		return lipgloss.JoinVertical(lipgloss.Left, m.renderCompactPanel(), bottom)
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.renderWideRow(), bottom)
}

// renderWideRow composes the multi-panel horizontal row (Wide mode).
func (m MainModel) renderWideRow() string {
	var panels []string
	if m.folderModel.IsVisible() {
		panels = append(panels, m.folderModel.View())
	}
	panels = append(panels, m.chatList.View(), m.conversation.View())
	return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}

// renderCompactPanel renders the single visible panel in Compact mode.
// COMPACT_ONE_PANEL invariant: exactly one panel is rendered (ADR-018 §D2).
func (m MainModel) renderCompactPanel() string {
	if m.compactVisible == CompactConversation {
		return m.conversation.View()
	}
	return m.chatList.View()
}
