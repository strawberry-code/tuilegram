package views

import (
	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
	"github.com/strawberry-code/tuilegram/internal/ui/components"
)

const chatListRatio = 0.28

// FocusPanel identifica quale pannello ha il focus.
type FocusPanel int

const (
	FocusChatList FocusPanel = iota
	FocusConversation
	FocusFolders // Step 29: sidebar has keyboard focus
)

// MainModel gestisce il layout a 2/3 pannelli dopo il login.
// Step 28: cmdPalette, whichKey, help overlay sub-models added.
// Step 29: folderModel (inline panel, ADR-016 §D2) + chatInfo overlay (ADR-017 §D1).
// Step 30: layoutMode + compactVisible for responsive layout (ADR-018).
// Single activeOverlay mutex enforced in handleKeyMsg (ADR-015 §D3).
type MainModel struct {
	chatList     ChatListModel
	conversation ConversationModel
	search       SearchModel
	cmdPalette   CmdPaletteModel // Step 28
	whichKey     WhichKeyModel   // Step 28
	help         HelpModel       // Step 28
	folderModel  FolderModel     // Step 29: inline sidebar
	chatInfo     ChatInfoModel   // Step 29: chat info overlay
	allChats     []model.Chat    // unfiltered dialogs; chatList shows filtered view
	Bridge       *telegram.Bridge
	Width        int
	Height       int
	focus        FocusPanel
	statusMsg    string
	typing       map[model.ChatID]TypingState
	// Step 30: responsive layout state (ADR-018).
	layoutMode       LayoutMode   // Wide | Compact (derived from WindowSizeMsg.Width)
	compactVisible   CompactPanel // meaningful only when layoutMode == LayoutCompact
	compactThreshold int          // Step 31: from config.toml [display] compact_threshold (ADR-019 D6)
	// Step 32: bbox cache for central mouse hit-test router (ADR-020 §D2).
	// Invalidated on WindowSizeMsg + LayoutModeChanged + FolderToggle (recomputeBboxes).
	bboxes mainBboxes
	// Step 34: notify banner (ADR-022). Rendered at bottom in place of status bar when Active.
	Notify components.NotifyModel
}

// NewMainModel crea il layout principale con soglia responsive da config (ADR-019 D6).
// threshold 0 → usa default (100, ADR-018 §D1). Chiamato da ui.NewAppModel.
func NewMainModel(threshold int) MainModel {
	if threshold <= 0 {
		threshold = defaultCompactThreshold
	}
	return MainModel{
		chatList:         NewChatListModel(),
		conversation:     NewConversationModel(),
		search:           NewSearchModel(),
		cmdPalette:       NewCmdPaletteModel(),
		whichKey:         NewWhichKeyModel(),
		folderModel:      NewFolderModel(),
		chatInfo:         NewChatInfoModel(),
		typing:           make(map[model.ChatID]TypingState),
		compactThreshold: threshold,
	}
}

// SetSize recalculates layout for all sub-models.
// Step 29: three-panel aware (folder sidebar + chatList + conversation).
// Step 30: branches on layoutMode — compact uses full-width single-pane layout.
func (m *MainModel) SetSize(w, h int) {
	m.Width = w
	m.Height = h
	if m.layoutMode == LayoutCompact {
		m.applyCompactSizes()
	} else {
		m.setWideSize(w, h)
	}
	m.recomputeBboxes() // Step 32: invalidate bbox cache (ADR-020 §D2)
}

// setWideSize applies the multi-panel layout calculation (Wide mode).
func (m *MainModel) setWideSize(w, h int) {
	available := h - 1 // reserve 1 row for status bar
	fldW := 0
	if m.folderModel.IsVisible() {
		fldW = folderSidebarWidth
	}
	m.folderModel.Width = fldW
	m.folderModel.Height = available
	rest := w - fldW
	clW := int(float64(rest) * chatListRatio)
	m.chatList.Width = clW
	m.chatList.Height = available
	convW := rest - clW
	m.conversation.SetSize(convW, available)
	m.chatInfo.SetSize(convW, available)
	m.search.Width = w
	m.search.Height = h
	m.cmdPalette.Width = w
	m.cmdPalette.Height = h
	m.whichKey.Width = w
	m.whichKey.Height = h
	m.help.Width = w
	m.help.Height = h
}
