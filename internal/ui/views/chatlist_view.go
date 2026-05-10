package views

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

func (m ChatListModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}

	header := m.renderTabBar()
	itemWidth := m.Width - 2
	visible := m.visibleItems()

	rows := make([]string, 0, visible+1)
	rows = append(rows, header)
	for i := m.offset; i < len(m.chats) && i < m.offset+visible; i++ {
		rows = append(rows, m.renderItem(i, itemWidth))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, rows...)

	return lipgloss.NewStyle().
		Width(m.Width).Height(m.Height).
		Border(lipgloss.NormalBorder(), false, true, false, false).
		BorderForeground(styles.ColorBorder()).
		Render(content)
}

// renderTabBar disegna i tab orizzontali (Step 34, ADR-022) — pill-style
// con il tab attivo evidenziato. Folder source = m.folders (popolato da
// SetFolders). Quando vuoto fallback al vecchio header "● CHATS".
func (m ChatListModel) renderTabBar() string {
	if len(m.folders) == 0 {
		return lipgloss.NewStyle().Bold(true).Foreground(styles.ColorPrimary()).PaddingLeft(1).Render(m.connDot() + " CHATS")
	}
	active := lipgloss.NewStyle().Foreground(styles.ColorButtonFg()).Background(styles.ColorPrimary()).Bold(true).Padding(0, 1)
	idle := lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Padding(0, 1)
	parts := []string{m.connDot()}
	for _, f := range m.folders {
		if f.ID == m.activeFolderID {
			parts = append(parts, active.Render(f.Title))
		} else {
			parts = append(parts, idle.Render(f.Title))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, parts...)
	return lipgloss.NewStyle().MaxHeight(1).MaxWidth(m.Width - 1).Render(bar)
}

func (m ChatListModel) renderItem(idx, width int) string {
	chat := m.chats[idx]
	selected := idx == m.selected

	dotOnline := lipgloss.NewStyle().Foreground(styles.ColorSuccess()).Render("●")
	dotUnread := lipgloss.NewStyle().Foreground(styles.ColorIncoming()).Render("●")
	dots := "   "
	if chat.IsOnline && chat.HasUnread() {
		dots = dotOnline + " " + dotUnread
	} else if chat.IsOnline {
		dots = dotOnline + "  "
	} else if chat.HasUnread() {
		dots = "  " + dotUnread
	}

	nameFg := styles.ColorText()
	if chat.IsMuted {
		nameFg = styles.ColorTextDim()
	}
	name := lipgloss.NewStyle().Foreground(nameFg).Render(chat.Title)

	mute := ""
	if chat.IsMuted {
		mute = lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render(" 🔇")
	}

	_, isTyping := m.typingPeers[chat.ID]
	typingMark := ""
	if isTyping {
		typingMark = lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Italic(true).Render(" ✎ typing")
	}

	// Step 34: truncate inline (no wrap) → ogni item = 1 riga garantita.
	// Padding 1+1 → contenuto utile = width-2.
	content := ansi.Truncate(dots+" "+name+mute+typingMark, width-2, "…")
	row := lipgloss.NewStyle().PaddingLeft(1).PaddingRight(1).Width(width)
	if selected {
		row = row.Background(styles.ColorSearchInlineBg()).Foreground(styles.ColorPrimary())
	}
	return row.Render(content)
}

func (m ChatListModel) connDot() string {
	switch m.Conn {
	case ConnConnected:
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess()).Render("●")
	case ConnReconnecting:
		return lipgloss.NewStyle().Foreground(styles.ColorWarning()).Render("○")
	case ConnDisconnected:
		return lipgloss.NewStyle().Foreground(styles.ColorError()).Render("✕")
	default:
		return lipgloss.NewStyle().Foreground(styles.ColorSuccess()).Render("●")
	}
}
