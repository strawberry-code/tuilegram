package convert

import (
	"strings"

	"github.com/gotd/td/tg"
)

// ConvertFwdLabel traduce tg.MessageFwdHeader nella stringa displayable
// per il campo model.Message.ForwardedFrom.
// Priorità implementata (ADR-021 §DC3):
//  1. FromName esplicito (campo server-fornito, include username e nomi)
//  2. PostAuthor per channel post firmati
//  3. Fallback → "Hidden"
//
// Invariante FORWARD_LABEL_FALLBACK_CHAIN: mai stringa vuota.
// Nota: FromID (priorità 1-3 del design) non è risolvibile senza user store;
// il server include già FromName nei forward visibili, quindi FromName è il
// punto di ingresso corretto. labelFromPeer era dead code (sempre "").
func ConvertFwdLabel(fwd *tg.MessageFwdHeader) string {
	if fwd == nil {
		return ""
	}

	// Priorità 1: FromName esplicito (server lo include per forward visibili).
	if name := strings.TrimSpace(fwd.FromName); name != "" {
		return name
	}

	// Priorità 2: PostAuthor per channel post con firma dell'autore.
	if fwd.PostAuthor != "" {
		return fwd.PostAuthor
	}

	return "Hidden"
}
