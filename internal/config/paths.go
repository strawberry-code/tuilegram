package config

import (
	"os"
	"path/filepath"
)

// PathResolution descrive i path risolti e la sorgente usata per il config.
type PathResolution struct {
	ConfigPath string // path assoluto a config.toml (può essere "")
	ThemePath  string // path assoluto a theme.toml (può essere "")
}

// ResolvePaths calcola i path di config.toml e theme.toml secondo la
// priority chain ADR-019 D2:
//
//  1. $TUILEGRAM_CONFIG_DIR/config.toml      (env escape hatch)
//  2. os.UserConfigDir() + /tuilegram/config.toml  (OS convention)
//
// Per il theme path, config.paths.theme viene applicato DOPO LoadConfig.
// Questo metodo ritorna solo i path candidati senza leggere config.paths.theme.
func ResolvePaths() PathResolution {
	return PathResolution{
		ConfigPath: resolveFile("config.toml"),
		ThemePath:  resolveFile("theme.toml"),
	}
}

// ResolveThemePath calcola il path del tema applicando config.paths.theme
// come override esplicito (priority più alta per il tema, ADR-019 D2).
// Se theme è un path relativo → log warning + ignore (path assoluto only).
func ResolveThemePath(explicitTheme, fallback string) (string, string) {
	if explicitTheme == "" {
		return fallback, ""
	}
	if !filepath.IsAbs(explicitTheme) {
		return fallback, "config.paths.theme deve essere un path assoluto (ignorato): " + explicitTheme
	}
	return explicitTheme, ""
}

// resolveFile cerca filename nella priority chain D2 e ritorna il primo
// path esistente. Ritorna "" se nessun candidato esiste.
func resolveFile(filename string) string {
	dirs := candidateDirs()
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, "tuilegram", filename)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// candidateDirs ritorna le directory candidate in ordine di priorità (D2).
func candidateDirs() []string {
	var dirs []string
	if env := os.Getenv("TUILEGRAM_CONFIG_DIR"); env != "" {
		dirs = append(dirs, env)
	}
	if ucd, err := os.UserConfigDir(); err == nil {
		dirs = append(dirs, ucd)
	}
	return dirs
}
