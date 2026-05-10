package model

import "fmt"

// Questo file contiene i formatter di supporto per MessageMedia.Summary().
// Separato per rispettare il limite 120 LOC.

// formatSize converte bytes in stringa human-readable con una cifra decimale.
// Esempi: 1234567 → "1.2 MB", 456789 → "446.1 KB", 12 → "12 B".
// Soglie: GB ≥ 1<<30, MB ≥ 1<<20, KB ≥ 1<<10, altrimenti B.
func formatSize(b int64) string {
	const (
		kb = 1 << 10
		mb = 1 << 20
		gb = 1 << 30
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// formatDuration converte secondi interi in "m:ss" (< 1h) o "h:mm:ss".
// 0 → "0:00". Usato da Voice e Video summary.
func formatDuration(secs int) string {
	if secs <= 0 {
		return "0:00"
	}
	h := secs / 3600
	m := (secs % 3600) / 60
	s := secs % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}
