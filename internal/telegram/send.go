package telegram

import (
	"context"
	"time"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// SendMessage invia un messaggio di testo a una chat.
// replyToID != 0 imposta la reply al messaggio indicato.
func (b *Bridge) SendMessage(ctx context.Context, chat model.Chat, text string, replyToID int) error {
	peer := chatIDToInputPeer(chat)
	req := &tg.MessagesSendMessageRequest{
		Peer:     peer,
		Message:  text,
		RandomID: time.Now().UnixNano(),
	}
	if replyToID != 0 {
		req.ReplyTo = &tg.InputReplyToMessage{ReplyToMsgID: replyToID}
	}
	_, err := b.api.MessagesSendMessage(ctx, req)
	return err
}
