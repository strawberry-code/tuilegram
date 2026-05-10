package views

import "github.com/strawberry-code/tuilegram/internal/telegram"

// handleReactionsUpdated applica uno snapshot di reactions a un messaggio nel viewport.
//
// Semantica viewport-scoped (ADR-012 §6, reactions-flow.md §6):
//   - ChatID != chat attiva → scarta silenziosamente (no cache cross-chat in Step 25).
//   - MessageID non trovato in cache → scarta silenziosamente (messaggio non caricato).
//   - Trovato → replace completo di m.Reactions (snapshot, non merge) + re-render.
//
// SYSTEM_NO_REACT (reactions.tla): il render ignora reactions su IsService=true
// anche se aggiornate qui (il rendering branch SystemBranch salta la reactions row).
func (m ConversationModel) handleReactionsUpdated(msg telegram.ReactionsUpdatedMsg) ConversationModel {
	// Viewport-scoped: scarta update di chat non attive.
	if !m.active || m.chat.ID != msg.ChatID {
		return m
	}
	// Locate message by ID — O(N) scan; tipicamente <500 msg (reactions-flow.md §1).
	found := false
	for i := range m.messages {
		if m.messages[i].ID == msg.MessageID {
			m.messages[i].Reactions = msg.Reactions
			found = true
			break
		}
	}
	if !found {
		// Messaggio non presente nel viewport (scrolled out / non caricato): no-op.
		return m
	}
	m.viewport.SetContent(m.renderMessages())
	return m
}
