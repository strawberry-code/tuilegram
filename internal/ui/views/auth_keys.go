package views

import (
	"context"
	"errors"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gotd/td/tgerr"

	"github.com/strawberry-code/tuilegram/internal/telegram"
)

func (m AuthModel) handleKey(msg tea.KeyMsg) (AuthModel, tea.Cmd) {
	switch msg.String() {
	case "enter":
		return m.handleEnter()
	case "esc":
		m.handleEsc()
	}
	return m, nil
}

func (m AuthModel) handleEnter() (AuthModel, tea.Cmd) {
	m.Err = ""
	switch m.Step {
	case AuthStepPhone:
		phone := strings.TrimSpace(m.phone.Value())
		if phone == "" {
			return m, nil
		}
		m.phoneNum = phone
		m.loading = true
		return m, m.sendCodeCmd(phone)
	case AuthStepCode:
		if !m.otp.IsFull() {
			return m, nil
		}
		m.loading = true
		return m, m.signInCmd(m.phoneNum, m.otp.Value(), m.codeHash)
	case AuthStepPassword:
		if m.password.Value() == "" {
			return m, nil
		}
		m.loading = true
		return m, m.checkPasswordCmd(m.password.Value())
	}
	return m, nil
}

func (m *AuthModel) handleEsc() {
	m.Err = ""
	switch m.Step {
	case AuthStepCode:
		m.Step = AuthStepPhone
		m.otp.Blur()
		m.otp.Reset()
		m.phone.Focus()
	case AuthStepPassword:
		m.Step = AuthStepCode
		m.password.Blur()
		m.password.Reset()
		m.otp.Focus()
	}
}

func (m AuthModel) sendCodeCmd(phone string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.Bridge.SendCode(context.Background(), phone)
		if err != nil {
			return telegram.CodeSentErrMsg{Err: err}
		}
		return telegram.CodeSentMsg(result)
	}
}

func (m AuthModel) signInCmd(phone, code, hash string) tea.Cmd {
	return func() tea.Msg {
		err := m.Bridge.SignIn(context.Background(), phone, code, hash)
		if err != nil {
			if isPasswordRequired(err) {
				return telegram.PasswordRequiredMsg{}
			}
			return telegram.SignInErrMsg{Err: err}
		}
		return telegram.SignInOkMsg{}
	}
}

func (m AuthModel) checkPasswordCmd(password string) tea.Cmd {
	return func() tea.Msg {
		err := m.Bridge.CheckPassword(context.Background(), password)
		if err != nil {
			return telegram.PasswordErrMsg{Err: err}
		}
		return telegram.PasswordOkMsg{}
	}
}

// isPasswordRequired verifica se l'errore indica che serve la password 2FA.
// Controlla in più modi perché gotd/td può restituire l'errore in formati diversi.
func isPasswordRequired(err error) bool {
	if tgerr.Is(err, "SESSION_PASSWORD_NEEDED") {
		return true
	}
	var rpcErr *tgerr.Error
	if errors.As(err, &rpcErr) {
		if rpcErr.Type == "SESSION_PASSWORD_NEEDED" {
			return true
		}
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "session_password_needed") ||
		strings.Contains(errStr, "2fa required") ||
		strings.Contains(errStr, "password") && strings.Contains(errStr, "required")
}
