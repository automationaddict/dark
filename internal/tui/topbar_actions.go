package tui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/topbar"
)

// TopBarActions is the set of asynchronous commands the Appearance →
// Top Bar sub-section can dispatch. Each entry returns a tea.Cmd that
// fires a NATS request and posts a typed reply back into the Update
// loop. Same shape as every other service's action struct.
type TopBarActions struct {
	SetRunning  func(running bool) tea.Cmd
	Restart     func() tea.Cmd
	Reset       func() tea.Cmd
	SetPosition func(value string) tea.Cmd
	SetLayer    func(value string) tea.Cmd
	SetHeight   func(value int) tea.Cmd
	SetSpacing  func(value int) tea.Cmd
	SetConfig   func(content string) tea.Cmd
	SetStyle    func(content string) tea.Cmd
}

// TopBarMsg carries a snapshot published by darkd.
type TopBarMsg topbar.Snapshot

// TopBarActionResultMsg is the reply from an action command.
type TopBarActionResultMsg struct {
	Snapshot topbar.Snapshot
	Err      string
}

// inTopBarContent is the focus gate every top bar trigger checks: we
// only react to action keys when the user is focused on the
// Appearance → Top Bar sub-section and has pressed enter to move
// focus into the content region.
func (m *Model) inTopBarContent() bool {
	if !m.state.ContentFocused {
		return false
	}
	if m.state.ActiveTab != core.TabSettings || m.state.ActiveSection().ID != "appearance" {
		return false
	}
	return m.state.ActiveAppearanceSection().ID == "topbar"
}

// triggerTopBarToggle flips the waybar running state.
func (m *Model) triggerTopBarToggle() tea.Cmd {
	if !m.inTopBarContent() {
		return nil
	}
	if m.topbar.SetRunning == nil {
		return m.notifyUnavailable("Top Bar")
	}
	target := !m.state.TopBar.Running
	m.state.TopBarBusy = true
	m.state.TopBarActionError = ""
	return m.topbar.SetRunning(target)
}

// triggerTopBarRestart stops and restarts waybar.
func (m *Model) triggerTopBarRestart() tea.Cmd {
	if !m.inTopBarContent() {
		return nil
	}
	if m.topbar.Restart == nil {
		return m.notifyUnavailable("Top Bar")
	}
	m.state.TopBarBusy = true
	m.state.TopBarActionError = ""
	return m.topbar.Restart()
}

// triggerTopBarReset opens a confirmation dialog before dispatching
// the Reset command. Reset rewrites the user's config.jsonc and
// style.css with Omarchy defaults; the backend backs up the
// existing files with a timestamp suffix, so the action is
// recoverable, but the confirmation is still warranted because a
// live waybar restart follows immediately.
func (m *Model) triggerTopBarReset() tea.Cmd {
	if !m.inTopBarContent() {
		return nil
	}
	if m.topbar.Reset == nil {
		return m.notifyUnavailable("Top Bar")
	}
	if !m.state.TopBar.DefaultsAvailable {
		m.notifyError("Top Bar", "Omarchy defaults directory not found — reset unavailable")
		return nil
	}
	actionsRef := m.topbar
	m.dialog = NewDialog("Reset waybar config to Omarchy defaults? (existing files are backed up)", nil, func(_ DialogResult) tea.Cmd {
		m.state.TopBarBusy = true
		m.state.TopBarActionError = ""
		return actionsRef.Reset()
	})
	return nil
}

// triggerTopBarCyclePosition rotates through the four legal
// positions. One key press moves to the next value and restarts
// waybar, so the effect is immediate.
func (m *Model) triggerTopBarCyclePosition() tea.Cmd {
	if !m.inTopBarContent() {
		return nil
	}
	if m.topbar.SetPosition == nil {
		return m.notifyUnavailable("Top Bar")
	}
	order := []string{"top", "bottom", "left", "right"}
	next := nextInCycle(order, m.state.TopBar.Position)
	m.state.TopBarBusy = true
	m.state.TopBarActionError = ""
	return m.topbar.SetPosition(next)
}

