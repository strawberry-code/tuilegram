// Package config gestisce il caricamento di config.toml (behavior switches).
// Il tema visivo è gestito dal package theme. Questo package è boot-time only:
// nessuna mutazione post-bootstrap (ADR-019 D6).
package config

// Config contiene i behavior switches esposti in Step 31 (ADR-019 D6).
// Tutti i campi hanno default validi; i campi non presenti in config.toml
// restano al default (fail-soft override-by-key, D5).
type Config struct {
	Display DisplayConfig
	Paths   PathsConfig
	Meta    MetaConfig
}

// DisplayConfig contiene i parametri di layout UI.
type DisplayConfig struct {
	// CompactThreshold è la soglia in colonne per Wide vs Compact (ADR-018 D1).
	// Range valido [40, 400]. Default 100. Valori out-of-range vengono clampati.
	CompactThreshold int
}

// PathsConfig consente all'utente di specificare path assoluti espliciti.
type PathsConfig struct {
	// Theme è il path assoluto a theme.toml. Se vuoto usa la priority chain D2.
	// Tilde expansion non supportata (ADR-019 D2 razionale).
	Theme string
}

// MetaConfig contiene metadati di schema per future migrazioni.
type MetaConfig struct {
	Version int // schema version; default 1
}

const (
	DefaultCompactThreshold = 100
	MinCompactThreshold     = 40
	MaxCompactThreshold     = 400
)

// DefaultConfig ritorna la configurazione di default (total, tutti i campi valorizzati).
func DefaultConfig() Config {
	return Config{
		Display: DisplayConfig{CompactThreshold: DefaultCompactThreshold},
		Paths:   PathsConfig{Theme: ""},
		Meta:    MetaConfig{Version: 1},
	}
}
