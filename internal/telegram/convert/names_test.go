package convert_test

import (
	"testing"

	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
)

func TestLookupName_Hit(t *testing.T) {
	names := map[int64]string{42: "Alice"}
	got, ok := convert.LookupName(names, 42)
	if !ok || got != "Alice" {
		t.Fatalf("expected (\"Alice\", true), got (%q, %v)", got, ok)
	}
}

func TestLookupName_Miss(t *testing.T) {
	if _, ok := convert.LookupName(map[int64]string{}, 1); ok {
		t.Fatal("expected ok=false for missing id")
	}
}

func TestLookupName_EmptyValueTreatedAsMissing(t *testing.T) {
	names := map[int64]string{7: ""}
	if _, ok := convert.LookupName(names, 7); ok {
		t.Fatal("empty string must be reported as missing")
	}
}

func TestResolveName_Hit(t *testing.T) {
	names := map[int64]string{1: "Bob"}
	if got := convert.ResolveName(names, 1); got != "Bob" {
		t.Fatalf("expected Bob, got %q", got)
	}
}

func TestResolveName_FallbackUnknown(t *testing.T) {
	if got := convert.ResolveName(map[int64]string{}, 99); got != "Unknown" {
		t.Fatalf("expected Unknown fallback, got %q", got)
	}
}

func TestResolveName_NilMap(t *testing.T) {
	if got := convert.ResolveName(nil, 1); got != "Unknown" {
		t.Fatalf("expected Unknown for nil map, got %q", got)
	}
}
