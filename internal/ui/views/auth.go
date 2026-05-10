package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/telegram"
	"github.com/strawberry-code/tuilegram/internal/ui/components"
)

// AuthStep rappresenta lo step corrente del flusso di autenticazione.
type AuthStep int

const (
	AuthStepPhone AuthStep = iota
	AuthStepCode
	AuthStepPassword
)

// AuthModel gestisce il flusso di autenticazione.
type AuthModel struct {
	Step     AuthStep
	phone    textinput.Model
	otp      components.OTPInputModel
	password textinput.Model
	Width    int
	Height   int
	Err      string
	loading  bool
	Bridge   *telegram.Bridge

	// Dati di sessione auth
	phoneNum string
	codeHash string
}

// NewAuthModel crea un nuovo modello di autenticazione.
func NewAuthModel() AuthModel {
	phone := textinput.New()
	phone.Placeholder = "+39 123 456 7890"
	phone.CharLimit = 20
	phone.Focus()

	password := textinput.New()
	password.EchoMode = textinput.EchoPassword
	password.EchoCharacter = '*'
	password.CharLimit = 128

	return AuthModel{
		Step:     AuthStepPhone,
		phone:    phone,
		otp:      components.NewOTPInput(6),
		password: password,
	}
}

func (m AuthModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AuthModel) Update(msg tea.Msg) (AuthModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	case tea.KeyMsg:
		if !m.loading {
			am, cmd := m.handleKey(msg)
			if cmd != nil {
				return am, cmd
			}
			m = am
		}
	}

	// Gestione risposte Telegram
	if cmd := m.handleTelegramMsg(msg); cmd != nil {
		return m, cmd
	}

	return m.updateActiveInput(msg)
}

func (m AuthModel) updateActiveInput(msg tea.Msg) (AuthModel, tea.Cmd) {
	if m.loading {
		return m, nil
	}
	var cmd tea.Cmd
	switch m.Step {
	case AuthStepPhone:
		m.phone, cmd = m.phone.Update(msg)
	case AuthStepCode:
		m.otp, cmd = m.otp.Update(msg)
	case AuthStepPassword:
		m.password, cmd = m.password.Update(msg)
	}
	return m, cmd
}
