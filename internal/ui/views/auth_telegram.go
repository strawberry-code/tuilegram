package views

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
	"github.com/strawberry-code/tuilegram/internal/ui/components"
)

// handleTelegramMsg gestisce le risposte async dal Bridge Telegram.
func (m *AuthModel) handleTelegramMsg(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case telegram.CodeSentMsg:
		m.loading = false
		m.codeHash = msg.CodeHash
		m.otp = components.NewOTPInput(msg.CodeLength)
		m.Step = AuthStepCode
		m.phone.Blur()
		m.otp.Focus()

	case telegram.CodeSentErrMsg:
		m.loading = false
		m.Err = msg.Err.Error()

	case telegram.SignInOkMsg:
		m.loading = false
		// Il Bridge notificherà ConnectedMsg al livello App

	case telegram.SignInErrMsg:
		m.loading = false
		m.Err = msg.Err.Error()
		m.otp.Reset()
		m.otp.Focus()

	case telegram.PasswordRequiredMsg:
		m.loading = false
		m.Step = AuthStepPassword
		m.otp.Blur()
		m.password.Focus()

	case telegram.PasswordOkMsg:
		m.loading = false

	case telegram.PasswordErrMsg:
		m.loading = false
		m.Err = msg.Err.Error()
		m.password.Reset()
		m.password.Focus()

	default:
		return nil
	}

	return nil
}
