package telegram

import (
	"context"

	"github.com/gotd/td/tg"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// FoldersLoadedMsg carries the server-side DialogFilter list (Step 29, ADR-016 §D1).
type FoldersLoadedMsg struct {
	Folders []model.ChatFolder
}

// FoldersErrMsg reports a failure to fetch dialog filters.
type FoldersErrMsg struct{ Err error }

// LoadFolders fetches Telegram DialogFilter list via messages.getDialogFilters.
// Returns an empty slice (not an error) if the account has no custom folders.
func (b *Bridge) LoadFolders(ctx context.Context) ([]model.ChatFolder, error) {
	result, err := b.api.MessagesGetDialogFilters(ctx)
	if err != nil {
		return nil, err
	}
	return convertFolders(result.Filters), nil
}

// convertFolders maps []tg.DialogFilterClass → []model.ChatFolder.
// DialogFilterDefault (built-in "All Chats") is skipped; the UI sentinel covers it.
func convertFolders(filters []tg.DialogFilterClass) []model.ChatFolder {
	out := make([]model.ChatFolder, 0, len(filters))
	for _, f := range filters {
		df, ok := f.(*tg.DialogFilter)
		if !ok {
			continue // skip DialogFilterDefault and DialogFilterChatlist
		}
		folder := model.ChatFolder{
			ID:    df.ID,
			Title: df.Title.Text, // TextWithEntities.Text is the plain string
		}
		for _, peer := range df.IncludePeers {
			if cid := peerToChat(peer); cid != (model.ChatID{}) {
				folder.IncludedChats = append(folder.IncludedChats, cid)
			}
		}
		out = append(out, folder)
	}
	return out
}

// peerToChat converts an InputPeerClass to a model.ChatID.
func peerToChat(peer tg.InputPeerClass) model.ChatID {
	switch p := peer.(type) {
	case *tg.InputPeerUser:
		return model.ChatID{PeerType: model.PeerUser, ID: p.UserID}
	case *tg.InputPeerChat:
		return model.ChatID{PeerType: model.PeerChat, ID: p.ChatID}
	case *tg.InputPeerChannel:
		return model.ChatID{PeerType: model.PeerChannel, ID: p.ChannelID}
	}
	return model.ChatID{}
}
