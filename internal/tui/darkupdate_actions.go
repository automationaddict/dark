package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/darkupdate"
)

// DarkUpdateActions is the set of asynchronous commands the
// self-update panel dispatches at darkd. Check queries GitHub for
// the latest release; Apply downloads + installs the current
// Latest tag.
type DarkUpdateActions struct {
	Check func() tea.Cmd
	Apply func() tea.Cmd
}

// DarkUpdateMsg carries a snapshot published by darkd.
type DarkUpdateMsg darkupdate.Snapshot

// DarkUpdateActionResultMsg is the reply from a check/apply
// command.
type DarkUpdateActionResultMsg struct {
	Snapshot darkupdate.Snapshot
	Err      string
	Action   string // "check" or "apply"
}

// inDarkUpdateContent is the focus gate for self-update triggers.
// Only the Dark sub-section of F2 → Updates accepts these keys.
func (m *Model) inDarkUpdateContent() bool {
	return m.state.ActiveTab == core.TabF2 &&
		m.state.F2OnUpdates() &&
		m.state.ActiveUpdateSection().ID == "dark"
}

// triggerDarkUpdateCheck fires the CheckLatest command.
func (m *Model) triggerDarkUpdateCheck() tea.Cmd {
	if !m.inDarkUpdateContent() {
		return nil
	}
	if m.darkupdate.Check == nil {
		return m.notifyUnavailable("Dark update")
	}
	m.state.DarkUpdateChecking = true
	m.state.DarkUpdateActionError = ""
	return m.darkupdate.Check()
}

// triggerDarkUpdateApply opens a confirmation dialog and, on
// confirm, fires the Apply command. Apply downloads + installs
// and is not reversible without another install, so confirm
// is warranted.
func (m *Model) triggerDarkUpdateApply() tea.Cmd {
	if !m.inDarkUpdateContent() {
		return nil
	}
	if m.darkupdate.Apply == nil {
		return m.notifyUnavailable("Dark update")
	}
	if !m.state.DarkUpdate.UpdateAvailable {
		m.notifyError("Dark update", "no update available — press c to check first")
		return nil
	}
	actionsRef := m.darkupdate
	latest := m.state.DarkUpdate.Latest
	m.dialog = NewDialog("Install dark "+latest+"?", nil, func(_ DialogResult) tea.Cmd {
		m.state.DarkUpdateApplying = true
		m.state.DarkUpdateActionError = ""
		return actionsRef.Apply()
	})
	return nil
}
