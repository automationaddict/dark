package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/notifycfg"
)

type NotifyConfigActions struct {
	ToggleDND  func() tea.Cmd
	DismissAll func() tea.Cmd
}

type NotifyCfgMsg notifycfg.Snapshot

type NotifyCfgActionResultMsg struct {
	Snapshot notifycfg.Snapshot
	Err      string
}

func (m *Model) inNotifyContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "notifications"
}

func (m *Model) triggerNotifyDNDToggle() tea.Cmd {
	if m.notifyCfg.ToggleDND == nil || !m.inNotifyContent() {
		return nil
	}
	return m.notifyCfg.ToggleDND()
}

func (m *Model) triggerNotifyDismissAll() tea.Cmd {
	if m.notifyCfg.DismissAll == nil || !m.inNotifyContent() {
		return nil
	}
	return m.notifyCfg.DismissAll()
}
