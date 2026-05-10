package convert_test

import (
	"testing"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
)

func TestToMessageMedia_Nil(t *testing.T) {
	if got := convert.ToMessageMedia(nil); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestToMessageMedia_Empty(t *testing.T) {
	if got := convert.ToMessageMedia(&tg.MessageMediaEmpty{}); got != nil {
		t.Fatalf("expected nil for MessageMediaEmpty, got %+v", got)
	}
}

func TestToMessageMedia_WebPage(t *testing.T) {
	if got := convert.ToMessageMedia(&tg.MessageMediaWebPage{}); got != nil {
		t.Fatalf("expected nil for MessageMediaWebPage, got %+v", got)
	}
}

func TestToMessageMedia_Photo_NoPhotoObject(t *testing.T) {
	// Photo field is nil (PhotoEmpty) — should still return MediaPhoto with fallback name.
	got := convert.ToMessageMedia(&tg.MessageMediaPhoto{})
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Type != model.MediaPhoto {
		t.Fatalf("expected MediaPhoto, got %v", got.Type)
	}
	if got.FileName != "photo.jpg" {
		t.Fatalf("expected FileName=photo.jpg, got %q", got.FileName)
	}
}

func TestToMessageMedia_Photo_WithSize(t *testing.T) {
	photo := &tg.Photo{
		Sizes: []tg.PhotoSizeClass{
			&tg.PhotoSize{Type: "s", W: 90, H: 90, Size: 1000},
			&tg.PhotoSize{Type: "m", W: 320, H: 320, Size: 50000},
		},
	}
	got := convert.ToMessageMedia(&tg.MessageMediaPhoto{Photo: photo})
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Type != model.MediaPhoto {
		t.Fatalf("expected MediaPhoto, got %v", got.Type)
	}
	if got.SizeBytes != 50000 {
		t.Fatalf("expected SizeBytes=50000 (largest PhotoSize), got %d", got.SizeBytes)
	}
	if got.Width != 320 || got.Height != 320 {
		t.Fatalf("unexpected dimensions: %dx%d", got.Width, got.Height)
	}
}

func TestToMessageMedia_Location(t *testing.T) {
	cases := []tg.MessageMediaClass{
		&tg.MessageMediaGeo{},
		&tg.MessageMediaGeoLive{},
		&tg.MessageMediaVenue{},
	}
	for _, c := range cases {
		got := convert.ToMessageMedia(c)
		if got == nil || got.Type != model.MediaLocation {
			t.Fatalf("%T: expected MediaLocation, got %v", c, got)
		}
	}
}

func TestToMessageMedia_Contact(t *testing.T) {
	got := convert.ToMessageMedia(&tg.MessageMediaContact{})
	if got == nil || got.Type != model.MediaContact {
		t.Fatalf("expected MediaContact, got %v", got)
	}
}

func TestToMessageMedia_Poll(t *testing.T) {
	got := convert.ToMessageMedia(&tg.MessageMediaPoll{})
	if got == nil || got.Type != model.MediaPoll {
		t.Fatalf("expected MediaPoll, got %v", got)
	}
}
