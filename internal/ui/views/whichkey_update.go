package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// PressPrefix handles a registered prefix key press.
// Guard (enforced by caller): activeOverlay == none AND state == Idle.
// Returns (updated model, tick cmd). latestPrefixID is bumped (TLA+ MONOTONIC_PREFIXID).
func (m WhichKeyModel) PressPrefix(key string) (WhichKeyModel, tea.Cmd) {
	m.latestPrefixID++
	id := m.latestPrefixID
	m.activePrefix = key
	m.state = wkPrefixPending
	return m, whichKeyTickCmd(key, id)
}

// HandleTimeout processes a WhichKeyTimeoutMsg (300ms tick).
// Stale-tick check: if msg.PrefixID != latestPrefixID → benign no-op
// (invariant STALE_TICK_BENIGN_WHICHKEY in whichkey.tla).
// Returns (updated model, nil). Overlay becomes visible only on fresh tick.
func (m WhichKeyModel) HandleTimeout(msg WhichKeyTimeoutMsg) WhichKeyModel {
	if msg.PrefixID != m.latestPrefixID {
		return m // stale tick — no-op
	}
	if m.state != wkPrefixPending {
		return m // already resolved or cancelled
	}
	m.state = wkVisible
	return m
}

// HandleContinuation resolves a chord (continuation key in registry).
// Valid in both PrefixPending (fast chord) and Visible states.
// Bumps latestPrefixID to invalidate any pending tick (FAST_CHORD_NO_OVERLAY).
// Returns (updated model, handler cmd or nil).
func (m WhichKeyModel) HandleContinuation(prefix, cont string) (WhichKeyModel, tea.Cmd) {
	conts, ok := m.continuations[prefix]
	if !ok {
		return m.Cancel(""), nil
	}
	entry, ok := conts[cont]
	if !ok {
		return m.Cancel(cont), nil // unknown continuation → cancel + best-effort re-dispatch
	}
	m.latestPrefixID++
	m.activePrefix = ""
	m.state = wkIdle
	if entry.Handler == nil {
		return m, nil
	}
	return m, entry.Handler()
}

// Cancel cancels the current prefix (Esc or unknown key).
// Bumps latestPrefixID to invalidate any pending tick.
// unknownKey is passed for best-effort re-dispatch (advisory, ADR-015 §D6).
// Returns (updated model) — the caller emits WhichKeyCancelMsg.
func (m WhichKeyModel) Cancel(unknownKey string) WhichKeyModel {
	_ = unknownKey // best-effort re-dispatch handled by the root Update switch
	m.latestPrefixID++
	m.activePrefix = ""
	m.state = wkIdle
	return m
}
