package theme

import "testing"

func TestEmbeddedDefaultTheme_Parses(t *testing.T) {
	def := DefaultTheme()
	if def.Primary == "" {
		t.Error("default theme Primary should not be empty")
	}
	if def.Name == "" {
		t.Error("default theme Name should not be empty")
	}
}

func TestKnownColorKeysCount(t *testing.T) {
	// 18 originali + 3 Step 33 (link, pinned, forward_label) = 21.
	if len(KnownColorKeys) != 21 {
		t.Errorf("expected 21 known color keys, got %d", len(KnownColorKeys))
	}
}

func TestLoadTheme_MissingPath_ReturnsDefault(t *testing.T) {
	def := DefaultTheme()
	got, warns := LoadTheme("", def)
	if got.Primary != def.Primary {
		t.Errorf("expected default primary %v, got %v", def.Primary, got.Primary)
	}
	if len(warns) == 0 {
		t.Error("expected at least one warning for missing path")
	}
}

func TestLoadTheme_InvalidTOML_ReturnsDefault(t *testing.T) {
	def := DefaultTheme()
	got, warns := LoadTheme("testdata/invalid_toml.toml", def)
	if got.Primary != def.Primary {
		t.Errorf("expected default primary on parse error, got %v", got.Primary)
	}
	if len(warns) == 0 {
		t.Error("expected warning on TOML syntax error")
	}
}

func TestLoadTheme_BadHexOnKey_UsesDefault(t *testing.T) {
	def := DefaultTheme()
	got, warns := LoadTheme("testdata/bad_hex.toml", def)
	// primary bad hex → default retained; incoming valid → overridden
	if got.Primary != def.Primary {
		t.Errorf("bad hex primary: expected default %v, got %v", def.Primary, got.Primary)
	}
	if got.Incoming == def.Incoming {
		t.Error("valid incoming hex should have been applied as override")
	}
	if len(warns) == 0 {
		t.Error("expected warning for bad hex")
	}
}

func TestLoadTheme_PartialOverride_MergedCorrectly(t *testing.T) {
	def := DefaultTheme()
	got, warns := LoadTheme("testdata/valid_partial.toml", def)
	// primary overridden to #FF00FF; all other keys unchanged
	if string(got.Primary) != "#FF00FF" {
		t.Errorf("expected #FF00FF primary, got %v", got.Primary)
	}
	if got.Incoming != def.Incoming {
		t.Errorf("incoming should remain default %v, got %v", def.Incoming, got.Incoming)
	}
	if got.Name != "test-partial" {
		t.Errorf("expected name 'test-partial', got %q", got.Name)
	}
	_ = warns
}

func TestDefaultTheme_IsTotal(t *testing.T) {
	def := DefaultTheme()
	// Verifica che tutti i campi critici siano valorizzati
	fields := []struct {
		name string
		val  string
	}{
		{"Primary", string(def.Primary)},
		{"Incoming", string(def.Incoming)},
		{"Success", string(def.Success)},
		{"Warning", string(def.Warning)},
		{"Error", string(def.Error)},
		{"Text", string(def.Text)},
		{"TextDim", string(def.TextDim)},
		{"GradientStart", def.GradientStart},
		{"GradientEnd", def.GradientEnd},
	}
	for _, f := range fields {
		if f.val == "" {
			t.Errorf("field %s should not be empty in default theme", f.name)
		}
	}
}
