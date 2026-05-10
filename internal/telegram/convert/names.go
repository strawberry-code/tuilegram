// Package convert — names.go
// Shared helpers for resolving int64 user IDs against a names map.
//
// Two contracts:
//   - LookupName: total predicate ((string, bool)). Caller decides fallback.
//   - ResolveName: convenience wrapper returning "Unknown" when missing.
//
// Spec: docs/design/phase-5-data/entity-mapping.md §Name Resolution
package convert

// LookupName reports whether id maps to a non-empty display name.
// Empty strings are treated as missing (defensive against partial entity data).
func LookupName(names map[int64]string, id int64) (string, bool) {
	if n, ok := names[id]; ok && n != "" {
		return n, true
	}
	return "", false
}

// ResolveName returns the display name for id, or "Unknown" when absent.
// Use this when the caller has no domain-specific fallback to apply.
func ResolveName(names map[int64]string, id int64) string {
	if n, ok := LookupName(names, id); ok {
		return n
	}
	return "Unknown"
}
