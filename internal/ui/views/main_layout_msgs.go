package views

// main_layout_msgs.go — tea.Msg types for responsive layout (Step 30).
// Extends message-taxonomy.md §Internal UI Messages (ADR-018).

// LayoutModeChangedMsg is synthesised in App.Update whenever WindowSizeMsg
// crosses the 100-col threshold (ADR-018 §D1). It is a fanout notification
// for sub-models; the primary trigger is the WindowSizeMsg itself.
// oldMode → newMode documents the direction of the transition.
type LayoutModeChangedMsg struct {
	OldMode LayoutMode
	NewMode LayoutMode
}

// LayoutPanelSwitchMsg is dispatched by Tab in Compact mode when no overlay
// is active (ADR-018 §D3, TAB_PRESERVES_LAYOUT invariant: Tab never flips
// layoutMode, it only toggles compactVisible).
// In Wide mode Tab emits FocusNextMsg (pre-existing focus cycle behaviour).
type LayoutPanelSwitchMsg struct{}
