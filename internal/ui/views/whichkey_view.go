package views

import (
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/components"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// View renders the which-key overlay when Visible; "" otherwise.
func (m WhichKeyModel) View() string {
	if m.state != wkVisible {
		return ""
	}
	return renderWhichKeyOverlay(m)
}

// renderWhichKeyOverlay renders the compact which-key menu bottom-right.
func renderWhichKeyOverlay(m WhichKeyModel) string {
	modal, ok := whichKeyModalFor(m)
	if !ok {
		return ""
	}
	return modal.Render(m.Width, m.Height)
}

// overlayWhichKey renders which-key composited on bg (Crush-style).
func overlayWhichKey(bg string, m WhichKeyModel, width int) string {
	modal, ok := whichKeyModalFor(m)
	if !ok {
		return bg
	}
	return modal.RenderOverlay(bg, width)
}

func whichKeyModalFor(m WhichKeyModel) (components.Modal, bool) {
	conts, ok := m.continuations[m.activePrefix]
	if !ok {
		return components.Modal{}, false
	}
	return components.Modal{
		Body:      buildWhichKeyBody(m.activePrefix, conts),
		Hints:     "esc cancel",
		Width:     40,
		Placement: components.PlacementBottomRight,
	}, true
}

// renderCompactBox renders the modal box without the Place call.
func renderCompactBox(modal components.Modal, canvasW int) string {
	w := 40
	if canvasW > 0 && w > canvasW-4 {
		w = canvasW - 4
	}
	if w < 20 {
		w = 20
	}
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColorPrimary()).
		Padding(0, 1).
		Width(w)
	hintsStyle := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	parts := []string{modal.Body}
	if modal.Hints != "" {
		parts = append(parts, hintsStyle.Render(modal.Hints))
	}
	content := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return borderStyle.Render(content)
}

// buildWhichKeyBody builds the "key → label" table for the overlay body.
func buildWhichKeyBody(prefix string, conts map[string]ContinuationEntry) string {
	keyStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(styles.ColorText())
	keys := make([]string, 0, len(conts))
	for k := range conts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	sb.WriteString(labelStyle.Render("prefix: " + prefix + "\n\n"))
	for i, key := range keys {
		if i > 0 {
			sb.WriteString("\n")
		}
		row := keyStyle.Render(key) + labelStyle.Render("  →  "+conts[key].Label)
		sb.WriteString(row)
	}
	return sb.String()
}
