package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// EditMessage modifica il testo di un messaggio già inviato.
func (b *Bridge) EditMessage(ctx context.Context, chat model.Chat, msgID int, text string) error {
	peer := chatIDToInputPeer(chat)
	_, err := b.api.MessagesEditMessage(ctx, &tg.MessagesEditMessageRequest{
		Peer:    peer,
		ID:      msgID,
		Message: text,
	})
	return err
}
