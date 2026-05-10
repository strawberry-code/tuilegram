package views

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// main_status_bar.go — dual-slot status bar rendering (ADR-021 §DD1).
// Estratto da main_view.go per rispettare il limite 120-LOC.

// renderStatusBar compone la status bar dual-slot (ADR-021 §DD1, DD4).
// Invariante STATUSBAR_TWO_SLOT: esattamente 2 slot (left hint, right error/info).
// Invariante STATUSBAR_ERROR_PRIORITY: ellipsize left prima del right.
func (m MainModel) renderStatusBar() string {
	hint := m.keymapHint()
	right, isErr := m.statusRight()
	const gap = 2
	rightW := lipgloss.Width(right)
	leftMax := m.Width - rightW - gap
	if leftMax < 3 {
		leftMax = 3
	}
	// Ellipsize hint se overflow (STATUSBAR_ERROR_PRIORITY: right preservato).
	hintRunes := []rune(hint)
	if len(hintRunes) > leftMax {
		hint = string(hintRunes[:leftMax-1]) + "…"
	}
	leftStr := lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render(" " + hint)
	var rightStr string
	if isErr {
		rightStr = lipgloss.NewStyle().Foreground(styles.ColorError()).Render("✕ " + right)
	} else {
		rightStr = lipgloss.NewStyle().Foreground(styles.ColorTextDim()).Render(right)
	}
	padding := strings.Repeat(" ", gap)
	return lipgloss.NewStyle().Width(m.Width).Render(leftStr + padding + rightStr)
}

// statusRight restituisce (text, isError) per il right-slot della status bar.
// Invariante STATUSBAR_ERROR_PRIORITY: IsError determina il colore (DD3).
func (m MainModel) statusRight() (string, bool) {
	if m.statusMsg == "" {
		return "", false
	}
	return m.statusMsg, false
}
