package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// overlay.go — Crush-style modal overlay (Step 34).
// Renders fg composited on top of bg, preserving cells outside fg's box.
// ANSI-aware via x/ansi (Cut + Truncate).

// Overlay paints fg centered on bg. Convenience wrapper for OverlayAt.
func Overlay(bg, fg string) string {
	return OverlayAt(bg, fg, lipgloss.Center, lipgloss.Center)
}

// OverlayAt paints fg on bg at given horizontal/vertical position.
// Positions: lipgloss.Left/Center/Right (h) and Top/Center/Bottom (v).
func OverlayAt(bg, fg string, hPos, vPos lipgloss.Position) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	bgW := maxLineWidth(bgLines)
	fgW := maxLineWidth(fgLines)
	x := int(float64(bgW-fgW) * float64(hPos))
	y := int(float64(len(bgLines)-len(fgLines)) * float64(vPos))
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	out := make([]string, len(bgLines))
	copy(out, bgLines)
	for i, fgLine := range fgLines {
		row := y + i
		if row < 0 || row >= len(out) {
			continue
		}
		out[row] = overlayLine(out[row], fgLine, x, bgW)
	}
	return strings.Join(out, "\n")
}

// overlayLine splices fg into bg at column x, preserving bg outside [x, x+fgW).
func overlayLine(bg, fg string, x, bgW int) string {
	fgW := lipgloss.Width(fg)
	leftEnd := x
	rightStart := x + fgW
	left := ansi.Cut(bg, 0, leftEnd)
	right := ""
	if rightStart < bgW {
		right = ansi.Cut(bg, rightStart, bgW)
	}
	return left + fg + right
}

// maxLineWidth scans lines and returns longest visible width.
func maxLineWidth(lines []string) int {
	w := 0
	for _, l := range lines {
		if lw := lipgloss.Width(l); lw > w {
			w = lw
		}
	}
	return w
}
