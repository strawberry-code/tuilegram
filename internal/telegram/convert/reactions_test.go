package convert_test

import (
	"testing"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/telegram/convert"
)

func TestFlattenReactions_EmptyResults(t *testing.T) {
	if got := convert.FlattenReactions(tg.MessageReactions{}); got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestFlattenReactions_AllCustomEmoji_ReturnsNil(t *testing.T) {
	r := tg.MessageReactions{
		Results: []tg.ReactionCount{
			{Reaction: &tg.ReactionCustomEmoji{DocumentID: 1}, Count: 5},
			{Reaction: &tg.ReactionCustomEmoji{DocumentID: 2}, Count: 3},
		},
	}
	if got := convert.FlattenReactions(r); got != nil {
		t.Fatalf("expected nil when all reactions are custom emoji, got %+v", got)
	}
}

func TestFlattenReactions_SkipsEmpty(t *testing.T) {
	r := tg.MessageReactions{
		Results: []tg.ReactionCount{
			{Reaction: &tg.ReactionEmpty{}, Count: 1},
			{Reaction: &tg.ReactionEmoji{Emoticon: "👍"}, Count: 2},
		},
	}
	got := convert.FlattenReactions(r)
	if len(got) != 1 || got[0].Emoji != "👍" || got[0].Count != 2 {
		t.Fatalf("expected single emoji entry, got %+v", got)
	}
}

func TestFlattenReactions_OrderingByCountDesc(t *testing.T) {
	r := tg.MessageReactions{
		Results: []tg.ReactionCount{
			{Reaction: &tg.ReactionEmoji{Emoticon: "😂"}, Count: 1},
			{Reaction: &tg.ReactionEmoji{Emoticon: "👍"}, Count: 5},
			{Reaction: &tg.ReactionEmoji{Emoticon: "❤️"}, Count: 3},
		},
	}
	got := convert.FlattenReactions(r)
	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(got))
	}
	if got[0].Emoji != "👍" || got[1].Emoji != "❤️" || got[2].Emoji != "😂" {
		t.Fatalf("expected count-desc order 👍/❤️/😂, got %+v", got)
	}
}

func TestFlattenReactions_TieBreakByEmojiAsc(t *testing.T) {
	r := tg.MessageReactions{
		Results: []tg.ReactionCount{
			{Reaction: &tg.ReactionEmoji{Emoticon: "🍔"}, Count: 2},
			{Reaction: &tg.ReactionEmoji{Emoticon: "🍎"}, Count: 2},
			{Reaction: &tg.ReactionEmoji{Emoticon: "🍌"}, Count: 2},
		},
	}
	got := convert.FlattenReactions(r)
	if got[0].Emoji >= got[1].Emoji || got[1].Emoji >= got[2].Emoji {
		t.Fatalf("tie-break must be emoji asc, got %+v", got)
	}
}

func TestFlattenReactions_ChosenByMePropagated(t *testing.T) {
	one := 0
	r := tg.MessageReactions{
		Results: []tg.ReactionCount{
			{Reaction: &tg.ReactionEmoji{Emoticon: "🔥"}, Count: 1, ChosenOrder: one, Flags: 1},
		},
	}
	got := convert.FlattenReactions(r)
	if len(got) != 1 || !got[0].ChosenByMe {
		t.Fatalf("expected ChosenByMe=true, got %+v", got)
	}
}
