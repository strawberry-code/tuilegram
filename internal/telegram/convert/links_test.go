package convert

import (
	"testing"

	"github.com/gotd/td/tg"
)

// TestExtractLinks_Empty verifica che nil entities restituisca nil.
func TestExtractLinks_Empty(t *testing.T) {
	got := ExtractLinks("hello world", nil)
	if got != nil {
		t.Errorf("expected nil for no entities, got %v", got)
	}
}

// TestExtractLinks_URLEntity verifica estrazione di MessageEntityURL.
func TestExtractLinks_URLEntity(t *testing.T) {
	text := "Visit https://example.com now"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityURL{Offset: 6, Length: 19},
	}
	got := ExtractLinks(text, entities)
	if len(got) != 1 {
		t.Fatalf("expected 1 link, got %d", len(got))
	}
	if got[0].URL != "https://example.com" {
		t.Errorf("expected URL https://example.com, got %q", got[0].URL)
	}
}

// TestExtractLinks_TextURLEntity verifica estrazione di MessageEntityTextURL.
func TestExtractLinks_TextURLEntity(t *testing.T) {
	text := "Click here for more"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityTextURL{Offset: 6, Length: 4, URL: "https://example.com/more"},
	}
	got := ExtractLinks(text, entities)
	if len(got) != 1 {
		t.Fatalf("expected 1 link, got %d", len(got))
	}
	if got[0].URL != "https://example.com/more" {
		t.Errorf("expected URL from TextURL entity, got %q", got[0].URL)
	}
}

// TestExtractLinks_NonHTTPFiltered verifica che scheme non http(s) siano esclusi.
func TestExtractLinks_NonHTTPFiltered(t *testing.T) {
	text := "Open tg://resolve?domain=test"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityTextURL{Offset: 5, Length: 24, URL: "tg://resolve?domain=test"},
	}
	got := ExtractLinks(text, entities)
	if len(got) != 0 {
		t.Errorf("non-http scheme should be filtered out, got %v", got)
	}
}

// TestExtractLinks_MentionIgnored verifica che mention entity sia ignorata.
func TestExtractLinks_MentionIgnored(t *testing.T) {
	text := "Hello @alice how are you"
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityMention{Offset: 6, Length: 6},
	}
	got := ExtractLinks(text, entities)
	if len(got) != 0 {
		t.Errorf("mention should not be extracted as link, got %v", got)
	}
}

// TestExtractLinks_EmojiBeforeURL verifica correttezza UTF-16 con emoji prima dell'URL.
// "👋 " è 1 runa ma 2 UTF-16 code units (surrogate pair); Offset=3 è UTF-16.
// L'emoji occupa code units 0-1, spazio occupa code unit 2, URL inizia a 3.
func TestExtractLinks_EmojiBeforeURL(t *testing.T) {
	text := "👋 https://example.com"
	// "👋" = 2 UTF-16 units; " " = 1 unit → URL inizia a offset=3 UTF-16.
	entities := []tg.MessageEntityClass{
		&tg.MessageEntityURL{Offset: 3, Length: 19},
	}
	got := ExtractLinks(text, entities)
	if len(got) != 1 {
		t.Fatalf("expected 1 link, got %d", len(got))
	}
	if got[0].URL != "https://example.com" {
		t.Errorf("expected https://example.com, got %q", got[0].URL)
	}
}
