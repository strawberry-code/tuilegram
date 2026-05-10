package theme

import (
	_ "embed"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	toml "github.com/pelletier/go-toml/v2"
)

//go:embed default.toml
var defaultThemeRaw []byte

// parsedDefault è inizializzato una sola volta all'avvio del package.
// Se default.toml embedded non è parsabile → panic (bug di build; catturato
// da TestEmbeddedDefaultTheme_Parses in CI prima del deploy).
var parsedDefault Theme

func init() {
	var raw rawThemeTOML
	if err := toml.Unmarshal(defaultThemeRaw, &raw); err != nil {
		panic("theme: default.toml embedded non parsabile: " + err.Error())
	}
	validateRawHex(raw) // build-time invariant: tutti gli hex devono essere validi
	parsedDefault = buildThemeFromRaw(raw)
}

// validateRawHex panica se default.toml ha hex invalidi. Catturato a startup,
// non in runtime: garantisce che parsedDefault sia totale e ben formato.
func validateRawHex(r rawThemeTOML) {
	checks := map[string]string{
		"primary": r.Colors.Primary, "incoming": r.Colors.Incoming,
		"success": r.Colors.Success, "warning": r.Colors.Warning,
		"error": r.Colors.Error, "private": r.Colors.Private,
		"text": r.Colors.Text, "text_dim": r.Colors.TextDim,
		"surface": r.Colors.Surface, "border": r.Colors.Border,
		"search_secondary": r.Colors.SearchSecondary, "search_inline_bg": r.Colors.SearchInlineBg,
		"button_fg": r.Colors.ButtonFg, "button_bg": r.Colors.ButtonBg,
		"button_disabled_fg": r.Colors.ButtonDisabledFg,
		"reaction":           r.Colors.Reaction, "reaction_chosen": r.Colors.ReactionChosen,
		"system_message": r.Colors.SystemMessage,
		"gradient.start": r.Gradient.Start, "gradient.end": r.Gradient.End,
		// Step 33 keys: sempre presenti nel default.toml (THEME_TOTAL invariante).
		"link": r.Colors.Link, "pinned": r.Colors.Pinned, "forward_label": r.Colors.ForwardLabel,
	}
	for key, val := range checks {
		if !isValidHex(val) {
			panic("theme: default.toml hex invalido per " + key + ": " + val)
		}
	}
	for i, c := range r.Colors.SenderPalette {
		if !isValidHex(c) {
			panic("theme: default.toml sender_palette[" + fmt.Sprintf("%d", i) + "] invalido: " + c)
		}
	}
}

// DefaultTheme ritorna una copia del tema embedded di default.
// Sempre total (invariante THEME_TOTAL). Nessun file su disco richiesto.
func DefaultTheme() Theme { return parsedDefault }

// buildThemeFromRaw converte la struttura TOML grezza in Theme tipizzato.
// Usa i valori di default per chiavi mancanti o non parsabili.
func buildThemeFromRaw(r rawThemeTOML) Theme {
	c := r.Colors
	g := r.Gradient
	m := r.Meta
	t := Theme{
		Primary:          lipgloss.Color(c.Primary),
		Incoming:         lipgloss.Color(c.Incoming),
		Success:          lipgloss.Color(c.Success),
		Warning:          lipgloss.Color(c.Warning),
		Error:            lipgloss.Color(c.Error),
		Private:          lipgloss.Color(c.Private),
		Text:             lipgloss.Color(c.Text),
		TextDim:          lipgloss.Color(c.TextDim),
		Surface:          lipgloss.Color(c.Surface),
		Border:           lipgloss.Color(c.Border),
		SearchSecondary:  lipgloss.Color(c.SearchSecondary),
		SearchInlineBg:   lipgloss.Color(c.SearchInlineBg),
		ButtonFg:         lipgloss.Color(c.ButtonFg),
		ButtonBg:         lipgloss.Color(c.ButtonBg),
		ButtonDisabledFg: lipgloss.Color(c.ButtonDisabledFg),
		Reaction:         lipgloss.Color(c.Reaction),
		ReactionChosen:   lipgloss.Color(c.ReactionChosen),
		SystemMessage:    lipgloss.Color(c.SystemMessage),
		Link:             lipgloss.Color(c.Link),
		Pinned:           lipgloss.Color(c.Pinned),
		ForwardLabel:     lipgloss.Color(c.ForwardLabel),
		GradientStart:    g.Start,
		GradientEnd:      g.End,
		Name:             m.Name,
		Author:           m.Author,
		Version:          m.Version,
		Description:      m.Description,
	}
	// Sender palette: usa i valori TOML; invariante lunghezza = 8.
	for i := 0; i < 8 && i < len(c.SenderPalette); i++ {
		t.SenderPalette[i] = lipgloss.Color(c.SenderPalette[i])
	}
	return t
}
