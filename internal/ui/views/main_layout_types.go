package views

// main_layout_types.go — LayoutMode / CompactPanel enums and pure derivation
// functions for the responsive layout introduced in Step 30 (ADR-018).
// These types are UI-internal (no domain dependency).

// compactThreshold is the column count below which layout flips to Compact.
// Default per ADR-018 §D1. Overridden at boot via config.toml [display] compact_threshold
// (ADR-019 D6). The active threshold lives in MainModel.compactThreshold.
const defaultCompactThreshold = 100

// LayoutMode is the active layout regime derived deterministically from
// WindowSizeMsg.Width (ADR-018 §D2 — no hysteresis, single threshold).
type LayoutMode int

const (
	// LayoutWide is the default two-panel (or three-panel with sidebar) layout.
	// Active when Width >= compactThreshold.
	LayoutWide LayoutMode = iota
	// LayoutCompact is the single-panel layout for narrow terminals.
	// Active when Width < compactThreshold.
	LayoutCompact
)

// CompactPanel identifies which single panel is visible in LayoutCompact.
// Meaningful only when layoutMode == LayoutCompact; ignored by the renderer
// in LayoutWide. Re-derived on each Wide→Compact transition (ADR-018 §D4).
type CompactPanel int

const (
	// CompactChatList shows the chat list full-width.
	CompactChatList CompactPanel = iota
	// CompactConversation shows the conversation (header + viewport + input) full-width.
	CompactConversation
)

// computeLayoutMode is the total, pure function that maps terminal width to
// LayoutMode (THRESHOLD_DETERMINISTIC invariant in responsive_layout.tla).
// threshold is from MainModel.compactThreshold (config-driven, ADR-019 D6).
func computeLayoutMode(width, threshold int) LayoutMode {
	if width < threshold {
		return LayoutCompact
	}
	return LayoutWide
}

// deriveCompactVisible implements the collapse derivation rule (ADR-018 §D4):
// preserve the user's current context if they were actively reading/composing
// a conversation; default to ChatList otherwise (safe fallback).
//
// Parameters reflect the *pre-collapse* state:
//   - activePanel: FocusChatList | FocusConversation | FocusFolders
//     (FocusConversation here subsumes the TLA+ sub-states {"Messages","Input"};
//     the inputFocus distinction lives inside ConversationModel, not FocusPanel.)
//   - chatActive:  true when a conversation is open (activeChatID != nil)
func deriveCompactVisible(activePanel FocusPanel, chatActive bool) CompactPanel {
	if chatActive && (activePanel == FocusConversation) {
		return CompactConversation
	}
	return CompactChatList
}
