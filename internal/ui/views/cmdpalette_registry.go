package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// CommandEntry describes a single palette command (ADR-015 §D2 — static, compile-time).
// Handler is a factory: returns a tea.Cmd (or nil for pure-UI sync actions).
// Section groups commands in the help overlay.
type CommandEntry struct {
	ID      string         // stable identifier, e.g. "chat.mute"
	Title   string         // human-readable, e.g. "Mute current chat"
	Keys    []string       // keybinding hint shown in help, e.g. ["m"]
	Section string         // help group, e.g. "Chat"
	Handler func() tea.Cmd // execution callback; nil → no-op
}

// CommandRegistry is a compile-time-ordered list of commands.
// Order = canonical display order in palette (ShowingAll state).
// Lookup is by ID for Submit; filter is by Title for fuzzy match.
type CommandRegistry []CommandEntry

// DefaultRegistry is the static command registry for Step 28.
// Handlers that require context (active chat, selection) return nil-cmd
// gracefully when the context is unavailable. No panic.
// NOTE: Scroll actions emit tea.Msgs processed by the conversation viewport.
var DefaultRegistry = CommandRegistry{
	// Navigation
	{
		ID: "nav.scroll_top", Title: "Scroll to top",
		Keys: []string{"gg"}, Section: "Navigation",
		Handler: func() tea.Cmd {
			return func() tea.Msg { return scrollTopMsg{} }
		},
	},
	{
		ID: "nav.scroll_bottom", Title: "Scroll to bottom",
		Keys: []string{"G"}, Section: "Navigation",
		Handler: func() tea.Cmd {
			return func() tea.Msg { return scrollBottomMsg{} }
		},
	},
	{
		ID: "nav.scroll_center", Title: "Center current message",
		Keys: []string{"zz"}, Section: "Navigation",
		Handler: func() tea.Cmd {
			return func() tea.Msg { return scrollCenterMsg{} }
		},
	},
	{
		ID: "nav.jump_unread", Title: "Jump to next unread chat",
		Keys: []string{"gu"}, Section: "Navigation",
		Handler: func() tea.Cmd {
			return func() tea.Msg { return jumpUnreadMsg{} }
		},
	},
	// Search
	{
		ID: "search.global", Title: "Search messages globally",
		Keys: []string{"/"}, Section: "Search",
		Handler: func() tea.Cmd {
			return func() tea.Msg { return SearchOpenMsg{} }
		},
	},
	{
		ID: "search.in_chat", Title: "Search in current chat",
		Keys: []string{"ctrl+f"}, Section: "Search",
		Handler: func() tea.Cmd { return nil }, // handled inline by Ctrl+F
	},
	// Help
	{
		ID: "ui.help", Title: "Show keybindings help",
		Keys: []string{"?"}, Section: "UI",
		Handler: func() tea.Cmd {
			return func() tea.Msg { return HelpOpenMsg{} }
		},
	},
	// Chat actions (require active chat — handlers emit msgs; no-ops gracefully)
	{
		ID: "chat.reply", Title: "Reply to selected message",
		Keys: []string{"r"}, Section: "Chat",
		Handler: func() tea.Cmd { return nil },
	},
	{
		ID: "chat.edit", Title: "Edit selected message",
		Keys: []string{"e"}, Section: "Chat",
		Handler: func() tea.Cmd { return nil },
	},
	{
		ID: "chat.delete", Title: "Delete selected message",
		Keys: []string{"D"}, Section: "Chat",
		Handler: func() tea.Cmd { return nil },
	},
	{
		ID: "chat.forward", Title: "Forward selected message",
		Keys: []string{"f"}, Section: "Chat",
		Handler: func() tea.Cmd { return nil },
	},
	// Account
	{
		ID: "account.logout", Title: "Logout (sign out + remove session)",
		Section: "Account",
		Handler: func() tea.Cmd {
			return func() tea.Msg { return LogoutRequestMsg{} }
		},
	},
}

// LogoutRequestMsg richiede sign-out + cleanup session. Handled in main_update.
type LogoutRequestMsg struct{}

// scrollTopMsg, scrollBottomMsg, scrollCenterMsg, jumpUnreadMsg are internal
// navigation messages dispatched from palette/whichkey handlers.
// Handled in main_update.go → conversation.Update.
type scrollTopMsg struct{}
type scrollBottomMsg struct{}
type scrollCenterMsg struct{}
type jumpUnreadMsg struct{}
