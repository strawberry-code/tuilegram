package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// whichKeyState mirrors the TLA+ states: Idle / PrefixPending / Visible.
type whichKeyState int

const (
	wkIdle          whichKeyState = iota
	wkPrefixPending               // timer running, overlay NOT yet visible
	wkVisible                     // timer fired, overlay visible
)

// ContinuationEntry describes a single chord action.
type ContinuationEntry struct {
	Label   string         // human-readable description for the overlay
	Handler func() tea.Cmd // may be nil (no-op chord)
}

// WhichKeyModel manages the 300ms prefix-disambiguation state machine (ADR-015 §D5).
// Compile-time continuations registry (ADR-015 §D2). Value-receiver pattern.
type WhichKeyModel struct {
	state          whichKeyState
	activePrefix   string
	latestPrefixID uint64 // monotonic counter; never decremented (TLA+ MONOTONIC_PREFIXID)
	// continuations: prefix → (continuation key → entry). Compile-time only.
	continuations map[string]map[string]ContinuationEntry
	Width, Height int // set by root for overlay placement
}

// DefaultContinuations is the static which-key registry for Step 28 (ADR-015 §D2).
// Step 33: aggiunto "gx" (open link) al gruppo "g" (ADR-021 §DB4).
var DefaultContinuations = map[string]map[string]ContinuationEntry{
	"g": {
		"g": {Label: "top of list/viewport", Handler: func() tea.Cmd {
			return func() tea.Msg { return scrollTopMsg{} }
		}},
		"G": {Label: "bottom of list/viewport", Handler: func() tea.Cmd {
			return func() tea.Msg { return scrollBottomMsg{} }
		}},
		"u": {Label: "jump to next unread", Handler: func() tea.Cmd {
			return func() tea.Msg { return jumpUnreadMsg{} }
		}},
		"i": {Label: "chat info", Handler: func() tea.Cmd { return nil }},
		"x": {Label: "open link in selected msg", Handler: func() tea.Cmd {
			return func() tea.Msg { return openLinkChordMsg{} }
		}},
	},
	"z": {
		"z": {Label: "center current message", Handler: func() tea.Cmd {
			return func() tea.Msg { return scrollCenterMsg{} }
		}},
		"t": {Label: "scroll to top", Handler: func() tea.Cmd {
			return func() tea.Msg { return scrollTopMsg{} }
		}},
		"b": {Label: "scroll to bottom", Handler: func() tea.Cmd {
			return func() tea.Msg { return scrollBottomMsg{} }
		}},
	},
}

// NewWhichKeyModel returns an idle which-key model with the default continuations.
func NewWhichKeyModel() WhichKeyModel {
	return WhichKeyModel{
		state:         wkIdle,
		continuations: DefaultContinuations,
	}
}

// IsPrefixKey reports whether key is a registered prefix (g, z, …).
func (m WhichKeyModel) IsPrefixKey(key string) bool {
	_, ok := m.continuations[key]
	return ok
}

// IsVisible reports whether the which-key overlay is currently shown.
// Used by root View() to decide whether to render the overlay.
func (m WhichKeyModel) IsVisible() bool {
	return m.state == wkVisible
}

// IsPending reports whether a prefix is pending (timer running, no overlay yet).
func (m WhichKeyModel) IsPending() bool {
	return m.state == wkPrefixPending
}

// ActivePrefix returns the current active prefix key, or "".
func (m WhichKeyModel) ActivePrefix() string {
	return m.activePrefix
}
