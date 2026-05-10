package convert

import "strings"

// SanitizeText rimuove control characters (< 0x20, == 0x7f) dal testo
// eccetto \t (0x09) e \n (0x0a), che sono whitespace legittimi.
// Applicato ai dati untrusted server-side (sender name, forward label, testo
// messaggio) prima del render terminale per prevenire ANSI/OSC injection.
// Non applicare agli URL: il sanitizer OSC 8 li gestisce separatamente.
func SanitizeText(s string) string {
	// Fast path: scan per control chars prima di allocare Builder.
	needsClean := false
	for _, r := range s {
		if (r < 0x20 && r != '\t' && r != '\n') || r == 0x7f {
			needsClean = true
			break
		}
	}
	if !needsClean {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\t' || r == '\n' {
			b.WriteRune(r)
			continue
		}
		if r < 0x20 || r == 0x7f {
			continue // strip control char
		}
		b.WriteRune(r)
	}
	return b.String()
}
