package telegram

import "github.com/strawberry-code/tuilegram/internal/model"

// BridgeCallbacks raccoglie i callback con cui il Bridge notifica il TUI.
// Sono campi pubblici impostati dal chiamante prima di Bridge.Start().
// Tutti i callback vengono invocati dalla goroutine Telegram — il TUI li
// riceve attraverso program.Send() e risponde nell'Update() single-threaded.
type BridgeCallbacks struct {
	// OnConnected è chiamato quando il client è autenticato e connesso.
	OnConnected func()

	// OnDisconnected è chiamato alla perdita di connessione non richiesta.
	OnDisconnected func(error)

	// OnReconnecting è chiamato a ogni tentativo di riconnessione automatica.
	OnReconnecting func()

	// OnAuthRequired è chiamato se la sessione non è valida o assente.
	OnAuthRequired func()

	// OnNewMessage è chiamato alla ricezione di un nuovo messaggio real-time.
	OnNewMessage func(msg model.Message, chatID model.ChatID)

	// OnUserTyping è chiamato quando un peer scrive in una chat privata (1:1).
	// peer.PeerType è sempre PeerUser; userID == peer.ID.
	// Il TUI applica ADR-010 (timestamp + re-arm tea.Tick) per il TTL 5s.
	OnUserTyping func(peer model.ChatID, userID int64)

	// OnReactionsUpdated è chiamato quando il server emette UpdateMessageReactions.
	// reactions è uno snapshot completo già ordinato (Count desc, Emoji asc).
	// Il TUI sostituisce m.Reactions interamente (replace semantics — ADR-012 A1).
	OnReactionsUpdated func(chatID model.ChatID, msgID int, reactions []model.Reaction)
}
