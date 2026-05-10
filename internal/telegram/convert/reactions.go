// Package convert — reactions.go
// Converts tg.MessageReactions → []model.Reaction (Step 25).
//
// Design decisions (ADR-012 A1):
//   - tg.ReactionCustomEmoji entries are skipped (Step 25 scope).
//   - Output slice is sorted: Count desc, Emoji asc as tie-break.
//   - Snapshot semantics: caller replaces m.Reactions entirely.
//
// Spec: docs/design/phase-5-data/entity-mapping.md §Reactions Mapping
// Invariant: REACTIONS_ORDERED (reactions.tla)
package convert

import (
	"sort"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// FlattenReactions converts a tg.MessageReactions snapshot to an ordered
// []model.Reaction slice. Returns nil if no standard emoji reactions exist.
//
// Filtering rules:
//   - *tg.ReactionEmoji     → included (standard emoji)
//   - *tg.ReactionCustomEmoji → skipped (premium; out of scope Step 25)
//   - *tg.ReactionEmpty     → skipped (server placeholder)
//
// Ordering: Count desc, then Emoji asc for deterministic tie-breaking.
// A non-deterministic ordering would cause render flicker when two reactions
// share the same count (REACTIONS_ORDERED invariant).
func FlattenReactions(r tg.MessageReactions) []model.Reaction {
	if len(r.Results) == 0 {
		return nil
	}
	out := make([]model.Reaction, 0, len(r.Results))
	for _, rc := range r.Results {
		emoji, ok := rc.Reaction.(*tg.ReactionEmoji)
		if !ok {
			// ReactionCustomEmoji or ReactionEmpty — skip.
			continue
		}
		_, chosen := rc.GetChosenOrder()
		out = append(out, model.Reaction{
			Emoji:      emoji.Emoticon,
			Count:      rc.Count,
			ChosenByMe: chosen,
		})
	}
	if len(out) == 0 {
		return nil
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count // desc
		}
		return out[i].Emoji < out[j].Emoji // asc tie-break
	})
	return out
}