// triggerTopBarCycleLayer rotates through the two layers dark
// exposes. overlay / top is the only meaningful split for a menu
// bar, so bottom / background aren't in the cycle — set those via
// the raw config editor if needed.
func (m *Model) triggerTopBarCycleLayer() tea.Cmd {
	if !m.inTopBarContent() {
		return nil
	}
	if m.topbar.SetLayer == nil {
		return m.notifyUnavailable("Top Bar")
	}
	order := []string{"top", "overlay"}
	next := nextInCycle(order, m.state.TopBar.Layer)
	m.state.TopBarBusy = true
	m.state.TopBarActionError = ""
	return m.topbar.SetLayer(next)
}

// triggerTopBarHeightDialog opens a single-field dialog for height.
func (m *Model) triggerTopBarHeightDialog() {
	if !m.inTopBarContent() || m.topbar.SetHeight == nil {
		return
	}
	actionsRef := m.topbar
	m.dialog = NewDialog("Top bar height (pixels, 0..200)", []DialogFieldSpec{
		{Key: "height", Label: "Height", Value: strconv.Itoa(m.state.TopBar.Height)},
	}, func(result DialogResult) tea.Cmd {
		raw := result["height"]
		value, err := strconv.Atoi(raw)
		if err != nil {
			m.notifyError("Top Bar", fmt.Sprintf("height must be a number, got %q", raw))
			return nil
		}
		m.state.TopBarBusy = true
		m.state.TopBarActionError = ""
		return actionsRef.SetHeight(value)
	})
}

// triggerTopBarSpacingDialog opens a single-field dialog for spacing.
func (m *Model) triggerTopBarSpacingDialog() {
	if !m.inTopBarContent() || m.topbar.SetSpacing == nil {
		return
	}
	actionsRef := m.topbar
	m.dialog = NewDialog("Top bar module spacing (pixels, 0..200)", []DialogFieldSpec{
		{Key: "spacing", Label: "Spacing", Value: strconv.Itoa(m.state.TopBar.Spacing)},
	}, func(result DialogResult) tea.Cmd {
		raw := result["spacing"]
		value, err := strconv.Atoi(raw)
		if err != nil {
			m.notifyError("Top Bar", fmt.Sprintf("spacing must be a number, got %q", raw))
			return nil
		}
		m.state.TopBarBusy = true
		m.state.TopBarActionError = ""
		return actionsRef.SetSpacing(value)
	})
}

// triggerTopBarEditConfig opens the full-screen editor pre-filled
// with the current config.jsonc content. Ctrl+S dispatches
// SetConfig which writes and restarts waybar.
func (m *Model) triggerTopBarEditConfig() tea.Cmd {
	if !m.inTopBarContent() || m.topbar.SetConfig == nil {
		return nil
	}
	actionsRef := m.topbar
	m.editor = NewEditorWithLanguage(
		"Top bar config (config.jsonc)",
		LangJSONC,
		m.state.TopBar.Config,
		m.width, m.height,
		func(content string) tea.Cmd {
			m.state.TopBarBusy = true
			m.state.TopBarActionError = ""
			return actionsRef.SetConfig(content)
		},
	)
	return nil
}

// triggerTopBarEditStyle opens the full-screen editor pre-filled
// with the current style.css content. Ctrl+S dispatches SetStyle
// which writes but does NOT restart waybar (reload_style_on_change
// handles it live).
func (m *Model) triggerTopBarEditStyle() tea.Cmd {
	if !m.inTopBarContent() || m.topbar.SetStyle == nil {
		return nil
	}
	actionsRef := m.topbar
	m.editor = NewEditorWithLanguage(
		"Top bar style (style.css)",
		LangCSS,
		m.state.TopBar.Style,
		m.width, m.height,
		func(content string) tea.Cmd {
			m.state.TopBarBusy = true
			m.state.TopBarActionError = ""
			return actionsRef.SetStyle(content)
		},
	)
	return nil
}

// nextInCycle returns the element after current in order, wrapping
// to the start when current is the last entry or isn't found at all.
func nextInCycle(order []string, current string) string {
	for i, v := range order {
		if v == current && i+1 < len(order) {
			return order[i+1]
		}
	}
	if len(order) == 0 {
		return ""
	}
	return order[0]
}
