package views

import (
	"strings"
	"testing"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// TestRenderTextWithLinks_NoLinks verifica che testo senza link sia invariato.
func TestRenderTextWithLinks_NoLinks(t *testing.T) {
	text := "Hello world"
	got := renderTextWithLinks(text, nil)
	if got != text {
		t.Errorf("expected %q, got %q", text, got)
	}
}

// TestRenderTextWithLinks_OSC8Present verifica che OSC 8 escape sia inserita.
func TestRenderTextWithLinks_OSC8Present(t *testing.T) {
	text := "Visit https://example.com now"
	links := []model.MessageLink{
		{Offset: 6, Length: 19, URL: "https://example.com"},
	}
	got := renderTextWithLinks(text, links)
	// OSC 8 prefix deve essere presente.
	if !strings.Contains(got, "\x1b]8;;") {
		t.Error("OSC 8 escape sequence not found in rendered link")
	}
	// L'URL deve essere nell'escape.
	if !strings.Contains(got, "https://example.com") {
		t.Error("URL not found in rendered output")
	}
}

// TestOSC8Wrap verifica il formato corretto dell'escape OSC 8.
func TestOSC8Wrap(t *testing.T) {
	result := osc8Wrap("https://example.com", "click here")
	// Deve iniziare con ESC]8;;url ESC\ text ESC]8;; ESC\
	if !strings.HasPrefix(result, "\x1b]8;;https://example.com\x1b\\") {
		t.Errorf("OSC 8 prefix malformed: %q", result[:min(len(result), 40)])
	}
	if !strings.Contains(result, "click here") {
		t.Error("link text not present in OSC 8 wrapped output")
	}
}

// TestRenderTextWithLinks_TextPreserved verifica che il testo fuori dai link sia presente nell'output.
func TestRenderTextWithLinks_TextPreserved(t *testing.T) {
	text := "before https://x.com after"
	links := []model.MessageLink{
		{Offset: 7, Length: 14, URL: "https://x.com"},
	}
	got := renderTextWithLinks(text, links)
	// "before" deve apparire nell'output (testo prima del link).
	if !strings.Contains(got, "before ") {
		t.Errorf("prefix 'before' not found in output: %q", got[:min(len(got), 30)])
	}
	// "after" deve apparire nell'output (testo dopo il link).
	if !strings.Contains(got, "after") {
		t.Errorf("suffix 'after' not found in output")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
