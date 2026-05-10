package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleKey routes keyboard input while the search overlay is active.
// Esc, Enter, j/k are intercepted; all other keys are forwarded to the
// textinput and trigger query-change detection.
func (m SearchModel) handleKey(msg tea.KeyMsg) (SearchModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m.Close(), nil

	case "enter":
		if len(m.Hits) > 0 && m.cursor < len(m.Hits) {
			hit := m.Hits[m.cursor]
			// JumpToMessage handler in main_search_handlers owns the close
			// (avoids double-close race with OverlayCloseMsg handler).
			jumpCmd := func() tea.Msg {
				return JumpToMessageMsg{ChatID: hit.ChatID, MessageID: hit.MessageID}
			}
			return m, jumpCmd
		}
		return m, nil

	case "down":
		if m.cursor < len(m.Hits)-1 {
			m.cursor++
		}
		return m, nil

	case "up":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	}

	return m.forwardToInput(msg)
}

// forwardToInput delivers the key to the textinput and detects query changes.
// On change: bumps QueryID, sets state to Typing, arms the debounce tick.
// Empty query after deletion transitions back to Idle (no RPC).
func (m SearchModel) forwardToInput(msg tea.KeyMsg) (SearchModel, tea.Cmd) {
	prevQuery := m.Query
	var tiCmd tea.Cmd
	m.input, tiCmd = m.input.Update(msg)
	m.Query = m.input.Value()

	if m.Query == prevQuery {
		return m, tiCmd
	}

	m.QueryID++
	if m.Query == "" {
		m.state = SearchStateIdle
		m.Hits = nil
		m.cursor = 0
		return m, tiCmd
	}
	m.state = SearchStateTyping
	return m, tea.Batch(tiCmd, searchDebounceCmd(m.QueryID))
}

// handleDebounce processes a SearchDebounceFiredMsg.
// Stale ticks (QueryID mismatch) are silently dropped (ADR-013).
// Empty query after debounce clears results without firing an RPC.
func (m SearchModel) handleDebounce(msg SearchDebounceFiredMsg) (SearchModel, tea.Cmd) {
	if msg.QueryID != m.QueryID {
		return m, nil // stale tick (ADR-013)
	}
	if m.Query == "" {
		m.state = SearchStateIdle
		m.Hits = nil
		return m, nil
	}
	m.state = SearchStateSearching
	return m, searchExecuteCmd(m.Bridge, m.Query, m.QueryID)
}

// handleResult applies a SearchResultMsg when QueryID matches.
// Stale results (from superseded RPCs) are silently dropped (ADR-013).
func (m SearchModel) handleResult(msg SearchResultMsg) (SearchModel, tea.Cmd) {
	if msg.QueryID != m.QueryID {
		return m, nil // stale RPC result (ADR-013)
	}
	if len(msg.Hits) == 0 {
		m.state = SearchStateEmpty
		m.Hits = nil
	} else {
		m.state = SearchStateResults
		m.Hits = msg.Hits
		m.cursor = 0
	}
	return m, nil
}

// handleErr applies a SearchErrMsg when QueryID matches.
func (m SearchModel) handleErr(msg SearchErrMsg) (SearchModel, tea.Cmd) {
	if msg.QueryID != m.QueryID {
		return m, nil
	}
	m.state = SearchStateError
	m.err = msg.Err
	return m, nil
}
