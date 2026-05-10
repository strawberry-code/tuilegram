package telegram

import "github.com/strawberry-code/tuilegram/internal/model"

// Eventi inviati dal Bridge al TUI loop via tea.Program.Send().

// ConnectedMsg indica che il client è connesso e autenticato.
type ConnectedMsg struct{}

// DisconnectedMsg indica che la connessione è stata persa.
type DisconnectedMsg struct{ Err error }

// ReconnectingMsg indica tentativo di reconnessione.
type ReconnectingMsg struct{}

// AuthRequiredMsg indica che serve autenticazione.
type AuthRequiredMsg struct{}

// CodeSentMsg indica che il codice è stato inviato con successo.
type CodeSentMsg struct {
	CodeHash   string
	CodeLength int
}

// CodeSentErrMsg indica errore nell'invio del codice.
type CodeSentErrMsg struct{ Err error }

// SignInOkMsg indica login completato con successo.
type SignInOkMsg struct{}

// SignInErrMsg indica errore nel login (codice sbagliato, ecc).
type SignInErrMsg struct{ Err error }

// PasswordRequiredMsg indica che serve la password 2FA.
type PasswordRequiredMsg struct{}

// PasswordOkMsg indica password 2FA accettata.
type PasswordOkMsg struct{}

// PasswordErrMsg indica password 2FA sbagliata.
type PasswordErrMsg struct{ Err error }

// DialogsLoadedMsg contiene la lista delle chat caricate.
type DialogsLoadedMsg struct{ Chats []model.Chat }

// DialogsErrMsg indica errore nel caricamento dei dialogs.
type DialogsErrMsg struct{ Err error }

// MessagesLoadedMsg contiene i messaggi caricati da una chat.
type MessagesLoadedMsg struct{ Messages []model.Message }

// MessagesErrMsg indica errore nel caricamento dei messaggi.
type MessagesErrMsg struct{ Err error }

// SendRequestMsg indica che l'utente vuole inviare un messaggio.
type SendRequestMsg struct {
	Chat      model.Chat
	Text      string
	ReplyToID int
}

// MessageSentMsg indica che il messaggio è stato inviato con successo.
type MessageSentMsg struct{}

// MessageSentErrMsg indica errore nell'invio del messaggio.
type MessageSentErrMsg struct{ Err error }

// NewMessageMsg indica che è arrivato un nuovo messaggio real-time.
type NewMessageMsg struct {
	Message model.Message
	ChatID  model.ChatID
}

// EditRequestMsg indica che l'utente vuole modificare un messaggio.
type EditRequestMsg struct {
	Chat  model.Chat
	MsgID int
	Text  string
}

// DeleteRequestMsg indica che l'utente vuole cancellare un messaggio.
type DeleteRequestMsg struct {
	Chat  model.Chat
	MsgID int
}

// UpdateUserTypingMsg è inviato al TUI quando un peer sta scrivendo in una
// chat privata (1:1). Corrisponde all'MTProto updateUserTyping.
// Il TUI aggiorna typing[Peer].lastTypingAt e schedula un tea.Tick(5s) per
// il cleanup TTL (ADR-010: timestamp-based + re-arm).
type UpdateUserTypingMsg struct {
	// Peer è il ChatID della chat privata (PeerType=PeerUser, ID=UserID).
	Peer model.ChatID
	// UserID è l'ID dell'utente che sta scrivendo (= Peer.ID per chat 1:1).
	UserID int64
}

// ReactionsUpdatedMsg è inviato al TUI quando il server emette un
// UpdateMessageReactions per un messaggio esistente (Step 25).
// Reactions è uno snapshot completo già ordinato (Count desc, Emoji asc)
// dal convert layer — il TUI sostituisce m.Reactions interamente (replace, non merge).
// Viewport-scoped cache (ADR-012 §6): il TUI scarta questo msg se
// ChatID != chat attiva; la prossima apertura ricaricherà la history fresca.
type ReactionsUpdatedMsg struct {
	ChatID    model.ChatID
	MessageID int
	Reactions []model.Reaction
}
