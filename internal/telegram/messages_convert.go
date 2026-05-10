package telegram

import (
	"time"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
)

// extractSenderID restituisce l'ID numerico del mittente del messaggio.
// 0 per messaggi outgoing (sender siamo noi) o senza FromID.
func extractSenderID(m *tg.Message) int64 {
	if m.Out {
		return 0
	}
	if from, ok := m.FromID.(*tg.PeerUser); ok {
		return from.UserID
	}
	return 0
}

// toMessage converts a *tg.Message (regular) to a domain Message.
// Populates: sender, text, date, status, replyToID, media, reactions,
// links (Step 33 DB1), forward header (Step 33 DC1).
// IsService is always false for this path (ADR-012 B1).
func toMessage(m *tg.Message, names map[int64]string, fallback string) model.Message {
	sender := resolveSender(m, names, fallback)
	status := model.StatusUnknown
	if m.Out {
		status = model.StatusDelivered
	}
	var replyToID int
	if rh, ok := m.ReplyTo.(*tg.MessageReplyHeader); ok {
		replyToID, _ = rh.GetReplyToMsgID()
	}
	// Step 25: extract reactions snapshot; nil if absent or all custom-emoji.
	var reactions []model.Reaction
	if r, ok := m.GetReactions(); ok {
		reactions = convert.FlattenReactions(r)
	}
	// Step 33: link entities (DB1) — server-authoritative, no client regex.
	links := convert.ExtractLinks(m.Message, m.Entities)
	// Step 33: forward header (DC1).
	isForwarded := false
	forwardedFrom := ""
	if fwd, ok := m.GetFwdFrom(); ok {
		isForwarded = true
		// BLOCKING #5: SanitizeText su label forward per prevenire ANSI injection.
		forwardedFrom = convert.SanitizeText(convert.ConvertFwdLabel(&fwd))
	}
	// BLOCKING #5: SanitizeText su testo e sender name (dati untrusted server-side).
	return model.Message{
		ID:            m.ID,
		SenderID:      extractSenderID(m),
		SenderName:    convert.SanitizeText(sender),
		Text:          convert.SanitizeText(m.Message),
		Date:          time.Unix(int64(m.Date), 0),
		IsOutgoing:    m.Out,
		Status:        status,
		ReplyToID:     replyToID,
		Media:         convert.ToMessageMedia(m.Media),
		Reactions:     reactions,
		Links:         links,
		IsForwarded:   isForwarded,
		ForwardedFrom: forwardedFrom,
	}
}

// toServiceMessage converts a *tg.MessageService to a domain Message.
// IsService=true; Reactions is always nil (invariant SYSTEM_NO_REACT,
// reactions.tla Step 25).
// Spec: entity-mapping.md §System Message Mapping
func toServiceMessage(m *tg.MessageService, names map[int64]string) model.Message {
	var actorID int64
	if from, ok := m.FromID.(*tg.PeerUser); ok {
		actorID = from.UserID
	}
	return model.Message{
		ID:          m.ID,
		Date:        time.Unix(int64(m.Date), 0),
		IsService:   true,
		ServiceText: convert.FormatAction(m.Action, names, actorID),
	}
}

// resolveSender determines the sender display name for a regular message.
// Priority: outgoing → "You"; FromID lookup → names map; fallback (chat title).
// Delegates the int64→name lookup to convert.LookupName so service-message and
// regular-message paths share a single source of truth (no duplication).
func resolveSender(m *tg.Message, names map[int64]string, fallback string) string {
	if m.Out {
		return "You"
	}
	if from, ok := m.FromID.(*tg.PeerUser); ok {
		if name, found := convert.LookupName(names, from.UserID); found {
			return name
		}
	}
	if fallback != "" {
		return fallback
	}
	return "Unknown"
}
