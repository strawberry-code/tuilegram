package views

import "github.com/strawberry-code/tuilegram/internal/model"

// chatinfo_msgs.go — tea.Msg types for the chat info overlay (Step 29).
// Extends message-taxonomy.md §Internal UI Messages.

// ChatInfoOpenMsg is dispatched by the global 'i' key handler.
// Guards in root: activeOverlay == none && activeChatID != nil.
type ChatInfoOpenMsg struct{}

// ChatInfoCloseMsg is dispatched by Esc or 'i' while the overlay is open.
type ChatInfoCloseMsg struct{}

// ChatInfoCompletionMsg is returned by fetchFullUserCmd (lazy completion).
// Guard: if ChatID != chatInfoTarget → stale → no-op (ADR-017 §D2,
// invariant STALE_COMPLETION_DROP in folders_chatinfo.tla).
type ChatInfoCompletionMsg struct {
	ChatID  model.ChatID
	Bio     string // merged into chatInfoCard.Bio on match
	Partial bool   // true when RPC returned but bio is still empty (step 29 stub)
}

// ChatInfoCard is the data snapshot shown in the overlay.
// Built synchronously from cache on open; updated via ChatInfoCompletionMsg.
// -1 for counter fields means "unknown / stub [?]" (ADR-017 §D3).
type ChatInfoCard struct {
	Name             string
	Username         string // "" if not applicable
	OnlineStatus     string // "", "Online", "Last seen…" (private only)
	Phone            string // "" if hidden / not applicable
	Bio              string // "" → show "—"; "…loading" if in-flight
	SharedMediaCount int    // -1 = [?]
	SharedFilesCount int    // -1 = [?]
	SharedLinksCount int    // -1 = [?]
	ChatType         model.ChatType
}
