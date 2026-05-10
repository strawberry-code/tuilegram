package views

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// newSearchInput crea il textinput per la barra search in chat.
func newSearchInput() textinput.Model {
	ti := textinput.New()
	ti.Placeholder = "search..."
	ti.CharLimit = 256
	return ti
}

// handleSearchOpen apre la barra: build index, reset state, focus input.
// Chiama SetSize per ridurre il viewport height di searchBarHeight (MAJOR fix).
func (m ConversationModel) handleSearchOpen() (ConversationModel, tea.Cmd) {
	if m.searchBar.Active {
		return m, nil // Ctrl+F ignorato se barra già aperta
	}
	returnTo := convBrowsingMessages
	if m.multiSelect {
		returnTo = convMultiSelect
	}
	m.searchBar = SearchInChatState{
		Active:   true,
		Index:    buildSearchIndex(m.messages),
		ReturnTo: returnTo,
	}
	m.searchInput = newSearchInput()
	m.SetSize(m.Width, m.Height) // ricalcola viewport height con barra attiva
	return m, m.searchInput.Focus()
}

// handleSearchClose chiude la barra e ripristina lo stato precedente.
// Chiama SetSize per ripristinare il viewport height completo (MAJOR fix).
func (m ConversationModel) handleSearchClose() (ConversationModel, tea.Cmd) {
	m.searchBar = SearchInChatState{}
	m.searchInput.Blur()
	m.SetSize(m.Width, m.Height) // ripristina viewport height senza barra
	m.viewport.SetContent(m.renderMessages())
	return m, nil
}

// handleSearchQueryChanged ricalcola i match in modo sincrono (ADR-014 §D2).
func (m ConversationModel) handleSearchQueryChanged(
	q string,
) (ConversationModel, tea.Cmd) {
	m.searchBar.Query = q
	if q == "" {
		m.searchBar.Matches = nil
		m.searchBar.CurrentIdx = 0
		m.viewport.SetContent(m.renderMessages())
		return m, nil
	}
	qLC := strings.ToLower(q)
	m.searchBar.Matches = computeMatches(m.searchBar.Index, qLC)
	m.searchBar.CurrentIdx = 0
	m = m.scrollToCurrentMatch()
	m.viewport.SetContent(m.renderMessages())
	return m, nil
}

// handleSearchNext avanza al match successivo (wrap-around).
// Re-render esplicito dopo lo scroll: necessario per spostare l'highlight
// bold sul nuovo currentIdx (senza questo il counter cambia ma il viewport
// continua a mostrare il vecchio match evidenziato — blocker da code review).
func (m ConversationModel) handleSearchNext() (ConversationModel, tea.Cmd) {
	n := len(m.searchBar.Matches)
	if n == 0 {
		return m, nil
	}
	m.searchBar.CurrentIdx = (m.searchBar.CurrentIdx + 1) % n
	m = m.scrollToCurrentMatch()
	m.viewport.SetContent(m.renderMessages())
	return m, nil
}

// handleSearchPrev torna al match precedente (wrap-around).
// Stesso re-render esplicito di handleSearchNext.
func (m ConversationModel) handleSearchPrev() (ConversationModel, tea.Cmd) {
	n := len(m.searchBar.Matches)
	if n == 0 {
		return m, nil
	}
	m.searchBar.CurrentIdx = (m.searchBar.CurrentIdx - 1 + n) % n
	m = m.scrollToCurrentMatch()
	m.viewport.SetContent(m.renderMessages())
	return m, nil
}

// scrollToCurrentMatch aggiorna cursor al Pos del match corrente e restituisce
// il modello aggiornato. Value receiver per coerenza con gli altri handler.
func (m ConversationModel) scrollToCurrentMatch() ConversationModel {
	if len(m.searchBar.Matches) == 0 {
		return m
	}
	cur := m.searchBar.Matches[m.searchBar.CurrentIdx]
	for _, im := range m.searchBar.Index {
		if im.MsgID == cur.MsgID {
			m.cursor = im.Pos
			return m
		}
	}
	return m
}
