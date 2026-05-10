package telegram

import "github.com/gotd/td/tg"

// buildUserNames costruisce una mappa id→displayName dai UserClass restituiti
// nelle risposte MessagesGet*. Usato da parseMessages per risolvere i sender.
func buildUserNames(userClasses []tg.UserClass) map[int64]string {
	users := indexUsers(userClasses)
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
