package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/model"
	"github.com/strawberry-code/tuilegram/internal/ui/components"
	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

const searchMaxVisible = 8

// renderSearchModal builds the Modal body and delegates centering to Modal.Render.
func renderSearchModal(m SearchModel) string {
	return searchModalFor(m).Render(m.Width, m.Height)
}

// overlaySearch renders search modal composited on bg (Crush-style).
func overlaySearch(bg string, m SearchModel, width int) string {
	return searchModalFor(m).RenderOverlay(bg, width)
}

func searchModalFor(m SearchModel) components.Modal {
	body := lipgloss.JoinVertical(lipgloss.Left,
		m.input.View(),
		"",
		renderSearchBody(m),
	)
	return components.Modal{
		Title: "Search",
		Body:  body,
		Hints: "↑↓ navigate  •  enter open  •  esc close",
		Tone:  components.ToneDefault,
	}
}

// renderSearchBody returns the state-specific content below the textinput.
func renderSearchBody(m SearchModel) string {
	dim := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	errSt := lipgloss.NewStyle().Foreground(styles.ColorError())
	switch m.state {
	case SearchStateIdle:
		return dim.Render("Type to search messages…")
	case SearchStateTyping:
		return dim.Render("…")
	case SearchStateSearching:
		return dim.Render("Searching…")
	case SearchStateEmpty:
		return dim.Render(fmt.Sprintf("No results for %q", m.Query))
	case SearchStateError:
		return errSt.Render("Search failed  · keep typing to retry")
	case SearchStateResults:
		return renderHitList(m.Hits, m.cursor)
	}
	return ""
}

// renderHitList renders a scrolling window of hits with cursor highlight.
func renderHitList(hits []model.SearchHit, cursor int) string {
	dim := lipgloss.NewStyle().Foreground(styles.ColorTextDim())
	normal := lipgloss.NewStyle().Foreground(styles.ColorText())
	cursor_ := lipgloss.NewStyle().Foreground(styles.ColorSurface()).Background(styles.ColorPrimary()).Bold(true)
	if len(hits) == 0 {
		return dim.Render("No results")
	}
	start, end := visibleWindow(cursor, len(hits), searchMaxVisible)
	var sb strings.Builder
	for i := start; i < end; i++ {
		line := formatHit(hits[i])
		if i == cursor {
			sb.WriteString(cursor_.Render("▸ " + line))
		} else {
			sb.WriteString(normal.Render("  " + line))
		}
		if i < end-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
