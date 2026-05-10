package components

import (
	"sort"

	"github.com/strawberry-code/tuilegram/internal/model"
)

type rankedChat struct {
	chat  model.Chat
	score int
	orig  int // original index for stable tie-break
}

// FuzzyRank filters and ranks chats by query. Empty query returns chats unchanged.
// Scoring: prefix match (1000+), contiguous substring (500+), subsequence (1-100).
// No match → excluded. Case-insensitive, Unicode NFC-normalized.
func FuzzyRank(chats []model.Chat, query string) []model.Chat {
	if query == "" {
		return chats
	}

	ranked := make([]rankedChat, 0, len(chats))
	for i, c := range chats {
		score := bestScore(c.Title, query)
		if score == 0 {
			continue
		}
		ranked = append(ranked, rankedChat{chat: c, score: score, orig: i})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].orig < ranked[j].orig
	})

	result := make([]model.Chat, len(ranked))
	for i, r := range ranked {
		result[i] = r.chat
	}
	return result
}

// bestScore returns the highest score across all scoring strategies.
// Returns 0 if no strategy matches.
func bestScore(title, query string) int {
	if s := scorePrefix(title, query); s > 0 {
		return s
	}
	if s := scoreSubstring(title, query); s > 0 {
		return s
	}
	return scoreSubsequence(title, query)
}
