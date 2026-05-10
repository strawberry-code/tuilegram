package views

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
)

// LoadFoldersCmd fetches server-side DialogFilter list (ADR-016 §D1).
// Returns FoldersLoadedMsg or FoldersErrMsg.
func (m MainModel) LoadFoldersCmd() tea.Cmd {
	return func() tea.Msg {
		folders, err := m.Bridge.LoadFolders(context.Background())
		if err != nil {
			return telegram.FoldersErrMsg{Err: err}
		}
		return telegram.FoldersLoadedMsg{Folders: folders}
	}
}

// fetchFullUserCmd performs a best-effort lazy fetch of the user's full profile
// (bio field). Step 29 stub: returns ChatInfoCompletionMsg{Partial: true} on
// any error; on success merges the bio from UserFull.About.
//
// telegram-dev follow-up needed: expose Bridge.GetFullUser(ctx, chat) → (bio, err)
// using api.UsersGetFullUser for private chats. For Step 29 this is a stub
// that always returns Partial=true (no actual RPC). The cache-first path in
// chatinfo_model.go already shows "—" for empty bio, so UX is correct.
func fetchFullUserCmd(bridge *telegram.Bridge, chatID model.ChatID) tea.Cmd {
	return func() tea.Msg {
		// STUB: real implementation via telegram-dev in a follow-up step.
		// When Bridge.GetFullUser is available, call it here and return
		// ChatInfoCompletionMsg{ChatID: chatID, Bio: bio}.
		_ = bridge
		return ChatInfoCompletionMsg{ChatID: chatID, Partial: true}
	}
}
