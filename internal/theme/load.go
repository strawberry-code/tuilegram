package theme

import (
	"fmt"
	"os"
	"regexp"

	toml "github.com/pelletier/go-toml/v2"
)

// hexPattern valida colori nel formato #RRGGBB (strict, D10).
var hexPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// rawThemeTOML rispecchia la struttura TOML di theme.toml per l'Unmarshal.
type rawThemeTOML struct {
	Meta     rawMeta     `toml:"meta"`
	Colors   rawColors   `toml:"colors"`
	Gradient rawGradient `toml:"gradient"`
}

type rawMeta struct {
	Name        string `toml:"name"`
	Author      string `toml:"author"`
	Version     string `toml:"version"`
	Description string `toml:"description"`
}

type rawColors struct {
	Primary          string `toml:"primary"`
	Incoming         string `toml:"incoming"`
	Success          string `toml:"success"`
	Warning          string `toml:"warning"`
	Error            string `toml:"error"`
	Private          string `toml:"private"`
	Text             string `toml:"text"`
	TextDim          string `toml:"text_dim"`
	Surface          string `toml:"surface"`
	Border           string `toml:"border"`
	SearchSecondary  string `toml:"search_secondary"`
	SearchInlineBg   string `toml:"search_inline_bg"`
	ButtonFg         string `toml:"button_fg"`
	ButtonBg         string `toml:"button_bg"`
	ButtonDisabledFg string `toml:"button_disabled_fg"`
	Reaction         string `toml:"reaction"`
	ReactionChosen   string `toml:"reaction_chosen"`
	SystemMessage    string `toml:"system_message"`
	// Step 33: 3 nuovi color keys + palette (ADR-021 §DD5).
	Link          string   `toml:"link"`
	Pinned        string   `toml:"pinned"`
	ForwardLabel  string   `toml:"forward_label"`
	SenderPalette []string `toml:"sender_palette"`
}

type rawGradient struct {
	Start string `toml:"start"`
	End   string `toml:"end"`
}

// LoadTheme carica theme.toml da path, merge sopra base con strategia
// override-by-key fail-soft (ADR-019 D3/D5). Se path è vuoto o file
// non leggibile → ritorna base + warning (D4). Mai panic su input utente.
func LoadTheme(path string, base Theme) (Theme, []string) {
	if path == "" {
		return base, []string{"no theme file found, using default"}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return base, []string{fmt.Sprintf("theme: cannot read %s: %v", path, err)}
	}
	var raw rawThemeTOML
	if err := toml.Unmarshal(data, &raw); err != nil {
		return base, []string{fmt.Sprintf("theme: parse error in %s: %v", path, err)}
	}
	return Merge(base, raw)
}

// isValidHex ritorna true se s è un hex color valido (#RRGGBB).
func isValidHex(s string) bool { return hexPattern.MatchString(s) }
