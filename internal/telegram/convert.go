package telegram

import (
	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
)

func convertDialogsWithDates(
	dialogs []tg.DialogClass,
	users map[int64]*tg.User,
	chats map[int64]*tg.Chat,
	channels map[int64]*tg.Channel,
	msgDates map[int]int,
) []model.Chat {
	result := make([]model.Chat, 0, len(dialogs))
	for _, d := range dialogs {
		dialog, ok := d.(*tg.Dialog)
		if !ok {
			continue
		}
		if chat, ok := convertDialog(dialog, users, chats, channels); ok {
			chat.TopMessageDate = msgDates[dialog.TopMessage]
			result = append(result, chat)
		}
	}
	return result
}

func convertDialog(
	d *tg.Dialog,
	users map[int64]*tg.User,
	chats map[int64]*tg.Chat,
	channels map[int64]*tg.Channel,
) (model.Chat, bool) {
	chat := model.Chat{
		UnreadCount: d.UnreadCount,
		IsPinned:    d.Pinned,
		IsMuted:     isDialogMuted(d),
	}

	switch p := d.Peer.(type) {
	case *tg.PeerUser:
		u, ok := users[p.UserID]
		if !ok {
			return chat, false
		}
		chat.ID = model.ChatID{PeerType: model.PeerUser, ID: p.UserID}
		// BLOCKING #5: sanitize server-supplied title per prevenire ANSI injection.
		chat.Title = convert.SanitizeText(userDisplayName(u))
		chat.Type = deriveChatTypeFromUser(u)
		chat.AccessHash = u.AccessHash
		chat.IsOnline = isUserOnline(u)
	case *tg.PeerChat:
		c, ok := chats[p.ChatID]
		if !ok {
			return chat, false
		}
		chat.ID = model.ChatID{PeerType: model.PeerChat, ID: p.ChatID}
		chat.Title = convert.SanitizeText(c.Title)
		chat.Type = model.ChatGroup
	case *tg.PeerChannel:
		ch, ok := channels[p.ChannelID]
		if !ok {
			return chat, false
		}
		chat.ID = model.ChatID{PeerType: model.PeerChannel, ID: p.ChannelID}
		chat.Title = convert.SanitizeText(ch.Title)
		chat.AccessHash = ch.AccessHash
		chat.Type = model.ChatChannel
		if !ch.Broadcast {
			chat.Type = model.ChatGroup
		}
	default:
		return chat, false
	}

	return chat, true
}

func userDisplayName(u *tg.User) string {
	if u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	if u.FirstName != "" {
		return u.FirstName
	}
	return "Unknown"
}

func isUserOnline(u *tg.User) bool {
	_, ok := u.Status.(*tg.UserStatusOnline)
	return ok
}

func deriveChatTypeFromUser(u *tg.User) model.ChatType {
	if u.Bot {
		return model.ChatBot
	}
	if u.Self {
		return model.ChatSavedMessages
	}
	return model.ChatPrivate
}

func isDialogMuted(d *tg.Dialog) bool {
	muteUntil, ok := d.NotifySettings.GetMuteUntil()
	return ok && muteUntil > 0
}
