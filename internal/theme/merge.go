package theme

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Merge applica le chiavi di raw sopra base con strategia override-by-key
// fail-soft (ADR-019 D3): valori non-hex → warning + keep base value.
// Funzione pura: non modifica base. Il risultato è sempre total (THEME_TOTAL).
func Merge(base Theme, raw rawThemeTOML) (Theme, []string) {
	out := base
	var warns []string

	// Override metadati (no validation, free text)
	if raw.Meta.Name != "" {
		out.Name = raw.Meta.Name
	}
	if raw.Meta.Author != "" {
		out.Author = raw.Meta.Author
	}
	if raw.Meta.Version != "" {
		out.Version = raw.Meta.Version
	}
	if raw.Meta.Description != "" {
		out.Description = raw.Meta.Description
	}

	warns = append(warns, mergeColors(&out, raw.Colors)...)
	warns = append(warns, mergeGradient(&out, raw.Gradient)...)
	return out, warns
}

// mergeColors applica per-key override fail-soft per la sezione [colors].
func mergeColors(out *Theme, c rawColors) []string {
	var warns []string
	apply := func(key, val string, dst *lipgloss.Color) {
		if val == "" {
			return // chiave non presente nel file utente → skip
		}
		if !isValidHex(val) {
			warns = append(warns, fmt.Sprintf("theme: bad hex for %q: %q (ignored)", key, val))
			return
		}
		*dst = lipgloss.Color(val)
	}
	apply("primary", c.Primary, &out.Primary)
	apply("incoming", c.Incoming, &out.Incoming)
	apply("success", c.Success, &out.Success)
	apply("warning", c.Warning, &out.Warning)
	apply("error", c.Error, &out.Error)
	apply("private", c.Private, &out.Private)
	apply("text", c.Text, &out.Text)
	apply("text_dim", c.TextDim, &out.TextDim)
	apply("surface", c.Surface, &out.Surface)
	apply("border", c.Border, &out.Border)
	apply("search_secondary", c.SearchSecondary, &out.SearchSecondary)
	apply("search_inline_bg", c.SearchInlineBg, &out.SearchInlineBg)
	apply("button_fg", c.ButtonFg, &out.ButtonFg)
	apply("button_bg", c.ButtonBg, &out.ButtonBg)
	apply("button_disabled_fg", c.ButtonDisabledFg, &out.ButtonDisabledFg)
	apply("reaction", c.Reaction, &out.Reaction)
	apply("reaction_chosen", c.ReactionChosen, &out.ReactionChosen)
	apply("system_message", c.SystemMessage, &out.SystemMessage)
	// Step 33: nuovi color keys (ADR-021 §DD5).
	apply("link", c.Link, &out.Link)
	apply("pinned", c.Pinned, &out.Pinned)
	apply("forward_label", c.ForwardLabel, &out.ForwardLabel)
	for i := 0; i < 8 && i < len(c.SenderPalette); i++ {
		apply(fmt.Sprintf("sender_palette[%d]", i), c.SenderPalette[i], &out.SenderPalette[i])
	}
	return warns
}

// mergeGradient applica per-key override fail-soft per la sezione [gradient].
func mergeGradient(out *Theme, g rawGradient) []string {
	var warns []string
	if g.Start != "" {
		if !isValidHex(g.Start) {
			warns = append(warns, fmt.Sprintf("theme: bad hex for gradient.start: %q (ignored)", g.Start))
		} else {
			out.GradientStart = g.Start
		}
	}
	if g.End != "" {
		if !isValidHex(g.End) {
			warns = append(warns, fmt.Sprintf("theme: bad hex for gradient.end: %q (ignored)", g.End))
		} else {
			out.GradientEnd = g.End
		}
	}
	return warns
}
