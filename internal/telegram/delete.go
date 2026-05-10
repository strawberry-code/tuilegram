package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// DeleteMessage cancella un messaggio. Usa l'API channel per i canali,
// MessagesDeleteMessages con revoke=true per chat/gruppi.
func (b *Bridge) DeleteMessage(ctx context.Context, chat model.Chat, msgID int) error {
	if chat.ID.PeerType == model.PeerChannel {
		ch := &tg.InputChannel{ChannelID: chat.ID.ID, AccessHash: chat.AccessHash}
		_, err := b.api.ChannelsDeleteMessages(ctx, &tg.ChannelsDeleteMessagesRequest{
			Channel: ch,
			ID:      []int{msgID},
		})
		return err
	}
	_, err := b.api.MessagesDeleteMessages(ctx, &tg.MessagesDeleteMessagesRequest{
		Revoke: true,
		ID:     []int{msgID},
	})
	return err
}
