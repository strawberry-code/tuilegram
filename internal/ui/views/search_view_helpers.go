package views

import (
	"fmt"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// formatHit formats a single search hit as a one-line summary:
// "ChatTitle · SenderName: snippet  (MM/DD)"
// If SenderName is empty (anonymous channel), ChatTitle is used instead.
// Snippet is truncated to 60 runes to keep rows within the modal width.
func formatHit(h model.SearchHit) string {
	sender := h.SenderName
	if sender == "" {
		sender = h.ChatTitle
	}
	snippet := h.Snippet
	runes := []rune(snippet)
	if len(runes) > 60 {
		snippet = string(runes[:60]) + "…"
	}
	date := h.Date.Format("01/02")
	return fmt.Sprintf("%s · %s: %s  (%s)", h.ChatTitle, sender, snippet, date)
}

// visibleWindow returns a [start, end) index range of size maxVisible
// centred on cursor within a list of total elements.
// Used by renderHitList to implement virtual scrolling.
func visibleWindow(cursor, total, maxVisible int) (int, int) {
	if total <= maxVisible {
		return 0, total
	}
	half := maxVisible / 2
	start := cursor - half
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}
	return start, end
}
