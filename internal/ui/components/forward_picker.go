package components

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/model"
)

// pickerState mirrors the forward-picker statechart states.
type pickerState int

const (
	pickerClosed     pickerState = iota
	pickerFiltering              // Filtering (Idle + Typing substates collapsed)
	pickerForwarding             // RPCInFlight
)

// ForwardPickerModel is the forward-picker overlay (Step 21).
// Value receiver — caller owns state transitions via returned copies.
type ForwardPickerModel struct {
	state    pickerState
	allChats []model.Chat // snapshot delivered by ForwardPickerReadyMsg
	filtered []model.Chat // re-ranked result of current query
	cursor   int
	input    textinput.Model
	lastErr  error // set when RPC failed; cleared on next query change
}

// NewForwardPicker creates an inactive forward picker.
func NewForwardPicker() ForwardPickerModel {
	ti := textinput.New()
	ti.Placeholder = "Forward to..."
	ti.CharLimit = 64
	return ForwardPickerModel{input: ti}
}

// Active returns true when the overlay is visible (Filtering or Forwarding).
func (m ForwardPickerModel) Active() bool {
	return m.state != pickerClosed
}

// Forwarding returns true when the RPC is in flight.
func (m ForwardPickerModel) Forwarding() bool {
	return m.state == pickerForwarding
}

// Open transitions Closed → Filtering with a chat snapshot.
func (m ForwardPickerModel) Open(chats []model.Chat) ForwardPickerModel {
	m.state = pickerFiltering
	m.allChats = chats
	m.filtered = chats
	m.cursor = 0
	m.lastErr = nil
	m.input.Reset()
	m.input.Focus()
	return m
}

// Close transitions any state → Closed.
func (m ForwardPickerModel) Close() ForwardPickerModel {
	m.state = pickerClosed
	m.input.Blur()
	return m
}

// BeginForwarding transitions Filtering → Forwarding.
func (m ForwardPickerModel) BeginForwarding() ForwardPickerModel {
	m.state = pickerForwarding
	m.input.Blur()
	return m
}

// EndForwarding transitions Forwarding → Closed (nil err) or → Filtering (err).
func (m ForwardPickerModel) EndForwarding(err error) ForwardPickerModel {
	if err == nil {
		return m.Close()
	}
	m.state = pickerFiltering
	m.lastErr = err
	m.input.Focus()
	return m
}

// Selected returns the chat under the cursor, if any.
func (m ForwardPickerModel) Selected() (model.Chat, bool) {
	if len(m.filtered) == 0 || m.cursor >= len(m.filtered) {
		return model.Chat{}, false
	}
	return m.filtered[m.cursor], true
}

// Update handles tea.Msg. Returns (ForwardPickerModel, tea.Cmd).
func (m ForwardPickerModel) Update(msg tea.Msg) (ForwardPickerModel, tea.Cmd) {
	return pickerUpdate(m, msg)
}

// View renders the overlay at the given dimensions.
func (m ForwardPickerModel) View(width, height int) string {
	return pickerView(m, width, height)
}
