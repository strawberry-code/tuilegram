package config

import "testing"

func TestLoadConfig_MissingFile_ReturnsDefault(t *testing.T) {
	cfg, warns := LoadConfig("")
	def := DefaultConfig()
	if cfg.Display.CompactThreshold != def.Display.CompactThreshold {
		t.Errorf("expected default threshold %d, got %d", def.Display.CompactThreshold, cfg.Display.CompactThreshold)
	}
	if len(warns) != 0 {
		t.Errorf("expected no warnings for empty path, got %v", warns)
	}
}

func TestLoadConfig_NonExistentFile_ReturnsDefault(t *testing.T) {
	cfg, warns := LoadConfig("/nonexistent/path/config.toml")
	if cfg.Display.CompactThreshold != DefaultCompactThreshold {
		t.Errorf("expected default threshold, got %d", cfg.Display.CompactThreshold)
	}
	if len(warns) == 0 {
		t.Error("expected warning for unreadable file")
	}
}

func TestLoadConfig_InvalidTOML_ReturnsDefault(t *testing.T) {
	cfg, warns := LoadConfig("testdata/invalid_toml.toml")
	if cfg.Display.CompactThreshold != DefaultCompactThreshold {
		t.Errorf("expected default threshold, got %d", cfg.Display.CompactThreshold)
	}
	if len(warns) == 0 {
		t.Error("expected warning for TOML syntax error")
	}
}

func TestLoadConfig_OutOfRange_ClampsToDefault(t *testing.T) {
	cfg, warns := LoadConfig("testdata/out_of_range.toml")
	if cfg.Display.CompactThreshold != DefaultCompactThreshold {
		t.Errorf("expected clamped default %d, got %d", DefaultCompactThreshold, cfg.Display.CompactThreshold)
	}
	if len(warns) == 0 {
		t.Error("expected warning for out-of-range threshold")
	}
}

func TestLoadConfig_ValidOverride_Applied(t *testing.T) {
	cfg, warns := LoadConfig("testdata/valid.toml")
	if cfg.Display.CompactThreshold != 80 {
		t.Errorf("expected threshold 80, got %d", cfg.Display.CompactThreshold)
	}
	if len(warns) != 0 {
		t.Errorf("unexpected warnings: %v", warns)
	}
}
