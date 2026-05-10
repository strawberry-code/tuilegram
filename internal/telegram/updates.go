package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
)

// setupUpdates registra gli handler per gli aggiornamenti real-time.
func (b *Bridge) setupUpdates(dispatcher *tg.UpdateDispatcher) {
	dispatcher.OnNewMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewMessage) error {
		return b.handleIncomingMsg(u.Message, e)
	})
	dispatcher.OnNewChannelMessage(func(ctx context.Context, e tg.Entities, u *tg.UpdateNewChannelMessage) error {
		return b.handleIncomingMsg(u.Message, e)
	})
	// Step 23: typing indicator per chat private (1:1).
	dispatcher.OnUserTyping(func(ctx context.Context, e tg.Entities, u *tg.UpdateUserTyping) error {
		return b.handleUserTyping(u)
	})
	// Step 25: reactions snapshot update.
	dispatcher.OnMessageReactions(func(ctx context.Context, e tg.Entities, u *tg.UpdateMessageReactions) error {
		return b.handleReactionsUpdate(u)
	})
}

func (b *Bridge) handleIncomingMsg(msgClass tg.MessageClass, e tg.Entities) error {
	if b.OnNewMessage == nil {
		return nil
	}
	names := buildNamesFromUserMap(e.Users)
	switch m := msgClass.(type) {
	case *tg.Message:
		if m.Message == "" && m.Media == nil {
			return nil
		}
		b.OnNewMessage(toMessage(m, names, ""), chatIDFromPeer(m.PeerID))
	case *tg.MessageService:
		// Step 25: service messages forwarded as IsService=true.
		b.OnNewMessage(toServiceMessage(m, names), chatIDFromPeer(m.PeerID))
	}
	return nil
}

// handleReactionsUpdate processes UpdateMessageReactions from the dispatcher.
// Snapshot is converted and delivered to the TUI via OnReactionsUpdated.
// Viewport-scoped cache (sequence diagram §6): TUI discards if chatID != active.
func (b *Bridge) handleReactionsUpdate(u *tg.UpdateMessageReactions) error {
	if b.OnReactionsUpdated == nil {
		return nil
	}
	reactions := convert.FlattenReactions(u.Reactions)
	b.OnReactionsUpdated(chatIDFromPeer(u.Peer), u.MsgID, reactions)
	return nil
}

// handleUserTyping gestisce updateUserTyping per chat private (1:1).
// Filtra a SendMessageTypingAction; ignora upload/record/ecc (fuori scope Step 23).
func (b *Bridge) handleUserTyping(u *tg.UpdateUserTyping) error {
	if b.OnUserTyping == nil {
		return nil
	}
	if _, ok := u.Action.(*tg.SendMessageTypingAction); !ok {
		return nil
	}
	peer := model.ChatID{PeerType: model.PeerUser, ID: u.UserID}
	b.OnUserTyping(peer, u.UserID)
	return nil
}
