package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// DeleteMessages cancella un insieme di messaggi identificati da msgIDs.
// Usa ChannelsDeleteMessages per i canali, MessagesDeleteMessages con
// revoke=true per le chat/gruppi (BATCH_ATOMICITY: una sola RPC).
func (b *Bridge) DeleteMessages(ctx context.Context, chat model.Chat, msgIDs []int) error {
	if chat.ID.PeerType == model.PeerChannel {
		ch := &tg.InputChannel{
			ChannelID:  chat.ID.ID,
			AccessHash: chat.AccessHash,
		}
		_, err := b.api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: ch,
			ID:      msgIDs,
		})
		return err
	}
	_, err := b.api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
		Revoke: true,
		ID:     msgIDs,
	})
	return err
}
