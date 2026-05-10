package views

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// searchBarHeight è l'altezza in righe della barra inline: 1 riga contenuto
// + 1 bordo top (searchBarStyle usa BorderTop). Usato da SetSize.
const searchBarHeight = 2

// SetSize aggiorna le dimensioni del viewport e della textarea.
// Invariante PINNED_OFFSET_RESERVED: viewport.Height -= pinnedBarHeight se pinnedMsg != nil.
// Quando la search bar è attiva riduce il viewport di searchBarHeight (ADR-014 §D1).
func (m *ConversationModel) SetSize(w, h int) {
	m.Width = w
	m.Height = h
	vpH := h - 2 - inputAreaHeight
	if m.pinnedMsg != nil {
		vpH -= pinnedBarHeight
	}
	if m.searchBar.Active {
		vpH -= searchBarHeight
	}
	m.viewport.Width = w - 2
	m.viewport.Height = vpH
	m.textarea.SetWidth(w - 12)
	// Step 34: re-render messages on resize → bubble width (60% di viewport)
	// si ricalcola, wrap a runtime allineato al nuovo viewport.Width.
	if m.active && len(m.messages) > 0 {
		m.viewport.SetContent(m.renderMessages())
	}
}

func (m ConversationModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	if !m.active {
		return m.viewEmpty()
	}
	if m.loading {
		return m.viewLoading()
	}
	return m.viewMessages()
}

// viewMessages compone il layout verticale: header + [pinnedBar] + viewport + [searchBar] + input.
// Pinned bar: 2 righe se pinnedMsg != nil (ADR-021 §A4, PINNED_OFFSET_RESERVED).
// Search bar: 2 righe se searchBar.Active (ADR-014 §D1).
func (m ConversationModel) viewMessages() string {
	if m.forwardPicker.Active() {
		return m.forwardPicker.View(m.Width, m.Height)
	}
	if m.editMode {
		return m.renderEditOverlay()
	}
	if m.deleteMode {
		return m.renderDeleteOverlay()
	}

	header := m.renderHeader()
	if m.replyTo != nil { // dynamic vpH carve-out for reply bar
		m.viewport.Height -= 1
	}
	msgs := m.viewport.View()
	input := m.renderInputArea()

	parts := []string{header}
	if bar := m.pinnedBar(); bar != "" {
		parts = append(parts, bar)
	}
	parts = append(parts, msgs)
	if m.searchBar.Active {
		parts = append(parts, m.renderSearchBar())
	}
	parts = append(parts, input)
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return lipgloss.NewStyle().Width(m.Width).Height(m.Height).Render(body)
}

func (m ConversationModel) renderInputArea() string {
	row := lipgloss.JoinHorizontal(lipgloss.Center,
		m.textarea.View(), "  ", m.sendBtn.View(),
	)
	body := row
	if m.replyTo != nil {
		preview := truncate(m.replyTo.Text, 40)
		body = replyBarStyle().Render("↩ "+m.replyTo.SenderName+": "+preview) + "\n" + row
	}
	return inputBorderStyle().Width(m.Width - 2).Render(body)
}

func (m ConversationModel) viewLoading() string {
	content := lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render(m.spinner.View() + " Loading...")
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
}

func (m ConversationModel) viewEmpty() string {
	from, to := styles.GradientColors()
	logo := styles.RenderGradient("T u i l e g r a m", from, to)
	hint := lipgloss.NewStyle().Foreground(styles.ColorTextDim()).MarginTop(2).Render("Select a chat")
	content := lipgloss.JoinVertical(lipgloss.Center, logo, hint)
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, content)
}
