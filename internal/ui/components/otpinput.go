package components

import (
	tea "github.com/charmbracelet/bubbletea"
)

// OTPInputModel è un componente custom per input a celle individuali (stile OTP/PIN).
// Ogni cella contiene un singolo digit. Auto-advance alla cella successiva.
type OTPInputModel struct {
	cells   []rune
	length  int
	cursor  int
	focused bool
}

// NewOTPInput crea un OTP input con il numero di celle specificato.
func NewOTPInput(length int) OTPInputModel {
	return OTPInputModel{
		cells:   make([]rune, length),
		length:  length,
		cursor:  0,
		focused: true,
	}
}

// Focus attiva il componente.
func (m *OTPInputModel) Focus() { m.focused = true }

// Blur disattiva il componente.
func (m *OTPInputModel) Blur() { m.focused = false }

// Value restituisce il codice inserito come stringa.
func (m OTPInputModel) Value() string {
	result := make([]rune, 0, m.length)
	for _, c := range m.cells {
		if c == 0 {
			break
		}
		result = append(result, c)
	}
	return string(result)
}

// IsFull restituisce true se tutte le celle sono compilate.
func (m OTPInputModel) IsFull() bool {
	for _, c := range m.cells {
		if c == 0 {
			return false
		}
	}
	return true
}

// Reset svuota tutte le celle e riporta il cursore all'inizio.
func (m *OTPInputModel) Reset() {
	m.cells = make([]rune, m.length)
	m.cursor = 0
}

func (m OTPInputModel) Update(msg tea.Msg) (OTPInputModel, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "backspace":
		if m.cursor > 0 {
			m.cursor--
			m.cells[m.cursor] = 0
		}
	case "left":
		if m.cursor > 0 {
			m.cursor--
		}
	case "right":
		if m.cursor < m.length-1 {
			m.cursor++
		}
	default:
		r := []rune(keyMsg.String())
		if len(r) == 1 && r[0] >= '0' && r[0] <= '9' {
			m.cells[m.cursor] = r[0]
			if m.cursor < m.length-1 {
				m.cursor++
			}
		}
	}

	return m, nil
}
