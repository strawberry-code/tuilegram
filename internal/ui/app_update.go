package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
)

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.auth.Width = msg.Width
		m.auth.Height = msg.Height
		m.main.SetSize(msg.Width, msg.Height)
	case tea.KeyMsg:
		if msg.String() == "ctrl+q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case splashDoneMsg:
		m.splashDone = true
		if m.pendingMsg != nil {
			return m.Update(m.pendingMsg)
		}
	case telegram.AuthRequiredMsg:
		if !m.splashDone {
			m.pendingMsg = msg
			return m, nil
		}
		m.state = StateAuth
		m.auth.Width = m.width
		m.auth.Height = m.height
	case telegram.ConnectedMsg:
		if !m.splashDone {
			m.pendingMsg = msg
			return m, nil
		}
		m.state = StateMain
		m.main.SetBridge(m.bridge)
		m.main.SetSize(m.width, m.height)
		return m, tea.Batch(m.main.LoadDialogsCmd(), m.main.LoadFoldersCmd())
	}

	return m.updateState(msg)
}

func (m AppModel) updateState(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.state {
	case StateInitializing:
		m.spinner, cmd = m.spinner.Update(msg)
	case StateAuth:
		m.auth, cmd = m.auth.Update(msg)
	case StateMain:
		m.main, cmd = m.main.Update(msg)
	}
	return m, cmd
}
