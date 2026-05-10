package components

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// scorePrefix returns prefix-match score or 0. Case-insensitive, NFC-normalized.
// Score: 1000 + len(query) bonus.
func scorePrefix(title, query string) int {
	if strings.HasPrefix(foldNFC(title), foldNFC(query)) {
		return 1000 + utf8.RuneCountInString(query)
	}
	return 0
}

// scoreSubstring returns contiguous-substring score or 0.
// Score: 500 + position bonus (earlier = higher: maxLen - pos).
func scoreSubstring(title, query string) int {
	t, q := foldNFC(title), foldNFC(query)
	idx := strings.Index(t, q)
	if idx < 0 {
		return 0
	}
	// pos in runes for bonus: earlier position is better
	pos := utf8.RuneCountInString(t[:idx])
	maxLen := utf8.RuneCountInString(t)
	return 500 + max(0, maxLen-pos)
}

// scoreSubsequence returns subsequence score or 0.
// Score: 100 - gap penalty (number of skipped runes between matches).
func scoreSubsequence(title, query string) int {
	t, q := []rune(foldNFC(title)), []rune(foldNFC(query))
	ti, qi := 0, 0
	gaps := 0
	lastMatch := -1
	for ti < len(t) && qi < len(q) {
		if t[ti] == q[qi] {
			if lastMatch >= 0 {
				gaps += ti - lastMatch - 1
			}
			lastMatch = ti
			qi++
		}
		ti++
	}
	if qi < len(q) {
		return 0 // not all query chars found
	}
	score := 100 - gaps
	if score < 1 {
		score = 1
	}
	return score
}

// foldNFC normalizes s to NFC and folds to lower-case (Unicode-safe).
func foldNFC(s string) string {
	normalized := norm.NFC.String(s)
	return strings.Map(func(r rune) rune {
		return unicode.ToLower(r)
	}, normalized)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
