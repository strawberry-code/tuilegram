// Package watcher implementa il hot-reload di theme.toml via fsnotify.
// La goroutine Watch è lifecycle-bound al context (WATCHER_BOUND_TO_LIFECYCLE).
// Pipeline single-inflight: un solo reload alla volta (SINGLE_RELOAD_INFLIGHT)
// garantito da reloadWorker che processa triggers serialmente in watcher_reload.go.
// Debounce di 100ms per coalescing editor-save multi-evento (es. vim).
package watcher

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fsnotify/fsnotify"

	"github.com/strawberry-code/tuilegram/internal/theme"
)

const debounceDelay = 100 * time.Millisecond

// Watch avvia una goroutine fsnotify che osserva path e invia messaggi
// al programma bubbletea via send. Termina quando ctx è cancellato.
//
// Invarianti (theming.tla §J):
//   - WATCHER_BOUND_TO_LIFECYCLE: goroutine termina su ctx.Done()
//   - SINGLE_RELOAD_INFLIGHT: reloadWorker serializza tutti i reload
//   - MERGE_ATOMIC_ON_UPDATE: merge completato prima di send (LoadTheme)
//   - INVALID_PRESERVES_THEME: errori → WatcherWarnMsg, tema non toccato
func Watch(ctx context.Context, path string, base theme.Theme, send func(tea.Msg)) {
	if path == "" {
		return // nessun file da osservare (boot senza theme.toml)
	}

	w, err := fsnotify.NewWatcher()
	if err != nil {
		send(theme.WatcherWarnMsg{Reason: fmt.Sprintf("watcher: init error: %v", err)})
		return
	}

	if err := w.Add(path); err != nil {
		w.Close()
		send(theme.WatcherWarnMsg{Reason: fmt.Sprintf("watcher: cannot watch %s: %v", path, err)})
		return
	}

	go runLoop(ctx, w, path, base, send)
}

// runLoop legge fsnotify events e dispatcha trigger al reloadWorker via
// canale bufferato di capacità 1. Coalescing automatico: se un trigger è
// già in coda, l'evento successivo viene ignorato (drop) — il reloadWorker
// applicherà comunque l'ultimo stato del file.
func runLoop(ctx context.Context, w *fsnotify.Watcher, path string, base theme.Theme, send func(tea.Msg)) {
	defer w.Close()

	triggers := make(chan struct{}, 1)
	go reloadWorker(ctx, triggers, path, base, send)

	for {
		select {
		case <-ctx.Done():
			return

		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Create) {
				continue
			}
			// Non-blocking send: se un reload è già in coda, coalesce.
			select {
			case triggers <- struct{}{}:
			default:
			}

		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			send(theme.WatcherWarnMsg{Reason: fmt.Sprintf("watcher: fs error: %v", err)})
		}
	}
}
