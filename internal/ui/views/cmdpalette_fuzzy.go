package views

import (
	"sort"
	"strings"
)

// cmdFuzzyMatch implements the subsequence fuzzy matcher specified in
// command-palette-help-whichkey.md §"Algoritmo fuzzy match (subsequence)"
// and ADR-015 §D4 (in-house, zero deps, reuses scoring strategy from
// existing fuzzy_score.go patterns).
//
// Returns (score, matched). score > 0 only if matched == true.
// Scoring bonuses:
//   - +1 per char matched
//   - +5+streak for consecutive matches (streak grows each consecutive hit)
//   - +10 for match at word boundary (position 0 or after a space)
//
// Length penalty: -(len(title)-len(query))/4.
func cmdFuzzyMatch(title, query string) (int, bool) {
	if query == "" {
		return 0, true
	}
	qLC := strings.ToLower(query)
	tLC := strings.ToLower(title)
	tRunes := []rune(tLC)
	qRunes := []rune(qLC)
	qi := 0
	score := 0
	streak := 0
	lastMatchPos := -1
	for ti := 0; ti < len(tRunes) && qi < len(qRunes); ti++ {
		if tRunes[ti] == qRunes[qi] {
			qi++
			// Consecutive streak bonus (grows with each consecutive hit).
			if lastMatchPos == ti-1 {
				streak++
				score += 5 + streak
			} else {
				streak = 0
				score++
			}
			// Word-boundary bonus.
			if ti == 0 || tRunes[ti-1] == ' ' {
				score += 10
			}
			lastMatchPos = ti
		}
	}
	if qi < len(qRunes) {
		return 0, false // not all query chars found
	}
	// Length penalty: reduces score for very long titles relative to query.
	penalty := (len(tRunes) - len(qRunes)) / 4
	score -= penalty
	if score < 1 {
		score = 1
	}
	return score, true
}

// rankedEntry pairs a CommandEntry with its fuzzy score for sorting.
type rankedEntry struct {
	entry CommandEntry
	score int
}

// cmdFuzzyFilter filters and ranks registry entries by query.
// Empty query → returns all entries in canonical order (ADR-015 §D2).
// Non-empty query → subsequence match on Title, sorted by score DESC then Title ASC.
// Complexity: O(|registry| * |title_len|) — negligible for ≤50 entries.
func cmdFuzzyFilter(registry CommandRegistry, query string) []CommandEntry {
	if query == "" {
		result := make([]CommandEntry, len(registry))
		copy(result, registry)
		return result
	}
	ranked := make([]rankedEntry, 0, len(registry))
	for _, e := range registry {
		if score, ok := cmdFuzzyMatch(e.Title, query); ok {
			ranked = append(ranked, rankedEntry{entry: e, score: score})
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score > ranked[j].score
		}
		return ranked[i].entry.Title < ranked[j].entry.Title
	})
	out := make([]CommandEntry, len(ranked))
	for i, r := range ranked {
		out[i] = r.entry
	}
	return out
}
