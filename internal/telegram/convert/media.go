// Package convert maps gotd/td MTProto types to domain model types.
// All functions are pure: no I/O, no RPC, no side-effects.
// Spec: docs/design/phase-5-data/entity-mapping.md §Media Mapping
// Dispatch order: docs/design/phase-2-behavioral/media-rendering.md §Decision Tree
package convert

import (
	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// ToMessageMedia converts a tg.MessageMediaClass to *model.MessageMedia.
// Returns nil for nil input, MessageMediaEmpty, MessageMediaWebPage,
// MessageMediaUnsupported, and other out-of-scope variants (Invoice, Game, Dice).
// Guarantees: non-nil output has Type != 0 (invariant from model package).
func ToMessageMedia(m tg.MessageMediaClass) *model.MessageMedia {
	if m == nil {
		return nil
	}
	switch v := m.(type) {
	case *tg.MessageMediaPhoto:
		return convertPhoto(v)
	case *tg.MessageMediaDocument:
		return convertDocument(v)
	case *tg.MessageMediaGeo:
		return &model.MessageMedia{Type: model.MediaLocation}
	case *tg.MessageMediaGeoLive:
		return &model.MessageMedia{Type: model.MediaLocation}
	case *tg.MessageMediaVenue:
		return &model.MessageMedia{Type: model.MediaLocation}
	case *tg.MessageMediaContact:
		return &model.MessageMedia{Type: model.MediaContact}
	case *tg.MessageMediaPoll:
		return &model.MessageMedia{Type: model.MediaPoll}
	// Out-of-scope: nil result
	case *tg.MessageMediaEmpty,
		*tg.MessageMediaWebPage,
		*tg.MessageMediaUnsupported,
		*tg.MessageMediaInvoice,
		*tg.MessageMediaGame,
		*tg.MessageMediaDice:
		return nil
	default:
		// Unknown future variants: nil per invariant §WebPage-style.
		return nil
	}
}

// convertPhoto handles tg.MessageMediaPhoto → model.MediaPhoto.
// Extracts Width/Height from the largest PhotoSize if available.
// FileName is always "photo.jpg" (photos have no file name in MTProto).
func convertPhoto(v *tg.MessageMediaPhoto) *model.MessageMedia {
	out := &model.MessageMedia{
		Type:     model.MediaPhoto,
		FileName: "photo.jpg",
		MimeType: "image/jpeg",
	}
	photo, ok := v.Photo.(*tg.Photo)
	if !ok {
		return out
	}
	var bestW, bestH, bestSize int
	for _, s := range photo.Sizes {
		ps, ok := s.(*tg.PhotoSize)
		if !ok {
			continue
		}
		if ps.Size > bestSize {
			bestSize = ps.Size
			bestW = ps.W
			bestH = ps.H
		}
	}
	out.SizeBytes = int64(bestSize)
	out.Width = bestW
	out.Height = bestH
	return out
}

// convertDocument dispatches a tg.MessageMediaDocument to the correct MediaType
// following the attribute-priority cascade defined in ADR-011 §2.
// Always returns non-nil (worst case: MediaDocument fallback).
func convertDocument(v *tg.MessageMediaDocument) *model.MessageMedia {
	doc, ok := v.Document.(*tg.Document)
	if !ok {
		// DocumentEmpty or unexpected union variant.
		return &model.MessageMedia{Type: model.MediaDocument}
	}
	return dispatchDocument(doc)
}
