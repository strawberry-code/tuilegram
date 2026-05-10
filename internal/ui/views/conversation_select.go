package views

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// toggleSelection toggles cursor's message in S (TLA+: SpaceToggle).
// MODE_COHERENCE maintained: multiSelect = (len(selection) > 0).
func (m ConversationModel) toggleSelection() ConversationModel {
	if m.cursor < 0 || m.cursor >= len(m.messages) {
		return m
	}
	id := m.messages[m.cursor].ID
	if _, ok := m.selection[id]; ok {
		delete(m.selection, id)
	} else {
		m.selection[id] = struct{}{}
	}
	m.multiSelect = len(m.selection) > 0
	return m
}

// clearSelection svuota S e esce da MultiSelect (TLA+: EscClear).
func (m ConversationModel) clearSelection() ConversationModel {
	m.selection = make(map[int]struct{})
	m.multiSelect = false
	return m
}

// selectionSnapshot ritorna una copia immutabile di S come slice di Message.
// SOURCE_SNAPSHOT: il caller usa questo slice per l'RPC; mutazioni successive
// a selection non influenzano il payload.
func (m ConversationModel) selectionSnapshot() []model.Message {
	snap := make([]model.Message, 0, len(m.selection))
	for _, msg := range m.messages {
		if _, ok := m.selection[msg.ID]; ok {
			snap = append(snap, msg)
		}
	}
	return snap
}

// beginBatchForward avvia il forward picker con la selezione corrente (o cursore).
// Fallback su {cursor} se S=∅ (TLA+: OpenForward).
func (m ConversationModel) beginBatchForward() (ConversationModel, tea.Cmd) {
	var msgs []model.Message
	if m.multiSelect {
		msgs = m.selectionSnapshot() // SOURCE_SNAPSHOT: copia immutabile
	} else {
		if m.cursor < 0 || m.cursor >= len(m.messages) {
			return m, nil
		}
		msgs = []model.Message{m.messages[m.cursor]}
	}
	m.forwardSource = msgs
	src := m.chat
	return m, func() tea.Msg {
		return telegram.ForwardRequestMsg{Source: src, Messages: msgs}
	}
}

// beginBatchDelete apre il confirm dialog con la selezione corrente (o cursore).
// Fallback su {cursor} se S=∅ (TLA+: OpenDelete).
func (m ConversationModel) beginBatchDelete() (ConversationModel, tea.Cmd) {
	var ids []int
	if m.multiSelect {
		snap := m.selectionSnapshot()
		ids = make([]int, len(snap))
		for i, msg := range snap {
			ids[i] = msg.ID
		}
	} else {
		if m.cursor < 0 || m.cursor >= len(m.messages) {
			return m, nil
		}
		// Step 20 compat: solo messaggi propri nel fallback single.
		if !m.messages[m.cursor].IsOutgoing {
			return m, nil
		}
		ids = []int{m.messages[m.cursor].ID}
	}
	m.deleteMsgIDs = ids
	m.deleteMode = true
	return m, nil
}
