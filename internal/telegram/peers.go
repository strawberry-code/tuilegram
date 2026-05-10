package telegram

import (
	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// chatIDFromPeer converte un PeerClass nel ChatID del dominio.
func chatIDFromPeer(peer tg.PeerClass) model.ChatID {
	switch p := peer.(type) {
	case *tg.PeerUser:
		return model.ChatID{PeerType: model.PeerUser, ID: p.UserID}
	case *tg.PeerChat:
		return model.ChatID{PeerType: model.PeerChat, ID: p.ChatID}
	case *tg.PeerChannel:
		return model.ChatID{PeerType: model.PeerChannel, ID: p.ChannelID}
	}
	return model.ChatID{}
}

// buildNamesFromUserMap costruisce una mappa id→nome da una mappa di User.
func buildNamesFromUserMap(users map[int64]*tg.User) map[int64]string {
	names := make(map[int64]string, len(users))
	for id, u := range users {
		name := u.FirstName
		if u.LastName != "" {
			name += " " + u.LastName
		}
		if name == "" {
			name = u.Username
		}
		names[id] = name
	}
	return names
}
