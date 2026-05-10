package theme

// ChangedMsg viene inviato dalla goroutine watcher (fsnotify) tramite
// program.Send quando theme.toml viene modificato e il merge ha successo.
// App.Update lo consuma → styles.SetActive(msg.NewTheme) + re-render.
// Invariante MERGE_ATOMIC_ON_UPDATE: il merge è completato prima dell'invio.
type ChangedMsg struct {
	NewTheme *Theme
}

// WatcherWarnMsg viene inviato dalla goroutine watcher quando la lettura
// o il parse di theme.toml fallisce. Il tema corrente è PRESERVATO
// (invariante INVALID_PRESERVES_THEME da theming.tla).
type WatcherWarnMsg struct {
	Reason string
}
