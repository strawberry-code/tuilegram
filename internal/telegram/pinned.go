package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// LoadPinnedMessage recupera il messaggio pinnato di una chat.
// Strategia (ADR-021 §A1):
//  1. Chiama MessagesGetFullChat o ChannelsGetFullChannel per ottenere pinnedMsgID.
//  2. Se pinnedMsgID == 0 → restituisce nil, nil (nessun pin).
//  3. Chiama MessagesGetMessages con pinnedMsgID → converte in model.Message.
//
// b.api è il *tg.Client raw (NON middleware-wrapped): non include flood-wait
// auto-retry. Il client gotd base in NewBridge usa telegram.NewClient senza
// contrib/middleware → le chiamate API qui possono ricevere FLOOD_WAIT e
// restituiranno errore al caller (loadPinnedMessageCmd logga via PinnedMsgLoadedMsg.Err).
// Se flood-wait auto-retry è necessario, wrappare b.api con gotd/contrib middleware.
//
// Usato da loadPinnedMessageCmd in conversation_pinned.go (Step 33).
func (b *Bridge) LoadPinnedMessage(ctx context.Context, chat model.Chat) (*model.Message, error) {
	pinnedMsgID, err := b.fetchPinnedMsgID(ctx, chat)
	if err != nil || pinnedMsgID == 0 {
		return nil, err
	}
	return b.fetchMessageByID(ctx, chat, pinnedMsgID)
}

// fetchPinnedMsgID ottiene il pinnedMsgID dal full chat/channel info.
func (b *Bridge) fetchPinnedMsgID(ctx context.Context, chat model.Chat) (int, error) {
	switch chat.ID.PeerType {
	case model.PeerChat:
		r, err := b.api.MessagesGetFullChat(ctx, chat.ID.ID)
		if err != nil {
			return 0, err
		}
		if fc, ok := r.FullChat.(*tg.ChatFull); ok {
			if pid, ok := fc.GetPinnedMsgID(); ok {
				return pid, nil
			}
		}
	case model.PeerChannel:
		inp := &tg.InputChannel{ChannelID: chat.ID.ID, AccessHash: chat.AccessHash}
		r, err := b.api.ChannelsGetFullChannel(ctx, inp)
		if err != nil {
			return 0, err
		}
		if cf, ok := r.FullChat.(*tg.ChannelFull); ok {
			if pid, ok := cf.GetPinnedMsgID(); ok {
				return pid, nil
			}
		}
	case model.PeerUser:
		// Per 1:1: UserFull ha PinnedMsgID.
		inp := &tg.InputUser{UserID: chat.ID.ID, AccessHash: chat.AccessHash}
		r, err := b.api.UsersGetFullUser(ctx, inp)
		if err != nil {
			return 0, err
		}
		// r.FullUser è di tipo tg.UserFull (non puntatore all'interfaccia).
		if pid, ok := r.FullUser.GetPinnedMsgID(); ok {
			return pid, nil
		}
	}
	return 0, nil
}

// fetchMessageByID recupera un singolo messaggio tramite MessagesGetMessages.
func (b *Bridge) fetchMessageByID(ctx context.Context, chat model.Chat, msgID int) (*model.Message, error) {
	r, err := b.api.MessagesGetMessages(ctx, []tg.InputMessageClass{&tg.InputMessageID{ID: msgID}})
	if err != nil {
		return nil, err
	}
	fallback := ""
	if chat.ID.PeerType == model.PeerUser {
		fallback = chat.Title
	}
	msgs := parseMessages(r, fallback)
	if len(msgs) == 0 {
		return nil, nil
	}
	m := msgs[0]
	return &m, nil
}
