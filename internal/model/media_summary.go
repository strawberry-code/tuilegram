package model

import "github.com/strawberry-code/tuilegram/internal/ui/render"

// Questo file contiene le funzioni helper per MessageMedia.Summary().
// Separato da media.go per rispettare il limite 120 LOC.

// photoSummary produce la summary per MediaPhoto.
// Template: "📷 {filename} ({size})"
func photoSummary(m MessageMedia) string {
	name := m.FileName
	if name == "" {
		name = "photo.jpg"
	}
	if m.SizeBytes > 0 {
		return "📷 " + name + " (" + formatSize(m.SizeBytes) + ")"
	}
	return "📷 " + name
}

// fileSummary gestisce Video e Document con template uniforme.
// Template: "{icon} {filename} ({size}) · {duration}"
// size==0 → omette la parentesi; dur==0 → omette la durata.
func fileSummary(icon, name, fallback string, size int64, dur int) string {
	if name == "" {
		name = fallback
	}
	base := icon + " " + name
	if size > 0 {
		base += " (" + formatSize(size) + ")"
	}
	if dur > 0 {
		base += " · " + formatDuration(dur)
	}
	return base
}

// voiceSummary produce la summary per MediaVoice.
// Template: "🎤 {waveform} {duration}"
// waveform via render.Waveform (N=10 glifi fissi, fallback "──────────" se dati vuoti).
func voiceSummary(m MessageMedia) string {
	wf := render.Waveform(m.WaveformData, 10)
	return "🎤 " + wf + " " + formatDuration(m.DurationSec)
}

// stickerSummary produce la summary per MediaSticker.
// Template: "{emoji} {pack name}" oppure solo "{emoji}" se pack assente.
// emoji fallback: "🖼️" se StickerEmoji vuoto.
func stickerSummary(m MessageMedia) string {
	icon := m.StickerEmoji
	if icon == "" {
		icon = "🖼️"
	}
	if m.StickerPackName == "" {
		return icon
	}
	return icon + " " + m.StickerPackName
}

// genericSummary produce la summary fallback per kind non renderizzati inline
// in Step 24 (Location, Contact, Poll, e qualsiasi variante futura).
func genericSummary(t MediaType) string {
	switch t {
	case MediaLocation:
		return "📍 Location"
	case MediaContact:
		return "👤 Contact"
	case MediaPoll:
		return "📊 Poll"
	}
	return "📄 Media"
}
