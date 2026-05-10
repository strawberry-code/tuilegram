package views

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// SearchOpenMsg triggers the global search overlay (keybinding: '/').
type SearchOpenMsg struct{}

// SearchDebounceFiredMsg is produced by the 1s debounce tick.
// QueryID is compared against SearchModel.QueryID to discard stale ticks.
type SearchDebounceFiredMsg struct{ QueryID uint64 }

// SearchResultMsg carries results from a SearchGlobal RPC.
// QueryID must match SearchModel.QueryID; otherwise dropped (ADR-013).
type SearchResultMsg struct {
	QueryID uint64
	Hits    []model.SearchHit
}

// SearchErrMsg carries an RPC error for the given query token.
type SearchErrMsg struct {
	QueryID uint64
	Err     error
}

// JumpToMessageMsg requests navigation to a specific message in a chat.
// Emitted by the search overlay on Enter; handled by MainModel.
type JumpToMessageMsg struct {
	ChatID    model.ChatID
	MessageID int
}

// searchDebounceCmd fires SearchDebounceFiredMsg after 1s.
// The tick carries the queryID so stale ticks can be dropped (ADR-013).
// 1s deliberately conservative: low traffic, low flood-wait risk; users
// expect a brief pause before search fires (less reactive than a 300ms
// "live" feel, but matches "blur to commit" UX patterns).
func searchDebounceCmd(queryID uint64) tea.Cmd {
	return tea.Tick(1*time.Second, func(time.Time) tea.Msg {
		return SearchDebounceFiredMsg{QueryID: queryID}
	})
}

// searchExecuteCmd runs SearchGlobal in a tea.Cmd goroutine and returns
// SearchResultMsg or SearchErrMsg with the matching queryID.
func searchExecuteCmd(bridge *telegram.Bridge, query string, queryID uint64) tea.Cmd {
	return func() tea.Msg {
		res, err := bridge.SearchGlobal(context.Background(), query, 30, queryID)
		if err != nil {
			return SearchErrMsg{QueryID: queryID, Err: err}
		}
		return SearchResultMsg{QueryID: res.QueryID, Hits: res.Hits}
	}
}
