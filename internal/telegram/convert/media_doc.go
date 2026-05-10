package convert

import (
	"strings"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// dispatchDocument applies the ADR-011 §2 attribute-priority cascade:
// 1. AttributeAudio.Voice==true  → MediaVoice
// 2. AttributeAudio.Voice==false → MediaDocument (audio-as-document per ADR)
// 3. AttributeSticker             → MediaSticker
// 4. AttributeVideo               → MediaVideo
// 5. mime "video/*"               → MediaVideo
// 6. mime "audio/*"               → MediaDocument (per ADR: only Voice gets MediaVoice)
// 7. default                      → MediaDocument
func dispatchDocument(doc *tg.Document) *model.MessageMedia {
	out := &model.MessageMedia{
		MimeType:  doc.MimeType,
		SizeBytes: doc.Size,
	}

	// Extract FileName from attributes (done once; other attrs also scanned below).
	var (
		audio   *tg.DocumentAttributeAudio
		sticker *tg.DocumentAttributeSticker
		video   *tg.DocumentAttributeVideo
	)
	for _, attr := range doc.Attributes {
		switch a := attr.(type) {
		case *tg.DocumentAttributeAudio:
			audio = a
		case *tg.DocumentAttributeSticker:
			sticker = a
		case *tg.DocumentAttributeVideo:
			video = a
		case *tg.DocumentAttributeFilename:
			out.FileName = a.FileName
		}
	}

	// Priority 1: Voice message (AttributeAudio.Voice == true).
	if audio != nil && audio.Voice {
		out.Type = model.MediaVoice
		out.DurationSec = audio.Duration
		out.WaveformData = audio.Waveform
		return out
	}

	// Priority 2: Audio document (AttributeAudio.Voice == false).
	// Per ADR-011: audio (non-voice) stays MediaDocument; only Voice gets MediaVoice.
	if audio != nil {
		out.Type = model.MediaDocument
		out.DurationSec = audio.Duration
		return out
	}

	// Priority 3: Sticker.
	if sticker != nil {
		return buildSticker(out, sticker)
	}

	// Priority 4: Video attribute.
	if video != nil {
		out.Type = model.MediaVideo
		out.DurationSec = int(video.Duration)
		out.Width = video.W
		out.Height = video.H
		return out
	}

	// Priority 5-6: Mime fallback.
	switch {
	case strings.HasPrefix(doc.MimeType, "video/"):
		out.Type = model.MediaVideo
	case strings.HasPrefix(doc.MimeType, "audio/"):
		// Per ADR: audio/* without explicit Voice attr → MediaDocument.
		out.Type = model.MediaDocument
	default:
		out.Type = model.MediaDocument
	}
	return out
}

// buildSticker populates the sticker fields from DocumentAttributeSticker.
// StickerPackName extracted only if Stickerset is InputStickerSetShortName;
// RPC is required for the human-readable title — left empty per ADR-011 §sticker.
func buildSticker(out *model.MessageMedia, s *tg.DocumentAttributeSticker) *model.MessageMedia {
	out.Type = model.MediaSticker
	out.StickerEmoji = s.Alt
	if ss, ok := s.Stickerset.(*tg.InputStickerSetShortName); ok {
		out.StickerPackName = ss.ShortName
	}
	return out
}
