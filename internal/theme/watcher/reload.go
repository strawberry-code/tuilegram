package watcher

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/theme"
)

// reloadWorker consuma triggers serialmente garantendo SINGLE_RELOAD_INFLIGHT.
// Per ogni trigger applica il debounce: se altri triggers arrivano nella
// finestra debounceDelay, il timer viene resettato (coalescing). Quando
// la finestra si chiude senza nuovi triggers, esegue reload sincrono.
func reloadWorker(ctx context.Context, triggers <-chan struct{}, path string, base theme.Theme, send func(tea.Msg)) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-triggers:
			if !waitDebounced(ctx, triggers) {
				return // ctx cancelled durante il debounce
			}
			reload(path, base, send)
		}
	}
}

// waitDebounced attende debounceDelay; ogni nuovo trigger ricevuto resetta
// la finestra. Ritorna true quando la finestra si chiude tranquilla, false
// se ctx è cancellato.
func waitDebounced(ctx context.Context, triggers <-chan struct{}) bool {
	timer := time.NewTimer(debounceDelay)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return false
		case <-triggers:
			if !timer.Stop() {
				<-timer.C
			}
			timer.Reset(debounceDelay)
		case <-timer.C:
			return true
		}
	}
}

// reload carica, valida e merga il tema; poi invia ChangedMsg o WatcherWarnMsg.
// Eseguito serialmente dal reloadWorker → MERGE_ATOMIC_ON_UPDATE garantito.
// Se LoadTheme produce warning reali, il tema corrente è preservato
// (INVALID_PRESERVES_THEME): inviamo TUTTI i warning, NESSUN ChangedMsg.
func reload(path string, base theme.Theme, send func(tea.Msg)) {
	t, warns := theme.LoadTheme(path, base)
	hasReal := false
	for _, w := range warns {
		if w == "no theme file found, using default" {
			continue // silent (ADR-019 D4)
		}
		send(theme.WatcherWarnMsg{Reason: w})
		hasReal = true
	}
	if hasReal {
		return
	}
	send(theme.ChangedMsg{NewTheme: &t})
}
