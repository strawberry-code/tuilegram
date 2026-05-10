package telegram

import (
	"context"
	"sort"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// LoadDialogs carica la lista delle chat dall'API Telegram, ordinate.
func (b *Bridge) LoadDialogs(ctx context.Context) ([]model.Chat, error) {
	result, err := b.api.MessagesGetDialogs(ctx, &tg.MessagesGetDialogsRequest{
		Limit:      100,
		OffsetPeer: &tg.InputPeerEmpty{},
	})
	if err != nil {
		return nil, err
	}

	chats, err := b.parseDialogs(result)
	if err != nil {
		return nil, err
	}

	sortChats(chats)
	return chats, nil
}

func (b *Bridge) parseDialogs(result tg.MessagesDialogsClass) ([]model.Chat, error) {
	var dialogs []tg.DialogClass
	var users map[int64]*tg.User
	var chatMap map[int64]*tg.Chat
	var channelMap map[int64]*tg.Channel
	var messages map[int]int // msgID → date

	switch r := result.(type) {
	case *tg.MessagesDialogs:
		dialogs = r.Dialogs
		users = indexUsers(r.Users)
		chatMap = indexChats(r.Chats)
		channelMap = indexChannels(r.Chats)
		messages = indexMessageDates(r.Messages)
	case *tg.MessagesDialogsSlice:
		dialogs = r.Dialogs
		users = indexUsers(r.Users)
		chatMap = indexChats(r.Chats)
		channelMap = indexChannels(r.Chats)
		messages = indexMessageDates(r.Messages)
	default:
		return nil, nil
	}

	return convertDialogsWithDates(dialogs, users, chatMap, channelMap, messages), nil
}

// sortChats ordina: pinned > unread > last message date (desc).
func sortChats(chats []model.Chat) {
	sort.SliceStable(chats, func(i, j int) bool {
		a, b := chats[i], chats[j]
		if a.IsPinned != b.IsPinned {
			return a.IsPinned
		}
		if a.HasUnread() != b.HasUnread() {
			return a.HasUnread()
		}
		return a.TopMessageDate > b.TopMessageDate
	})
}
