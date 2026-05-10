package convert_test

import (
	"testing"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
)

func makeDoc(mime string, attrs ...tg.DocumentAttributeClass) *tg.MessageMediaDocument {
	return &tg.MessageMediaDocument{
		Document: &tg.Document{MimeType: mime, Size: 1024, Attributes: attrs},
	}
}

func TestDocument_Voice(t *testing.T) {
	wf := []byte{0x01, 0x02, 0x03}
	attr := &tg.DocumentAttributeAudio{Voice: true, Duration: 42, Waveform: wf}
	got := convert.ToMessageMedia(makeDoc("audio/ogg", attr))
	if got == nil || got.Type != model.MediaVoice {
		t.Fatalf("expected MediaVoice, got %v", got)
	}
	if got.DurationSec != 42 {
		t.Fatalf("expected DurationSec=42, got %d", got.DurationSec)
	}
	if string(got.WaveformData) != string(wf) {
		t.Fatal("WaveformData not copied")
	}
}

func TestDocument_AudioNonVoice_IsDocument(t *testing.T) {
	// Per ADR-011: audio (non-voice) → MediaDocument, not MediaVoice.
	attr := &tg.DocumentAttributeAudio{Voice: false, Duration: 200}
	got := convert.ToMessageMedia(makeDoc("audio/mp3", attr))
	if got == nil || got.Type != model.MediaDocument {
		t.Fatalf("expected MediaDocument for non-voice audio, got %v", got)
	}
}

func TestDocument_Sticker(t *testing.T) {
	attr := &tg.DocumentAttributeSticker{
		Alt:        "🎉",
		Stickerset: &tg.InputStickerSetShortName{ShortName: "PartyPack"},
	}
	got := convert.ToMessageMedia(makeDoc("image/webp", attr))
	if got == nil || got.Type != model.MediaSticker {
		t.Fatalf("expected MediaSticker, got %v", got)
	}
	if got.StickerEmoji != "🎉" {
		t.Fatalf("expected StickerEmoji=🎉, got %q", got.StickerEmoji)
	}
	if got.StickerPackName != "PartyPack" {
		t.Fatalf("expected StickerPackName=PartyPack, got %q", got.StickerPackName)
	}
}

func TestDocument_Video_Attribute(t *testing.T) {
	attr := &tg.DocumentAttributeVideo{Duration: 135.5, W: 1280, H: 720}
	got := convert.ToMessageMedia(makeDoc("video/mp4", attr))
	if got == nil || got.Type != model.MediaVideo {
		t.Fatalf("expected MediaVideo, got %v", got)
	}
	if got.DurationSec != 135 {
		t.Fatalf("expected DurationSec=135, got %d", got.DurationSec)
	}
	if got.Width != 1280 || got.Height != 720 {
		t.Fatalf("unexpected dimensions: %dx%d", got.Width, got.Height)
	}
}

func TestDocument_VideoMime_NoAttr(t *testing.T) {
	// No Video attribute, but mime = video/mp4 → MediaVideo via mime fallback.
	got := convert.ToMessageMedia(makeDoc("video/mp4"))
	if got == nil || got.Type != model.MediaVideo {
		t.Fatalf("expected MediaVideo via mime fallback, got %v", got)
	}
}

func TestDocument_PlainPDF(t *testing.T) {
	attr := &tg.DocumentAttributeFilename{FileName: "report.pdf"}
	got := convert.ToMessageMedia(makeDoc("application/pdf", attr))
	if got == nil || got.Type != model.MediaDocument {
		t.Fatalf("expected MediaDocument for PDF, got %v", got)
	}
	if got.FileName != "report.pdf" {
		t.Fatalf("expected FileName=report.pdf, got %q", got.FileName)
	}
	if got.SizeBytes != 1024 {
		t.Fatalf("expected SizeBytes=1024, got %d", got.SizeBytes)
	}
	if got.MimeType != "application/pdf" {
		t.Fatalf("expected MimeType=application/pdf, got %q", got.MimeType)
	}
}

func TestDocument_VoicePriorityOverMime(t *testing.T) {
	// Voice attribute must win over audio/ogg mime (priority 1 > mime fallback).
	attr := &tg.DocumentAttributeAudio{Voice: true, Duration: 10}
	got := convert.ToMessageMedia(makeDoc("audio/ogg", attr))
	if got == nil || got.Type != model.MediaVoice {
		t.Fatalf("Voice attribute must override audio/* mime: got %v", got)
	}
}
