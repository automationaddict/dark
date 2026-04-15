package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/screensaver"
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

// inScreensaverContent is the gate every screensaver trigger walks
// through before firing. We only react to action keys when the user
// is focused on the Appearance → Screensaver sub-section and has
// pressed enter to move focus into the content region.
func (m *Model) inScreensaverContent() bool {
	if !m.state.ContentFocused {
		return false
	}
	if m.state.ActiveTab != core.TabSettings || m.state.ActiveSection().ID != "appearance" {
		return false
	}
	return m.state.ActiveAppearanceSection().ID == "screensaver"
}

// triggerScreensaverToggle flips the enabled flag and dispatches the
// set_enabled command. Mirrors the shape of every other toggle action.
func (m *Model) triggerScreensaverToggle() tea.Cmd {
	if !m.inScreensaverContent() {
		return nil
	}
	if m.screensaver.SetEnabled == nil {
		return m.notifyUnavailable("Screensaver")
	}
	if !m.state.ScreensaverLoaded {
		return nil
	}
	target := !m.state.Screensaver.Enabled
	m.state.ScreensaverBusy = true
	m.state.ScreensaverActionError = ""
	return m.screensaver.SetEnabled(target)
}

// triggerScreensaverPreview fires the preview command and flips the
// ScreensaverPreviewing flag so the view can show "previewing…" until
// the child exits. The preview blocks the daemon side, so the reply
// only arrives when the screensaver has closed.
func (m *Model) triggerScreensaverPreview() tea.Cmd {
	if !m.inScreensaverContent() {
		return nil
	}
	if m.screensaver.Preview == nil {
		return m.notifyUnavailable("Screensaver")
	}
	if !m.state.Screensaver.Supported {
		m.notifyError("Screensaver", "preview requires tte and a supported terminal (alacritty, ghostty, or kitty)")
		return nil
	}
	m.state.ScreensaverBusy = true
	m.state.ScreensaverPreviewing = true
	m.state.ScreensaverActionError = ""
	m.notifyInfo("Screensaver", "Launching preview — any key exits")
	return m.screensaver.Preview()
}

// triggerScreensaverEditContent writes the current branding content
// to a scratch file and hands off to $EDITOR. On exit Model.Update
// reads the file and dispatches set_content.
func (m *Model) triggerScreensaverEditContent() tea.Cmd {
	if !m.inScreensaverContent() {
		return nil
	}
	if m.screensaver.SetContent == nil {
		return m.notifyUnavailable("Screensaver")
	}
	return editEphemeralContent(editKindScreensaverContent, "screensaver", ".txt", m.state.Screensaver.Content)
}
