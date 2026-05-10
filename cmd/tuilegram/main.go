package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/config"
	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/telegram"
	"github.com/strawberry-code/tuilegram/internal/theme"
	"github.com/strawberry-code/tuilegram/internal/theme/watcher"
	"github.com/strawberry-code/tuilegram/internal/ui"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

func main() {
	// --- Bootstrap: config + theme (ADR-019, statechart §H) ---
	paths := config.ResolvePaths()

	cfg, cfgWarns := config.LoadConfig(paths.ConfigPath)
	for _, w := range cfgWarns {
		fmt.Fprintln(os.Stderr, "tuilegram:", w)
	}

	// Applica override esplicito da config.paths.theme se presente.
	themePath, themePathWarn := config.ResolveThemePath(cfg.Paths.Theme, paths.ThemePath)
	if themePathWarn != "" {
		fmt.Fprintln(os.Stderr, "tuilegram:", themePathWarn)
	}

	defTheme := theme.DefaultTheme()
	thm, themeWarns := theme.LoadTheme(themePath, defTheme)
	for _, w := range themeWarns {
		// "no theme file found" è silenzioso in assenza di file (ADR-019 D4).
		if w != "no theme file found, using default" {
			fmt.Fprintln(os.Stderr, "tuilegram:", w)
		}
	}

	// Applica il tema al package styles (atomic swap, ADR-019 D8).
	styles.SetActive(&thm)

	// --- Telegram ---
	tgCfg := telegram.LoadConfig()
	if !tgCfg.IsValid() {
		fmt.Fprintln(os.Stderr, "error: TELEGRAM_APP_ID e TELEGRAM_APP_HASH richiesti")
		os.Exit(1)
	}

	bridge := telegram.NewBridge(tgCfg)
	appModel := ui.NewAppModel(bridge, cfg)

	p := tea.NewProgram(appModel, tea.WithAltScreen(), tea.WithMouseCellMotion())

	bridge.OnConnected = func() { p.Send(telegram.ConnectedMsg{}) }
	bridge.OnDisconnected = func(err error) { p.Send(telegram.DisconnectedMsg{Err: err}) }
	bridge.OnReconnecting = func() { p.Send(telegram.ReconnectingMsg{}) }
	bridge.OnAuthRequired = func() { p.Send(telegram.AuthRequiredMsg{}) }
	bridge.OnNewMessage = func(msg model.Message, chatID model.ChatID) {
		p.Send(telegram.NewMessageMsg{Message: msg, ChatID: chatID})
	}
	bridge.OnUserTyping = func(peer model.ChatID, userID int64) {
		p.Send(telegram.UpdateUserTypingMsg{Peer: peer, UserID: userID})
	}
	bridge.OnReactionsUpdated = func(chatID model.ChatID, msgID int, reactions []model.Reaction) {
		p.Send(telegram.ReactionsUpdatedMsg{ChatID: chatID, MessageID: msgID, Reactions: reactions})
	}

	bridge.Start()
	defer bridge.Stop()

	// --- Hot-reload watcher (ADR-019 D9, theming.tla §J) ---
	// Goroutine lifecycle bound al context (WATCHER_BOUND_TO_LIFECYCLE).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher.Watch(ctx, themePath, defTheme, p.Send) // spawn interno

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
