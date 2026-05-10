package telegram

import "github.com/strawberry-code/tuilegram/internal/model"

// ForwardRequestMsg è emesso dalla UI quando l'utente preme 'f' su un messaggio.
// Messages è una slice per riuso in Step 22 (multi-forward); Step 21 invia sempre 1 elemento.
type ForwardRequestMsg struct {
	Source   model.Chat
	Messages []model.Message
}

// ForwardPickerReadyMsg consegna lo snapshot dei dialogs al picker overlay.
type ForwardPickerReadyMsg struct {
	Chats []model.Chat
}

// ForwardSubmitMsg è emesso dal picker quando l'utente preme Enter sulla chat target.
type ForwardSubmitMsg struct {
	Target   model.Chat
	Source   model.Chat
	Messages []model.Message
}

// ForwardResultMsg riporta l'esito dell'RPC di forward.
type ForwardResultMsg struct {
	Target model.Chat
	Err    error
}

// OverlayCloseMsg chiude qualsiasi overlay attivo (picker Esc, confirm dismiss, ecc).
type OverlayCloseMsg struct{}
