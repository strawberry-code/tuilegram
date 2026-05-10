package components

// modal.go — Unified overlay primitive (Crush-inspired, Step 34).
// Title rendered inline border-top: ╭─ Title ─────╮
// Border color = ColorBorder (dim), title = ColorPrimary (magenta, bold).
// All app dialogs (palette, search, help, edit, delete, ...) use Modal.Render().

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

// Tone controls border colour: Default = ColorBorder, Warning = ColorError.
type Tone int

const (
	ToneDefault Tone = iota
	ToneWarning
)

// Modal is the unified overlay primitive.
type Modal struct {
	Title     string
	Body      string
	Hints     string
	Width     int
	Tone      Tone
	Placement Placement
}

// Render composes the modal box and places it on a canvas of canvasW×canvasH.
func (m Modal) Render(canvasW, canvasH int) string {
	w := m.resolveWidth(canvasW)
	box := m.buildBox(w)
	hPos, vPos := placementToLipgloss(m.Placement)
	return lipgloss.Place(canvasW, canvasH, hPos, vPos, box)
}

// RenderOverlay composes the modal box and overlays it on bg (Crush style).
// Position derived from m.Placement; cells outside box preserve bg content.
func (m Modal) RenderOverlay(bg string, canvasW int) string {
	w := m.resolveWidth(canvasW)
	box := m.buildBox(w)
	hPos, vPos := placementToLipgloss(m.Placement)
	return OverlayAt(bg, box, hPos, vPos)
}

// resolveWidth clamps width per spec.
func (m Modal) resolveWidth(canvasW int) int {
	w := m.Width
	if w == 0 {
		w = 70
		if auto := canvasW - 8; auto < w {
			w = auto
		}
	}
	if cap := canvasW - 4; w > cap {
		w = cap
	}
	if w < 20 {
		w = 20
	}
	return w
}

// buildBox: top-border width derived from body's actual rendered width.
func (m Modal) buildBox(w int) string {
	bcolor := styles.ColorBorder()
	if m.Tone == ToneWarning {
		bcolor = styles.ColorError()
	}
	body := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderTop(false).
		BorderForeground(bcolor).
		Padding(1, 2).
		Width(w).
		Render(m.composeBody())
	bodyW := lipgloss.Width(body)
	return lipgloss.JoinVertical(lipgloss.Left, m.renderTopBorder(bodyW, bcolor), body)
}

// composeBody joins body + hints with dim style.
func (m Modal) composeBody() string {
	var parts []string
	if m.Body != "" {
		parts = append(parts, m.Body)
	}
	if m.Hints != "" {
		hint := lipgloss.NewStyle().Foreground(styles.ColorTextDim()).MarginTop(1).Render(m.Hints)
		parts = append(parts, hint)
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// renderTopBorder builds custom top: ╭─ Title ─────╮ ; w = total visible width.
func (m Modal) renderTopBorder(w int, bcolor lipgloss.Color) string {
	total := w
	corner := lipgloss.NewStyle().Foreground(bcolor)
	if m.Title == "" {
		return corner.Render("╭" + strings.Repeat("─", total-2) + "╮")
	}
	titleStyle := lipgloss.NewStyle().Foreground(styles.ColorPrimary()).Bold(true)
	maxTitle := total - 6
	t := m.Title
	if lipgloss.Width(t) > maxTitle {
		t = t[:maxTitle]
	}
	plain := " " + t + " "
	plainLen := lipgloss.Width(plain)
	// total = "╭─" (2) + plain (plainLen) + dashes + "╮" (1) → dashes = total-3-plainLen.
	dashes := total - 3 - plainLen
	if dashes < 0 {
		dashes = 0
	}
	return corner.Render("╭─") + titleStyle.Render(plain) + corner.Render(strings.Repeat("─", dashes)+"╮")
}
