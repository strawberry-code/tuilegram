package model

import "time"

// DeliveryStatus rappresenta lo stato di consegna di un messaggio outgoing.
type DeliveryStatus int

const (
	StatusUnknown   DeliveryStatus = iota
	StatusSent                     // ✓  grigio — inviato al server
	StatusDelivered                // ✓✓ grigio — consegnato
	StatusRead                     // ✓✓ blu    — letto
)

// Reaction è una singola reazione emoji con contatore e flag "chosen by me".
// La slice Message.Reactions è ordinata per Count desc, Emoji asc (ADR-012 A1).
// Invariante REACTIONS_ORDERED: garantita dal convert layer a ogni snapshot replace.
type Reaction struct {
	Emoji      string
	Count      int
	ChosenByMe bool
}

// MessageLink è un link rilevato nelle entità Telegram del messaggio.
// Offset e Length sono UTF-16 code units (Telegram MTProto spec —
// NON rune Go né byte). Usare convert.Utf16Slice per estrarre substrings.
// Invariante LINK_DETECTION_AUTHORITATIVE: deriva solo da tg.Message.Entities.
type MessageLink struct {
	Offset int
	Length int
	URL    string
}

// Message è un messaggio in una conversazione.
// IsService=true indica un system message (join/leave/pin/ecc.); in tal caso
// Reactions è sempre nil e ServiceText contiene il testo pre-formattato
// (invarianti SYSTEM_NO_REACT + SYSTEM_IMMUTABLE — reactions.tla Step 25).
type Message struct {
	ID          int
	SenderID    int64 // ID numerico Telegram del mittente (per sender color)
	SenderName  string
	Text        string
	Date        time.Time
	IsOutgoing  bool
	Status      DeliveryStatus
	ReplyToID   int           // ID del messaggio citato (0 = nessuna reply)
	Media       *MessageMedia // nil = nessun media; popolato dal convert layer (Step 24)
	IsService   bool          // true per tg.MessageService (Step 25)
	ServiceText string        // testo centrato pre-formattato (es. "Alice joined")
	Reactions   []Reaction    // snapshot ordinato; nil se assenti o IsService=true
	// Step 33: link entities (DB1) — popolate dal convert layer.
	Links []MessageLink
	// Step 33: forward display (DC1) — popolate dal convert layer.
	IsForwarded   bool   // true se il messaggio è un forward (FwdFrom != nil)
	ForwardedFrom string // label displayable: "@user" | "Name" | "Hidden"
}
