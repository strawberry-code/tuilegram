package views

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// CmdPaletteModel is the command palette overlay (Step 28, ADR-015).
// Value-receiver pattern: callers receive state via returned copies.
// Active == true iff palette is mounted (activeOverlay == cmdPalette in root).
type CmdPaletteModel struct {
	Active   bool
	query    string
	filtered []CommandEntry // current filtered list (cmdFuzzyFilter result)
	cursor   int            // clamped to [0, len(filtered)-1]
	input    textinput.Model
	registry CommandRegistry
	// layout dimensions set by root on WindowSizeMsg
	Width, Height int
}

// NewCmdPaletteModel initialises an inactive palette with the default registry.
func NewCmdPaletteModel() CmdPaletteModel {
	ti := textinput.New()
	ti.Placeholder = "Type a command…"
	ti.CharLimit = 64
	return CmdPaletteModel{
		registry: DefaultRegistry,
		input:    ti,
	}
}

// Open transitions Closed → Open: resets state, focuses input, loads full list.
func (m CmdPaletteModel) Open() CmdPaletteModel {
	m.Active = true
	m.query = ""
	m.filtered = cmdFuzzyFilter(m.registry, "")
	m.cursor = 0
	m.input.Reset()
	m.input.Focus()
	return m
}

// Close transitions any Open sub-state → Closed. Blurs input.
func (m CmdPaletteModel) Close() CmdPaletteModel {
	m.Active = false
	m.input.Blur()
	return m
}

// Update dispatches tea.Msg while palette is Active.
// Returns (updated model, cmd). If not Active, returns unchanged.
func (m CmdPaletteModel) Update(msg tea.Msg) (CmdPaletteModel, tea.Cmd) {
	if !m.Active {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	// Forward other msgs (cursor blink, etc.) to textinput.
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the palette modal when Active; returns "" otherwise.
func (m CmdPaletteModel) View() string {
	if !m.Active {
		return ""
	}
	return renderPaletteModal(m)
}
