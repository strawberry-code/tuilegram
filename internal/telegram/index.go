package telegram

import "github.com/gotd/td/tg"

// Funzioni di indicizzazione per le entità Telegram.
// Trasformano slice in mappe per lookup O(1) durante la conversione.

func indexUsers(userClasses []tg.UserClass) map[int64]*tg.User {
	m := make(map[int64]*tg.User, len(userClasses))
	for _, u := range userClasses {
		if user, ok := u.(*tg.User); ok {
			m[user.ID] = user
		}
	}
	return m
}

func indexChats(chatClasses []tg.ChatClass) map[int64]*tg.Chat {
	m := make(map[int64]*tg.Chat)
	for _, c := range chatClasses {
		if chat, ok := c.(*tg.Chat); ok {
			m[chat.ID] = chat
		}
	}
	return m
}

func indexChannels(chatClasses []tg.ChatClass) map[int64]*tg.Channel {
	m := make(map[int64]*tg.Channel)
	for _, c := range chatClasses {
		if ch, ok := c.(*tg.Channel); ok {
			m[ch.ID] = ch
		}
	}
	return m
}

// indexMessageDates mappa messageID → date (unix timestamp).
func indexMessageDates(msgClasses []tg.MessageClass) map[int]int {
	m := make(map[int]int, len(msgClasses))
	for _, mc := range msgClasses {
		if msg, ok := mc.(*tg.Message); ok {
			m[msg.ID] = msg.Date
		}
	}
	return m
}
