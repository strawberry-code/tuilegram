package telegram

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// MaxSearchQueryRunes è il cap server-side per query.searchGlobal.
// Allineato al CharLimit del textinput UI (256). Difensivo contro
// caller che bypassano l'UI (test, future CLI). Vedi ADR-013.
const MaxSearchQueryRunes = 256

// ErrQueryTooLong restituito da SearchGlobal se la query supera MaxSearchQueryRunes.
var ErrQueryTooLong = errors.New("search query exceeds max rune length")

// SearchGlobal esegue messages.searchGlobal e restituisce i risultati.
// queryID è un token opaque propagato nel SearchResult per consentire al
// chiamante di scartare risultati stale (ADR-013: latestQueryID policy).
// Una query vuota restituisce immediatamente SearchResult{} senza RPC.
// Errori RPC propagati as-is (no retry interno; floodwait middleware è
// out-of-scope per Step 26 — vedi backlog).
func (b *Bridge) SearchGlobal(ctx context.Context, query string, limit int, queryID uint64) (model.SearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return model.SearchResult{QueryID: queryID}, nil
	}
	if utf8.RuneCountInString(query) > MaxSearchQueryRunes {
		return model.SearchResult{QueryID: queryID}, ErrQueryTooLong
	}
	if limit <= 0 {
		limit = 30
	}

	raw, err := b.api.MessagesSearchGlobal(ctx, &tg.MessagesSearchGlobalRequest{
		Q:          query,
		Filter:     &tg.InputMessagesFilterEmpty{},
		MinDate:    0,
		MaxDate:    0,
		OffsetRate: 0,
		OffsetPeer: &tg.InputPeerEmpty{},
		OffsetID:   0,
		Limit:      limit,
	})
	if err != nil {
		return model.SearchResult{}, err
	}

	hits := extractHits(raw)
	return model.SearchResult{QueryID: queryID, Hits: hits}, nil
}

// extractHits converte tg.MessagesMessagesClass in []model.SearchHit.
// Gestisce i tre case: MessagesMessages, MessagesMessagesSlice, MessagesChannelMessages.
// MessageService e MessageEmpty sono ignorati (solo *tg.Message produce hit).
func extractHits(result tg.MessagesMessagesClass) []model.SearchHit {
	var rawMsgs []tg.MessageClass
	var userClasses []tg.UserClass
	var chatClasses []tg.ChatClass

	switch r := result.(type) {
	case *tg.MessagesMessages:
		rawMsgs, userClasses, chatClasses = r.Messages, r.Users, r.Chats
	case *tg.MessagesMessagesSlice:
		rawMsgs, userClasses, chatClasses = r.Messages, r.Users, r.Chats
	case *tg.MessagesChannelMessages:
		rawMsgs, userClasses, chatClasses = r.Messages, r.Users, r.Chats
	default:
		return nil
	}

	userTitles := buildUserTitles(userClasses)
	chatTitles := buildChatTitles(chatClasses)

	hits := make([]model.SearchHit, 0, len(rawMsgs))
	for _, raw := range rawMsgs {
		msg, ok := raw.(*tg.Message)
		if !ok {
			continue // skip MessageService, MessageEmpty
		}
		hit := buildHit(msg, userTitles, chatTitles)
		hits = append(hits, hit)
	}
	return hits
}
