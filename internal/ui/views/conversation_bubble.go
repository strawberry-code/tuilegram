package views

import "github.com/charmbracelet/x/ansi"

// conversation_bubble.go — Step 34: bubble width cap + wrap helpers.
// Cap = 60% del viewport, min 20 col. Wrap word-aware con hard-fallback
// su parole troppo lunghe (ansi.Wrap preserva escape sequences).

// bubbleWidth: max chars per riga del bubble — 60% del viewport.
// Sotto i 20 cap a 20 per evitare wrap aggressivo su finestre piccole.
func bubbleWidth(viewportW int) int {
	w := viewportW * 60 / 100
	if w < 20 {
		w = 20
	}
	return w
}

// wrapBubble word-wrap + hard-wrap su parole troppo lunghe (ansi-aware).
func wrapBubble(text string, viewportW int) string {
	return ansi.Wrap(text, bubbleWidth(viewportW), "")
}
