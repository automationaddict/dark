package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/services/screensaver"
)

// ScreensaverActions is the set of asynchronous commands the
// Appearance → Screensaver sub-section can dispatch at darkd. Each
// returns a tea.Cmd that issues a NATS request and posts a typed
// result back into the Update loop, mirroring the shape of every
// other service's action struct.
type ScreensaverActions struct {
	SetEnabled func(enabled bool) tea.Cmd
	SetContent func(content string) tea.Cmd
	Preview    func() tea.Cmd
}

// ScreensaverMsg is dispatched whenever darkd publishes a screensaver
// snapshot on the periodic publish path.
type ScreensaverMsg screensaver.Snapshot

// ScreensaverActionResultMsg is the reply from a set-enabled /
// set-content / preview command. On success, Snapshot carries the
// refreshed state. Action identifies which command produced the
// reply so the Update handler can decide whether to clear
// ScreensaverPreviewing.
type ScreensaverActionResultMsg struct {
	Snapshot screensaver.Snapshot
	Err      string
	Action   string // "set_enabled", "set_content", "preview"
}
