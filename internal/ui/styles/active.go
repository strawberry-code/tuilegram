package styles

import (
	"sync/atomic"

	"github.com/strawberry-code/tuilegram/internal/theme"
)

// active è il puntatore atomico al tema correntemente applicato.
// Atomic swap per garantire NO_TORN_RELOAD (theming.tla): un singolo write
// pointer atomico su AMD64/ARM64 è sufficiente; atomic.Pointer è esplicito
// e garantisce ordering su ogni arch supportata da Go.
//
// Inizializzato in init() con il DefaultTheme embedded. Mai nil post-init
// (invariante THEME_TOTAL). SetActive è thread-safe (single write atomico).
var active atomic.Pointer[theme.Theme]

func init() {
	def := theme.DefaultTheme()
	active.Store(&def)
}

// Active ritorna il puntatore al tema correntemente attivo. Mai nil.
func Active() *theme.Theme { return active.Load() }

// SetActive sostituisce atomicamente il tema attivo. Chiamato da:
//   - main() al boot (single-thread, safe)
//   - App.Update su ThemeChangedMsg (bubbletea message loop, single goroutine)
//
// Il caller deve passare un *Theme già totale (merge completato, MERGE_ATOMIC_ON_UPDATE).
func SetActive(t *theme.Theme) { active.Store(t) }
