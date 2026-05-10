package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// link_render.go — rendering di link con lipgloss underline + OSC 8 (ADR-021 §DB2).
// Invariante LINK_DETECTION_AUTHORITATIVE: usa solo msg.Links (da Entities server).

// renderTextWithLinks applica underline + OSC 8 ai link nel testo del messaggio.
// Segmenta il testo usando cursore UTF-16 (Telegram entity spec) via convert.Utf16Slice.
// Per terminali senza supporto OSC 8: escape ignorata silenziosamente (ADR-021 §DB2).
func renderTextWithLinks(text string, links []model.MessageLink) string {
	if len(links) == 0 {
		return text
	}
	var sb strings.Builder
	cursorU16 := 0 // cursore in UTF-16 code units (Telegram entity spec)
	for _, lnk := range links {
		if lnk.Offset < 0 || lnk.Length <= 0 {
			continue // WARNING #5: skip entità malformate
		}
		if lnk.Offset > cursorU16 {
			// testo prima dell'entity (pre-gap)
			pre := convert.Utf16Slice(text, cursorU16, lnk.Offset-cursorU16)
			sb.WriteString(pre)
		}
		linkText := convert.Utf16Slice(text, lnk.Offset, lnk.Length)
		styled := lipgloss.NewStyle().
			Underline(true).
			Foreground(styles.ColorLink()).
			Render(linkText)
		sb.WriteString(osc8Wrap(lnk.URL, styled))
		cursorU16 = lnk.Offset + lnk.Length
	}
	// testo dopo l'ultima entity
	totalU16 := convert.Utf16Length(text)
	if cursorU16 < totalU16 {
		sb.WriteString(convert.Utf16Slice(text, cursorU16, totalU16-cursorU16))
	}
	return sb.String()
}

// osc8Wrap racchiude text con OSC 8 hyperlink escape sequence.
// Formato: ESC ] 8 ; ; <url> ESC \ <text> ESC ] 8 ; ; ESC \
// URL viene sanitizzato per rimuovere control chars (BLOCKING #4: OSC injection).
func osc8Wrap(url, text string) string {
	const osc = "\x1b]8;;"
	const st = "\x1b\\"
	return osc + sanitizeURLForOSC8(url) + st + text + osc + st
}

// sanitizeURLForOSC8 rimuove control characters (< 0x20, == 0x7f) dall'URL.
// Previene OSC 8 injection via URL contenenti \x1b\\ che chiuderebbero prematuramente
// la sequenza e inietterebbero comandi terminale arbitrari (BLOCKING #4).
func sanitizeURLForOSC8(url string) string {
	needsClean := false
	for _, r := range url {
		if r < 0x20 || r == 0x7f {
			needsClean = true
			break
		}
	}
	if !needsClean {
		return url
	}
	var b strings.Builder
	b.Grow(len(url))
	for _, r := range url {
		if r < 0x20 || r == 0x7f {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
