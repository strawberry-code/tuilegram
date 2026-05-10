package views

import tea "github.com/charmbracelet/bubbletea"

// main_overlay_update.go — Update handlers for Step 28 overlay messages
// (CmdPalette, WhichKey, Help, scroll/nav side-effects).
// Extracted from main_update.go to satisfy the 120-LOC limit.

// handleOverlayMsg processes all Step-28 overlay tea.Msg types.
// Returns (model, cmd, handled). handled=false means the message was not an
// overlay message and the caller must continue its own switch.
func (m MainModel) handleOverlayMsg(msg tea.Msg) (MainModel, tea.Cmd, bool) {
	switch msg := msg.(type) {
	case CmdPaletteOpenMsg:
		m.cmdPalette = m.cmdPalette.Open()
		return m, nil, true
	case CmdPaletteCloseMsg:
		m.cmdPalette = m.cmdPalette.Close()
		return m, nil, true
	case CmdPaletteSubmitMsg:
		return m, nil, true // closed before submit; cmd already dispatched (ADR-015 §D3)
	case WhichKeyTimeoutMsg:
		m.whichKey = m.whichKey.HandleTimeout(msg)
		return m, nil, true
	case WhichKeyChordMsg:
		var cmd tea.Cmd
		m.whichKey, cmd = m.whichKey.HandleContinuation(msg.Prefix, msg.Cont)
		return m, cmd, true
	case WhichKeyCancelMsg:
		m.whichKey = m.whichKey.Cancel("")
		return m, nil, true
	case HelpOpenMsg:
		m.help = m.help.Open()
		return m, nil, true
	case HelpCloseMsg:
		m.help = m.help.Close()
		return m, nil, true
	case scrollTopMsg:
		m.conversation.viewport.GotoTop()
		return m, nil, true
	case scrollBottomMsg:
		m.conversation.viewport.GotoBottom()
		return m, nil, true
	case scrollCenterMsg:
		return m.centerViewport(), nil, true
	case jumpUnreadMsg:
		m.chatList.JumpToUnread()
		return m, nil, true
	case LogoutRequestMsg:
		return m, m.LogoutCmd(), true
	case openLinkChordMsg:
		// Step 33: gx chord → apre il primo link del messaggio selezionato.
		// BLOCKING #2: handleGxChord restituisce (MainModel, tea.Cmd) non bloccante.
		m2, cmd := m.handleGxChord()
		return m2, cmd, true
	}
	return m, nil, false
}
