package telegram

import "github.com/strawberry-code/tuilegram/internal/model"

// SelectToggleMsg è emesso quando l'utente preme Space su un messaggio.
// Toggling il MessageID nel set di selezione S.
// Se S passa da ∅ a ≠∅ → enter MultiSelect; se da ≠∅ a ∅ → exit.
type SelectToggleMsg struct {
	MsgID int
}

// SelectClearMsg è emesso quando l'utente preme Esc in MultiSelect.
// Svuota S e ritorna a BrowsingMessages.
type SelectClearMsg struct{}

// BatchActionDoneMsg è emesso dopo un'azione batch completata con successo
// (ForwardResultMsg{ok} o DeleteResultMsg{ok} in multiselect).
// Svuota S e riporta in BrowsingMessages.
type BatchActionDoneMsg struct{}

// BatchDeleteRequestMsg è emesso quando D viene premuto in MultiSelect
// oppure su un singolo messaggio (fallback su cursore).
// Sostituisce il vecchio per-message DeleteRequestMsg nel batch path.
type BatchDeleteRequestMsg struct {
	Chat     model.Chat
	MsgIDs   []int
	Messages []model.Message // snapshot immutabile al momento dell'apertura
}
