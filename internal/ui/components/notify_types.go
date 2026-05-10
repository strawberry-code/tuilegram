package components

// notify_types.go — Type definitions and constants for notify component.
// Split from notify.go to respect 120-LOC rule.

type NotifyKind int

const (
	NotifyInfo NotifyKind = iota
	NotifySuccess
	NotifyError
)

type NotifyMsg struct {
	Kind NotifyKind
	Text string
}
type notifyTickMsg struct{ gen int }
type notifyTimeoutMsg struct{ gen int }

type notifyState int

const (
	nsIdle notifyState = iota
	nsMounting
	nsVisible
	nsDismissing
)

const (
	notifyFrames = 3
	notifyTickMs = 60
	notifyHoldMs = 3000
)

// IsNotifyMsg returns true if msg targets the NotifyModel (NotifyMsg or its internal ticks).
// Used by routers (e.g., MainModel.Update) to dispatch only relevant messages.
func IsNotifyMsg(msg interface{}) bool {
	switch msg.(type) {
	case NotifyMsg, notifyTickMsg, notifyTimeoutMsg:
		return true
	}
	return false
}
