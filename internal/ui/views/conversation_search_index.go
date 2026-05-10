package views

import (
	"strings"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// buildSearchIndex costruisce l'indice da una slice di messaggi.
// Esclude system messages (IsService) e messaggi senza testo (invariante SYSTEM_NOT_INDEXED).
// textLC è pre-lowercased per evitare allocazioni ad ogni keystroke.
func buildSearchIndex(msgs []model.Message) []IndexedMessage {
	idx := make([]IndexedMessage, 0, len(msgs))
	for i, m := range msgs {
		if m.IsService || m.Text == "" {
			continue
		}
		idx = append(idx, IndexedMessage{
			MsgID:  m.ID,
			TextLC: strings.ToLower(m.Text),
			Pos:    i,
		})
	}
	return idx
}

// computeMatches esegue la ricerca substring case-insensitive sull'indice.
// Ritorna matches in ordine cronologico (stessa order dell'index).
// Complessità: O(N * |textLC|) per ogni keystroke; accettabile per N <= 1000.
// qLC deve essere già lowercased dal caller (una sola volta per keystroke).
func computeMatches(idx []IndexedMessage, qLC string) []SearchMatch {
	if qLC == "" {
		return nil
	}
	out := make([]SearchMatch, 0)
	for _, im := range idx {
		spans := allOccurrences(im.TextLC, qLC)
		if len(spans) > 0 {
			out = append(out, SearchMatch{MsgID: im.MsgID, Spans: spans})
		}
	}
	return out
}

// allOccurrences trova tutte le posizioni (byte offset) di sub in text.
// Non overlapping; usa strings.Index ripetuto. O(|text|) per occorrenza.
func allOccurrences(text, sub string) []TextSpan {
	var spans []TextSpan
	offset := 0
	for {
		i := strings.Index(text[offset:], sub)
		if i < 0 {
			break
		}
		start := offset + i
		end := start + len(sub)
		spans = append(spans, TextSpan{Start: start, End: end})
		offset = end
		if offset >= len(text) {
			break
		}
	}
	return spans
}

// appendIndexEntry aggiunge un nuovo messaggio all'indice (re-index incrementale
// per NewMessageMsg). No-op se il messaggio è service o ha testo vuoto.
func appendIndexEntry(idx []IndexedMessage, m model.Message, pos int) []IndexedMessage {
	if m.IsService || m.Text == "" {
		return idx
	}
	return append(idx, IndexedMessage{
		MsgID:  m.ID,
		TextLC: strings.ToLower(m.Text),
		Pos:    pos,
	})
}
