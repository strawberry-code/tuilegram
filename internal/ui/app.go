package ui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/strawberry-code/tuilegram/internal/config"
	"github.com/strawberry-code/tuilegram/internal/telegram"
	"github.com/strawberry-code/tuilegram/internal/ui/views"
)

const minSplashDuration = 1500 * time.Millisecond

type splashDoneMsg struct{}

// AppState rappresenta lo stato top-level dell'applicazione.
type AppState int

const (
	StateInitializing AppState = iota
	StateAuth
	StateMain
)

// AppModel è il root model dell'applicazione.
type AppModel struct {
	state      AppState
	auth       views.AuthModel
	main       views.MainModel
	bridge     *telegram.Bridge
	spinner    spinner.Model
	width      int
	height     int
	splashDone bool
	pendingMsg tea.Msg
}

// NewAppModel crea il model iniziale con il bridge Telegram e la config (ADR-019 D6).
// cfg.Display.CompactThreshold parametrizza la soglia responsive (Step 31).
func NewAppModel(bridge *telegram.Bridge, cfg config.Config) AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot

	auth := views.NewAuthModel()
	auth.Bridge = bridge

	return AppModel{
		state:   StateInitializing,
		auth:    auth,
		main:    views.NewMainModel(cfg.Display.CompactThreshold),
		bridge:  bridge,
		spinner: s,
	}
}

func (m AppModel) Init() tea.Cmd {
	splashTimer := tea.Tick(minSplashDuration, func(time.Time) tea.Msg {
		return splashDoneMsg{}
	})
	return tea.Batch(m.spinner.Tick, m.auth.Init(), splashTimer)
}
