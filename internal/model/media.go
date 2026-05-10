package model

// MediaType è il discriminante del tagged-struct MessageMedia.
// Pattern identico a DeliveryStatus (Step 16) e ChatType.
type MediaType int

const (
	MediaPhoto    MediaType = iota + 1 // tg.MessageMediaPhoto
	MediaVideo                         // tg.MessageMediaDocument + AttributeVideo / mime video/*
	MediaDocument                      // tg.MessageMediaDocument (default fallback)
	MediaVoice                         // tg.MessageMediaDocument + AttributeAudio{Voice:true}
	MediaSticker                       // tg.MessageMediaDocument + AttributeSticker
	MediaLocation                      // tg.MessageMediaGeo / Venue (Step 24: fallback render)
	MediaContact                       // tg.MessageMediaContact (Step 24: fallback render)
	MediaPoll                          // tg.MessageMediaPoll (Step 24: fallback render)
)

// MessageMedia è un tagged-struct che porta il payload media di un messaggio.
// Il campo Type è il discriminante; gli altri campi sono popolati selettivamente
// per kind (vedi commenti inline). Il puntatore *MessageMedia in Message è nil
// quando il messaggio non ha media.
//
// Invariante: se presente, Type != 0.
type MessageMedia struct {
	Type MediaType // discriminante — SEMPRE valorizzato

	// Campi comuni a Photo / Video / Document.
	FileName  string // "" → usa fallback dal mime (photo.jpg, video.mp4, …)
	MimeType  string // mime IANA ("image/jpeg", "video/mp4", …)
	SizeBytes int64  // 0 → non mostrare "(size)" nella summary

	// Voice / Video: durata in secondi (0 se assente).
	DurationSec int

	// Voice: campioni 5-bit packed little-endian (len==0 → fallback flat line).
	WaveformData []byte

	// Sticker: emoji alternativa (es. "😀") e nome del pack (es. "Animated Cats").
	// "" emoji → fallback "🖼️"; "" pack → solo emoji nella summary.
	StickerEmoji    string
	StickerPackName string

	// Dimensioni (Photo / Video), opzionali, non usate nel render Step 24.
	Width  int
	Height int
}

// Icon restituisce l'emoji-icona del tipo media.
// Per Sticker restituisce la StickerEmoji (o "🖼️" se assente).
func (m MessageMedia) Icon() string {
	switch m.Type {
	case MediaPhoto:
		return "📷"
	case MediaVideo:
		return "🎬"
	case MediaDocument:
		return "📎"
	case MediaVoice:
		return "🎤"
	case MediaSticker:
		if m.StickerEmoji == "" {
			return "🖼️"
		}
		return m.StickerEmoji
	default:
		return "📄"
	}
}

// Summary restituisce la stringa inline per il bubble render.
// Garantisce stringa non vuota per ogni kind (invariante RENDER_FALLBACK_SAFE).
// Dispatch a media_summary.go per tenere i file sotto 120 LOC.
func (m MessageMedia) Summary() string {
	switch m.Type {
	case MediaPhoto:
		return photoSummary(m)
	case MediaVideo:
		return fileSummary("🎬", m.FileName, "video.mp4", m.SizeBytes, m.DurationSec)
	case MediaDocument:
		return fileSummary("📎", m.FileName, "document", m.SizeBytes, 0)
	case MediaVoice:
		return voiceSummary(m)
	case MediaSticker:
		return stickerSummary(m)
	default:
		return genericSummary(m.Type)
	}
}
