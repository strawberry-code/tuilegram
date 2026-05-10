package views

import (
	"strings"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// reindexNewMessage aggiunge un messaggio all'indice se search attiva.
// CurrentIdx invariato (append in coda, identità preservata — ADR-014 §D2).
func (m *ConversationModel) reindexNewMessage(msg model.Message) {
	if !m.searchBar.Active {
		return
	}
	pos := len(m.messages) - 1
	m.searchBar.Index = appendIndexEntry(m.searchBar.Index, msg, pos)
	if m.searchBar.Query == "" || msg.IsService || msg.Text == "" {
		return
	}
	qLC := strings.ToLower(m.searchBar.Query)
	spans := allOccurrences(strings.ToLower(msg.Text), qLC)
	if len(spans) > 0 {
		m.searchBar.Matches = append(m.searchBar.Matches,
			SearchMatch{MsgID: msg.ID, Spans: spans})
	}
}
