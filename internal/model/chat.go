package model

// ChatType è il tipo semantico della chat.
type ChatType int

const (
	ChatPrivate ChatType = iota
	ChatGroup
	ChatChannel
	ChatBot
	ChatSavedMessages
)

// PeerType identifica il namespace Telegram del peer.
type PeerType int

const (
	PeerUser PeerType = iota
	PeerChat
	PeerChannel
)

// ChatID identifica univocamente una chat Telegram.
type ChatID struct {
	PeerType PeerType
	ID       int64
}

// Chat rappresenta una chat nel dominio.
type Chat struct {
	ID             ChatID
	Type           ChatType
	Title          string
	UnreadCount    int
	IsOnline       bool
	IsMuted        bool
	IsPinned       bool
	AccessHash     int64
	TopMessageDate int // unix timestamp dell'ultimo messaggio, per sorting
	// Step 33: ID del messaggio pinnato più recente (DA1).
	// 0 = nessun pin. Popolato dal convert layer da tg.Dialog.PinnedMsgID.
	PinnedMsgID int
}

// HasUnread restituisce true se la chat ha messaggi non letti.
func (c Chat) HasUnread() bool {
	return c.UnreadCount > 0 && !c.IsMuted
}
