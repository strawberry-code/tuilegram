package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Step 28 — Command Palette tea.Msg types (message-taxonomy.md §Internal UI Messages).

// CmdPaletteOpenMsg triggers palette open (Ctrl+P).
// Guard: activeOverlay == none (enforced in handleKeyMsg).
type CmdPaletteOpenMsg struct{}

// CmdPaletteSubmitMsg is emitted on Enter when len(filtered) > 0.
// Handling: activeOverlay := none FIRST, then dispatch handler (ADR-015 §D3).
type CmdPaletteSubmitMsg struct{ CmdID string }

// CmdPaletteCloseMsg is emitted on Esc in the palette.
type CmdPaletteCloseMsg struct{}

// Step 28 — Which-Key tea.Msg types.

// WhichKeyTimeoutMsg is the deferred result of the 300ms tea.Tick.
// Stale-tick check: if PrefixID != latestPrefixID → benign no-op (ADR-015 §D5,
// invariant STALE_TICK_BENIGN_WHICHKEY in whichkey.tla).
type WhichKeyTimeoutMsg struct {
	Prefix   string
	PrefixID uint64
}

// WhichKeyChordMsg is emitted when the continuation key arrives (in PrefixPending
// OR WhichKeyVisible). Bumps latestPrefixID; resolves chord immediately.
type WhichKeyChordMsg struct {
	Prefix string
	Cont   string
}

// WhichKeyCancelMsg is emitted on Esc or unknown key during PrefixPending/Visible.
// Bumps latestPrefixID to invalidate any pending tick.
type WhichKeyCancelMsg struct{ Prefix string }

// Step 28 — Help Overlay tea.Msg types.

// HelpOpenMsg triggers the help overlay (?).
// Guard: activeOverlay == none.
type HelpOpenMsg struct{}

// HelpCloseMsg closes the help overlay (Esc or ?).
type HelpCloseMsg struct{}

// whichKeyTickCmd schedules a WhichKeyTimeoutMsg after 300ms (ADR-015 §D5).
// The prefixID is embedded in the closure so stale ticks can be detected at fire-time.
func whichKeyTickCmd(prefix string, prefixID uint64) tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(time.Time) tea.Msg {
		return WhichKeyTimeoutMsg{Prefix: prefix, PrefixID: prefixID}
	})
}
