package telegram

import (
	"context"
	"crypto/rand"
	"encoding/binary"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// ForwardMessage inoltra uno o più messaggi (identificati da msgIDs) dalla chat
// source alla chat target tramite MessagesForwardMessages.
// RandomID richiede un int64 per ogni messaggio: generati via crypto/rand.
func (b *Bridge) ForwardMessage(ctx context.Context, target, source model.Chat, msgIDs []int) error {
	randomIDs, err := generateRandomIDs(len(msgIDs))
	if err != nil {
		return err
	}
	_, err = b.api.MessagesForwardMessages(ctx, &tg.MessagesForwardMessagesRequest{
		FromPeer: chatIDToInputPeer(source),
		ToPeer:   chatIDToInputPeer(target),
		ID:       msgIDs,
		RandomID: randomIDs,
	})
	return err
}

// generateRandomIDs produce n int64 crittograficamente casuali per i RandomID
// richiesti dall'API Telegram (de-duplicazione lato server).
func generateRandomIDs(n int) ([]int64, error) {
	ids := make([]int64, n)
	for i := range ids {
		if err := binary.Read(rand.Reader, binary.LittleEndian, &ids[i]); err != nil {
			return nil, err
		}
	}
	return ids, nil
}
