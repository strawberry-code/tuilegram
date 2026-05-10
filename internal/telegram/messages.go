package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// LoadMessages carica la history di una chat (ultimi 50 messaggi).
func (b *Bridge) LoadMessages(ctx context.Context, chat model.Chat) ([]model.Message, error) {
	peer := chatIDToInputPeer(chat)
	result, err := b.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:  peer,
		Limit: 50,
	})
	if err != nil {
		return nil, err
	}
	// In chat 1:1 Telegram omette spesso FromID; usiamo il titolo come fallback sender.
	fallback := ""
	if chat.ID.PeerType == model.PeerUser {
		fallback = chat.Title
	}
	return parseMessages(result, fallback), nil
}

// LoadMessagesAround loads messages centered around a given message ID.
// Used for jump-to-result from global search: returns ~25 messages before
// and ~25 after the target so the message lands roughly in the middle of
// the viewport.
// MTProto: getHistory with OffsetID=centerMsgID, AddOffset=-25, Limit=50.
// If centerMsgID <= 0 falls back to LoadMessages behavior.
func (b *Bridge) LoadMessagesAround(ctx context.Context, chat model.Chat, centerMsgID int) ([]model.Message, error) {
	if centerMsgID <= 0 {
		return b.LoadMessages(ctx, chat)
	}
	peer := chatIDToInputPeer(chat)
	result, err := b.api.MessagesGetHistory(ctx, &tg.MessagesGetHistoryRequest{
		Peer:      peer,
		OffsetID:  centerMsgID,
		AddOffset: -25,
		Limit:     50,
	})
	if err != nil {
		return nil, err
	}
	fallback := ""
	if chat.ID.PeerType == model.PeerUser {
		fallback = chat.Title
	}
	return parseMessages(result, fallback), nil
}

// chatIDToInputPeer converte un ChatID nel tipo InputPeerClass corretto.
func chatIDToInputPeer(chat model.Chat) tg.InputPeerClass {
	switch chat.ID.PeerType {
	case model.PeerUser:
		return &tg.InputPeerUser{UserID: chat.ID.ID, AccessHash: chat.AccessHash}
	case model.PeerChat:
		return &tg.InputPeerChat{ChatID: chat.ID.ID}
	case model.PeerChannel:
		return &tg.InputPeerChannel{ChannelID: chat.ID.ID, AccessHash: chat.AccessHash}
	}
	return &tg.InputPeerEmpty{}
}

func parseMessages(result tg.MessagesMessagesClass, fallback string) []model.Message {
	var rawMsgs []tg.MessageClass
	var users []tg.UserClass

	switch r := result.(type) {
	case *tg.MessagesMessages:
		rawMsgs, users = r.Messages, r.Users
	case *tg.MessagesMessagesSlice:
		rawMsgs, users = r.Messages, r.Users
	case *tg.MessagesChannelMessages:
		rawMsgs, users = r.Messages, r.Users
	default:
		return nil
	}

	names := buildUserNames(users)
	msgs := make([]model.Message, 0, len(rawMsgs))
	for _, raw := range rawMsgs {
		switch m := raw.(type) {
		case *tg.Message:
			if m.Message == "" && m.Media == nil {
				continue // skip empty
			}
			msgs = append(msgs, toMessage(m, names, fallback))
		case *tg.MessageService:
			// Step 25: system messages included in history (IsService=true).
			msgs = append(msgs, toServiceMessage(m, names))
			// *tg.MessageEmpty: skip (server placeholder)
		}
	}

	// Inverti: Telegram restituisce dal più recente al più vecchio.
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}
	return msgs
}
