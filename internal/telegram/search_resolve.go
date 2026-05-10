package telegram

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

const snippetMaxRunes = 120

// truncateSnippet tronca s a snippetMaxRunes rune aggiungendo "…" se necessario.
// Usa utf8.RuneCountInString per gestire correttamente i multibyte.
func truncateSnippet(s string) string {
	if utf8.RuneCountInString(s) <= snippetMaxRunes {
		return s
	}
	runes := []rune(s)
	return string(runes[:snippetMaxRunes]) + "…"
}

// resolveUserTitle restituisce il display name di un utente.
// Priorità: FirstName+LastName → FirstName → Username → "Unknown".
func resolveUserTitle(u *tg.User) string {
	if u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	if u.Username != "" {
		return u.Username
	}
	return "Unknown"
}

// buildChatTitles costruisce una mappa peerID → displayTitle da un set di ChatClass.
// Distingue *tg.Chat (basic group) e *tg.Channel (supergroup/channel).
func buildChatTitles(chatClasses []tg.ChatClass) map[int64]string {
	m := make(map[int64]string, len(chatClasses))
	for _, c := range chatClasses {
		switch ch := c.(type) {
		case *tg.Chat:
			m[ch.ID] = ch.Title
		case *tg.Channel:
			m[ch.ID] = ch.Title
		}
	}
	return m
}

// buildUserTitles costruisce una mappa userID → displayTitle da un set di UserClass.
func buildUserTitles(userClasses []tg.UserClass) map[int64]string {
	m := make(map[int64]string, len(userClasses))
	for _, u := range userClasses {
		if user, ok := u.(*tg.User); ok {
			m[user.ID] = resolveUserTitle(user)
		}
	}
	return m
}

// buildHit costruisce un SearchHit da un *tg.Message usando le mappe title già pronte.
// Delegato a search_resolve per tenere search.go sotto il limite 120-LOC (LOC-gate).
func buildHit(msg *tg.Message, userTitles, chatTitles map[int64]string) model.SearchHit {
	chatID := chatIDFromPeer(msg.PeerID)
	chatTitle := resolveChatTitle(chatID, userTitles, chatTitles)
	senderName := resolveMsgSender(msg, userTitles)
	snippet := truncateSnippet(strings.TrimSpace(msg.Message))
	return model.SearchHit{
		ChatID:     chatID,
		ChatTitle:  chatTitle,
		MessageID:  msg.ID,
		SenderName: senderName,
		Snippet:    snippet,
		Date:       time.Unix(int64(msg.Date), 0),
	}
}

// resolveChatTitle restituisce il titolo della chat dato il suo ChatID.
func resolveChatTitle(id model.ChatID, userTitles, chatTitles map[int64]string) string {
	switch id.PeerType {
	case model.PeerUser:
		if t, ok := userTitles[id.ID]; ok {
			return t
		}
	case model.PeerChat, model.PeerChannel:
		if t, ok := chatTitles[id.ID]; ok {
			return t
		}
	}
	return "Unknown"
}

// resolveMsgSender restituisce il display name del mittente.
// Canali broadcast non hanno FromID; restituisce "" in quel caso.
func resolveMsgSender(msg *tg.Message, userTitles map[int64]string) string {
	from, ok := msg.FromID.(*tg.PeerUser)
	if !ok {
		return ""
	}
	if name, found := userTitles[from.UserID]; found {
		return name
	}
	return ""
}
