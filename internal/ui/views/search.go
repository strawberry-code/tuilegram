package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// SearchState mirrors the Open sub-states in the search statechart.
type SearchState int

const (
	SearchStateIdle      SearchState = iota // query empty, no results
	SearchStateTyping                       // query non-empty, debounce pending
	SearchStateSearching                    // RPC in flight
	SearchStateResults                      // RPC returned hits
	SearchStateEmpty                        // RPC returned zero hits
	SearchStateError                        // RPC returned error
)

// SearchModel is the global-search overlay (Step 26).
// Value-receiver pattern: callers own state via returned copies.
// Bridge must be assigned by the root before any cmd is fired.
type SearchModel struct {
	Active        bool
	Query         string
	QueryID       uint64 // monotonic token; bumped on every keystroke + Close
	input         textinput.Model
	Hits          []model.SearchHit
	cursor        int // index of selected hit
	state         SearchState
	err           error
	Width, Height int
	Bridge        *telegram.Bridge
}

// NewSearchModel returns an inactive, zero-state SearchModel.
func NewSearchModel() SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search messages…"
	ti.CharLimit = 256
	return SearchModel{input: ti}
}

// Open transitions Closed → Idle: resets state and focuses the input.
// QueryID is NOT reset here; Close() already bumped it to invalidate
// any in-flight RPC from the previous session (ADR-013).
func (m SearchModel) Open() SearchModel {
	m.Active = true
	m.Query = ""
	m.Hits = nil
	m.cursor = 0
	m.state = SearchStateIdle
	m.err = nil
	m.input.Reset()
	m.input.Focus()
	return m
}

// Close transitions any Open sub-state → Closed.
// Bumps QueryID to invalidate any in-flight RPC result (ADR-013).
func (m SearchModel) Close() SearchModel {
	m.Active = false
	m.QueryID++ // stale-result invalidation (ADR-013)
	m.input.Blur()
	return m
}

// Update dispatches tea.Msg to the appropriate handler.
// Returns early if overlay is not Active.
func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	if !m.Active {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	case SearchDebounceFiredMsg:
		return m.handleDebounce(msg)
	case SearchResultMsg:
		return m.handleResult(msg)
	case SearchErrMsg:
		return m.handleErr(msg)
	}
	// Propagate other msgs (e.g. cursor blink) to the textinput.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the modal canvas when Active; returns "" otherwise.
func (m SearchModel) View() string {
	if !m.Active {
		return ""
	}
	return renderSearchModal(m)
}
