package components

// notify.go — Banner notify component (Step 34, ADR-022).
// Slide-in/out animation via frame counter, autohide hold timer, replace-on-new.
// Invarianti: NOTIFY_NO_QUEUE, NOTIFY_FRAME_BOUNDED, NOTIFY_REPLACE.
// Type definitions in notify_types.go.

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/strawberry-code/tuilegram/internal/ui/styles"
)

type NotifyModel struct {
	state notifyState
	frame int
	gen   int
	kind  NotifyKind
	text  string
}

func (n NotifyModel) Init() tea.Cmd { return nil }

func (n NotifyModel) Update(msg tea.Msg) (NotifyModel, tea.Cmd) {
	switch m := msg.(type) {
	case NotifyMsg:
		n.gen++
		n.state = nsMounting
		n.frame = 0
		n.kind = m.Kind
		n.text = m.Text
		return n, n.tickCmd()
	case notifyTickMsg:
		if m.gen != n.gen {
			return n, nil
		}
		n.frame++
		if n.frame >= notifyFrames {
			n.frame = notifyFrames - 1
			if n.state == nsMounting {
				n.state = nsVisible
				return n, n.holdCmd()
			}
			if n.state == nsDismissing {
				n.state = nsIdle
				n.text = ""
			}
			return n, nil
		}
		return n, n.tickCmd()
	case notifyTimeoutMsg:
		if m.gen != n.gen || n.state != nsVisible {
			return n, nil
		}
		n.state = nsDismissing
		n.frame = 0
		return n, n.tickCmd()
	}
	return n, nil
}

func (n NotifyModel) tickCmd() tea.Cmd {
	g := n.gen
	return tea.Tick(notifyTickMs*time.Millisecond, func(time.Time) tea.Msg { return notifyTickMsg{gen: g} })
}

func (n NotifyModel) holdCmd() tea.Cmd {
	g := n.gen
	return tea.Tick(notifyHoldMs*time.Millisecond, func(time.Time) tea.Msg { return notifyTimeoutMsg{gen: g} })
}

func (n NotifyModel) View(width int) string {
	if n.state == nsIdle || width <= 0 {
		return ""
	}
	fg, bg := styles.ColorButtonFg(), styles.ColorPrimary()
	label := "[INFO]"
	switch n.kind {
	case NotifySuccess:
		bg = styles.ColorSuccess()
		label = "[OKAY]"
	case NotifyError:
		bg = styles.ColorError()
		label = "[FAIL]"
	}
	style := lipgloss.NewStyle().Foreground(fg).Background(bg).Bold(true).Padding(0, 1).Width(width)
	return style.Render(label + " " + n.text)
}

func (n NotifyModel) Active() bool { return n.state != nsIdle }
