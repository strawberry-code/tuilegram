package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// GradientColors ritorna (start, end) del gradiente dal tema attivo.
// I valori hex sono convertiti in colorful.Color; errori di parse usano
// il fallback magenta/viola hardcoded (non dovrebbe accadere — default.toml
// è garantito valido da TestEmbeddedDefaultTheme_Parses).
func GradientColors() (colorful.Color, colorful.Color) {
	t := Active()
	start, err := colorful.Hex(t.GradientStart)
	if err != nil {
		start, _ = colorful.Hex("#FF60FF") // fallback magenta
	}
	end, err := colorful.Hex(t.GradientEnd)
	if err != nil {
		end, _ = colorful.Hex("#6B50FF") // fallback viola
	}
	return start, end
}

// RenderGradient applica un gradiente orizzontale per-carattere a un blocco
// di testo multi-riga. Ogni riga ha il gradiente ricalcolato indipendentemente
// così le colonne alla stessa posizione hanno lo stesso colore.
func RenderGradient(text string, from, to colorful.Color) string {
	lines := strings.Split(text, "\n")
	rendered := make([]string, 0, len(lines))

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			rendered = append(rendered, line)
			continue
		}
		rendered = append(rendered, colorLine(line, from, to))
	}

	return strings.Join(rendered, "\n")
}

// colorLine applica il gradiente a una singola riga, carattere per carattere.
func colorLine(line string, from, to colorful.Color) string {
	runes := []rune(line)
	if len(runes) == 0 {
		return ""
	}

	var b strings.Builder
	b.Grow(len(line) * 20) // stima per escape sequences ANSI

	for i, r := range runes {
		// Interpolazione HCL: mantiene colori percettivamente uniformi
		t := 0.0
		if len(runes) > 1 {
			t = float64(i) / float64(len(runes)-1)
		}
		c := from.BlendHcl(to, t).Clamped()

		style := lipgloss.NewStyle().Foreground(lipgloss.Color(c.Hex()))
		b.WriteString(style.Render(string(r)))
	}

	return b.String()
}
