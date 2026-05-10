package views

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// handleDeleteKey gestisce Y/N nel confirm dialog di cancellazione (batch-aware).
// ADR-009: N/Esc preserva S (l'utente può ritentare o cambiare selezione).
func (m ConversationModel) handleDeleteKey(msg tea.KeyMsg) (ConversationModel, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		return m.confirmDelete()
	case "n", "N", "esc":
		// Cancel: chiude overlay; S preservato per permettere retry/modifica.
		m.deleteMode = false
		m.deleteMsgIDs = nil
	}
	return m, nil
}

// confirmDelete rimuove i messaggi dal viewport e avvia la batch RPC.
// SOURCE_SNAPSHOT invariant: deleteMsgIDs è lo snapshot immutabile catturato
// in beginBatchDelete; mutazioni concorrenti a selection non influenzano il payload.
// BATCH_ATOMICITY: una sola BatchDeleteRequestMsg → una sola RPC in main.
func (m ConversationModel) confirmDelete() (ConversationModel, tea.Cmd) {
	ids := m.deleteMsgIDs
	chat := m.chat
	// Costruisce set per rimozione O(1).
	idSet := make(map[int]struct{}, len(ids))
	for _, id := range ids {
		idSet[id] = struct{}{}
	}
	// Rimuove ottimisticamente i messaggi dal modello.
	kept := m.messages[:0]
	for _, msg := range m.messages {
		if _, del := idSet[msg.ID]; !del {
			kept = append(kept, msg)
		}
	}
	m.messages = kept
	m.deleteMode = false
	m.deleteMsgIDs = nil
	// MODE_COHERENCE: clear selection → torna a BrowsingMessages.
	m.selection = make(map[int]struct{})
	m.multiSelect = false
	if m.cursor >= len(m.messages) {
		m.cursor = max(0, len(m.messages)-1)
	}
	m.viewport.SetContent(m.renderMessages())
	return m, func() tea.Msg {
		return telegram.BatchDeleteRequestMsg{Chat: chat, MsgIDs: ids}
	}
}
