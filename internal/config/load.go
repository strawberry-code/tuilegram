package config

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"
)

// rawConfigTOML rispecchia la struttura TOML di config.toml per Unmarshal.
type rawConfigTOML struct {
	Meta    rawConfigMeta    `toml:"meta"`
	Display rawConfigDisplay `toml:"display"`
	Paths   rawConfigPaths   `toml:"paths"`
}

type rawConfigMeta struct {
	Version int `toml:"version"`
}

type rawConfigDisplay struct {
	CompactThreshold *int `toml:"compact_threshold"` // pointer per distinguere "non presente" da 0
}

type rawConfigPaths struct {
	Theme string `toml:"theme"`
}

// LoadConfig carica config.toml da path con strategia fail-soft (ADR-019 D5).
// Se path è "" → DefaultConfig + warning silenzioso (D4, no crash).
// TOML syntax error → DefaultConfig + warning. Per-key validation (D10).
func LoadConfig(path string) (Config, []string) {
	if path == "" {
		return DefaultConfig(), nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultConfig(), []string{fmt.Sprintf("config: cannot read %s: %v", path, err)}
	}
	var raw rawConfigTOML
	if err := toml.Unmarshal(data, &raw); err != nil {
		return DefaultConfig(), []string{fmt.Sprintf("config: parse error in %s: %v", path, err)}
	}
	return mergeConfig(DefaultConfig(), raw)
}

// mergeConfig applica override-by-key con fail-soft validation (D10).
// Ritorna Config total + warnings non-fatali.
func mergeConfig(base Config, raw rawConfigTOML) (Config, []string) {
	out := base
	var warns []string

	// [meta] version: free integer, nessuna validation di range
	if raw.Meta.Version != 0 {
		out.Meta.Version = raw.Meta.Version
	}

	// [display] compact_threshold: range [40, 400] (D10)
	if raw.Display.CompactThreshold != nil {
		v := *raw.Display.CompactThreshold
		if v < MinCompactThreshold || v > MaxCompactThreshold {
			warns = append(warns, fmt.Sprintf(
				"config: compact_threshold %d fuori range [%d, %d], clamped a %d",
				v, MinCompactThreshold, MaxCompactThreshold, DefaultCompactThreshold,
			))
			out.Display.CompactThreshold = DefaultCompactThreshold
		} else {
			out.Display.CompactThreshold = v
		}
	}

	// [paths] theme: path assoluto only (ADR-019 D2 — validation differita a ResolveThemePath)
	if raw.Paths.Theme != "" {
		if !filepath.IsAbs(raw.Paths.Theme) {
			warns = append(warns, "config: paths.theme deve essere assoluto (ignorato): "+raw.Paths.Theme)
		} else {
			out.Paths.Theme = raw.Paths.Theme
		}
	}

	return out, warns
}
