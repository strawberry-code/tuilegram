package views

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Update processes chat-info overlay messages.
// When not Active, only ChatInfoOpenMsg is handled (others are silent no-op).
func (m ChatInfoModel) Update(msg tea.Msg) (ChatInfoModel, tea.Cmd) {
	if !m.Active {
		return m, nil
	}
	switch msg := msg.(type) {
	case ChatInfoCloseMsg:
		return m.Close(), nil
	case ChatInfoCompletionMsg:
		return m.handleCompletion(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// handleKey processes keyboard input while the overlay is open.
// F is UX-consumed (no-op) per ADR-017 §D5 to prevent layout shift.
// Esc and 'i' close the overlay. j/k/PgUp/PgDn scroll the body.
func (m ChatInfoModel) handleKey(msg tea.KeyMsg) (ChatInfoModel, tea.Cmd) {
	switch msg.String() {
	case "esc", "i":
		return m.Close(), nil
	case "j", "down":
		m.vp.ScrollDown(1)
	case "k", "up":
		m.vp.ScrollUp(1)
	case "pgdown":
		m.vp.HalfPageDown()
	case "pgup":
		m.vp.HalfPageUp()
	case "F":
		// UX-consume: sidebar toggle blocked while overlay is open (ADR-017 §D5).
		// Silent no-op — no message, no layout change.
	}
	return m, nil
}

// handleCompletion merges bio from the lazy fetch.
// STALE_COMPLETION_DROP: if chatID != target → no-op (ADR-017 §D2).
// Write-through to card is benign even if we close before it arrives.
func (m ChatInfoModel) handleCompletion(msg ChatInfoCompletionMsg) (ChatInfoModel, tea.Cmd) {
	if m.target == nil || *m.target != msg.ChatID {
		// Stale completion: target changed. No-op per invariant STALE_COMPLETION_DROP.
		return m, nil
	}
	m.MergeCompletion(msg)
	// Refresh viewport content after merge.
	m.vp.SetContent(renderInfoBody(m.card))
	return m, nil
}

// SetSize updates the viewport dimensions so the body scrolls correctly.
func (m *ChatInfoModel) SetSize(w, h int) {
	m.Width = w
	m.Height = h
	// Leave room for modal border (2×2) and hints row (1).
	m.vp.Width = w - 6
	m.vp.Height = h - 6
	if m.Active {
		m.vp.SetContent(renderInfoBody(m.card))
	}
}

// RefreshContent forces a viewport re-render (called after Open).
func (m *ChatInfoModel) RefreshContent() {
	m.vp.SetContent(renderInfoBody(m.card))
}
