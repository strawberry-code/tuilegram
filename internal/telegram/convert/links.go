package convert

import (
	"strings"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// ExtractLinks converte le entity Telegram di un messaggio in []model.MessageLink.
// Invariante LINK_DETECTION_AUTHORITATIVE: usa solo tg.Message.Entities, mai regex.
// Solo MessageEntityURL e MessageEntityTextURL sono inclusi (ADR-021 §DB1).
// Offset/Length sono UTF-16 code units (Telegram MTProto spec) — usa Utf16Slice.
func ExtractLinks(msgText string, entities []tg.MessageEntityClass) []model.MessageLink {
	if len(entities) == 0 {
		return nil
	}
	var links []model.MessageLink
	for _, e := range entities {
		switch ent := e.(type) {
		case *tg.MessageEntityURL:
			// URL è nel corpo del messaggio: estraiamo con offset UTF-16 corretto.
			url := Utf16Slice(msgText, ent.Offset, ent.Length)
			if isHTTP(url) {
				links = append(links, model.MessageLink{
					Offset: ent.Offset,
					Length: ent.Length,
					URL:    url,
				})
			}
		case *tg.MessageEntityTextURL:
			// Il testo visibile è nel corpo; l'URL reale è nel campo URL dell'entity.
			if isHTTP(ent.URL) {
				links = append(links, model.MessageLink{
					Offset: ent.Offset,
					Length: ent.Length,
					URL:    ent.URL,
				})
			}
		}
	}
	return links
}

// isHTTP controlla che l'URL abbia schema http:// o https://.
// Invariante LINK_OPEN_HTTP_ONLY (ADR-021 §DB6).
func isHTTP(url string) bool {
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
