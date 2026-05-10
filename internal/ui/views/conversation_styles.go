package views

import (
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// conversation_styles.go — lipgloss style helpers per la conversazione.
// Estratto da conversation_view.go per rispettare il limite 120-LOC.
// Tutte le funzioni sono stateless: creano uno stile fresco ad ogni chiamata
// leggendo il tema corrente via styles.Color*() (ADR-019 §D7).
//
// NIT #1 (deferred): lipgloss.NewStyle() è allocato per ogni render.
// Package-level vars sarebbero più efficienti ma congelerebbero i colori al
// momento dell'init, rompendo il live-reload del tema (Step 31 ADR-019).
// Trade-off accettato: il tema è dinamico, le allocazioni sono lightweight
// (lipgloss.Style è una struct value, non heap-heavy). Rivalutare se il
// profiler evidenzia pressure su questa path (perf-profiler §bench).

// inputBorderStyle: solo BorderTop (Step 34) — composer stuck-to-bottom.
// Niente top/bottom invisibili che generano "gap" visuale.
func inputBorderStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderTop(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(styles.ColorBorder())
}

func incomingTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.ColorIncoming())
}

func outgoingTextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.ColorPrimary())
}

func msgTimeStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(styles.ColorTextDim())
}
