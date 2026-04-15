package tui

import (
	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/workspaces"
)

// WorkspacesActions is the set of asynchronous commands the
// Workspaces panel dispatches at darkd. Each closure fires a NATS
// request and posts the reply back into the Update loop.
type WorkspacesActions struct {
	Switch            func(id string) tea.Cmd
	Rename            func(id int, name string) tea.Cmd
	MoveToMonitor     func(id int, monitor string) tea.Cmd
	SetLayout         func(id int, layout string) tea.Cmd
	SetDefaultLayout  func(layout string) tea.Cmd
	SetDwindleOption  func(key, value string) tea.Cmd
	SetMasterOption   func(key, value string) tea.Cmd
	SetCursorWarp     func(enabled bool) tea.Cmd
	SetAnimations     func(enabled bool) tea.Cmd
	SetHideSpecial    func(enabled bool) tea.Cmd
}

// WorkspacesMsg carries a snapshot published by darkd.
type WorkspacesMsg workspaces.Snapshot

// WorkspacesActionResultMsg is the reply from an action command.
type WorkspacesActionResultMsg struct {
	Snapshot workspaces.Snapshot
	Err      string
}

// inWorkspacesContent is the focus gate every workspaces trigger
// checks: user is on the Workspaces sub-section of Settings and
// has moved focus into the content region.
func (m *Model) inWorkspacesContent() bool {
	if !m.state.ContentFocused {
		return false
	}
	if m.state.ActiveTab != core.TabSettings || m.state.ActiveSection().ID != "workspaces" {
		return false
	}
	return true
}

// inWorkspacesOverview narrows to the Overview sub-section —
// actions that operate on the selected workspace row only apply
// there, not on Layout or Behavior.
func (m *Model) inWorkspacesOverview() bool {
	return m.inWorkspacesContent() && m.state.ActiveWorkspacesSection().ID == "overview"
}

// inWorkspacesLayout / inWorkspacesBehavior narrow to the sub-
// sections where the layout options and behavior flags live.
func (m *Model) inWorkspacesLayout() bool {
	return m.inWorkspacesContent() && m.state.ActiveWorkspacesSection().ID == "layout"
}

func (m *Model) inWorkspacesBehavior() bool {
	return m.inWorkspacesContent() && m.state.ActiveWorkspacesSection().ID == "behavior"
}

// triggerWorkspaceSwitch dispatches a switch to the highlighted
// workspace on the Overview sub-section. enter key.
func (m *Model) triggerWorkspaceSwitch() tea.Cmd {
	if !m.inWorkspacesOverview() {
		return nil
	}
	if m.workspaces.Switch == nil {
		return m.notifyUnavailable("Workspaces")
	}
	ws, ok := m.state.SelectedWorkspace()
	if !ok {
		return nil
	}
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.Switch(strconv.Itoa(ws.ID))
}

// triggerWorkspaceRename opens a single-field dialog for renaming
// the highlighted workspace. `r` on the Overview sub-section.
func (m *Model) triggerWorkspaceRename() tea.Cmd {
	if !m.inWorkspacesOverview() {
		return nil
	}
	if m.workspaces.Rename == nil {
		return m.notifyUnavailable("Workspaces")
	}
	ws, ok := m.state.SelectedWorkspace()
	if !ok {
		return nil
	}
	actionsRef := m.workspaces
	id := ws.ID
	m.dialog = NewDialog(fmt.Sprintf("Rename workspace %d", ws.ID), []DialogFieldSpec{
		{Key: "name", Label: "Name", Value: ws.Name},
	}, func(result DialogResult) tea.Cmd {
		name := result["name"]
		if name == "" {
			return nil
		}
		m.state.WorkspacesBusy = true
		m.state.WorkspacesActionError = ""
		return actionsRef.Rename(id, name)
	})
	return nil
}

// triggerWorkspaceMoveToMonitor opens a select dialog listing
// every monitor and moves the highlighted workspace to the
// chosen one. `m` on Overview.
func (m *Model) triggerWorkspaceMoveToMonitor() tea.Cmd {
	if !m.inWorkspacesOverview() {
		return nil
	}
	if m.workspaces.MoveToMonitor == nil {
		return m.notifyUnavailable("Workspaces")
	}
	ws, ok := m.state.SelectedWorkspace()
	if !ok {
		return nil
	}
	// Build the monitor list from the display snapshot — we
	// already have that data in state and it's fresher than
	// anything we'd derive from the workspaces payload.
	monitors := make([]string, 0, len(m.state.Display.Monitors))
	for _, mon := range m.state.Display.Monitors {
		monitors = append(monitors, mon.Name)
	}
	if len(monitors) < 2 {
		m.notifyError("Workspaces", "only one monitor is connected")
		return nil
	}
	actionsRef := m.workspaces
	id := ws.ID
	m.dialog = NewDialog(fmt.Sprintf("Move workspace %d to monitor", ws.ID), []DialogFieldSpec{
		{Key: "monitor", Label: "Monitor", Kind: DialogFieldSelect, Options: monitors, Value: ws.Monitor},
	}, func(result DialogResult) tea.Cmd {
		monitor := result["monitor"]
		if monitor == "" {
			return nil
		}
		m.state.WorkspacesBusy = true
		m.state.WorkspacesActionError = ""
		return actionsRef.MoveToMonitor(id, monitor)
	})
	return nil
}

// triggerWorkspaceLayoutCycle flips the highlighted workspace
// between dwindle and master. `L` on Overview.
func (m *Model) triggerWorkspaceLayoutCycle() tea.Cmd {
	if !m.inWorkspacesOverview() {
		return nil
	}
	if m.workspaces.SetLayout == nil {
		return m.notifyUnavailable("Workspaces")
	}
	ws, ok := m.state.SelectedWorkspace()
	if !ok {
		return nil
	}
	next := "dwindle"
	if ws.TiledLayout == "dwindle" {
		next = "master"
	}
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.SetLayout(ws.ID, next)
}

// triggerWorkspaceDefaultLayoutCycle cycles the global default
// layout between dwindle and master. `L` on Layout sub-section.
func (m *Model) triggerWorkspaceDefaultLayoutCycle() tea.Cmd {
	if !m.inWorkspacesLayout() {
		return nil
	}
	if m.workspaces.SetDefaultLayout == nil {
		return m.notifyUnavailable("Workspaces")
	}
	next := "dwindle"
	if m.state.Workspaces.DefaultLayout == "dwindle" {
		next = "master"
	}
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.SetDefaultLayout(next)
}

// triggerWorkspaceDwindleToggle and triggerWorkspaceMasterCycle
// are per-option flip helpers used by the Layout sub-section's
// key binds.

func (m *Model) triggerWorkspaceDwindleToggle(key string, current bool) tea.Cmd {
	if !m.inWorkspacesLayout() {
		return nil
	}
	if m.workspaces.SetDwindleOption == nil {
		return m.notifyUnavailable("Workspaces")
	}
	next := "true"
	if current {
		next = "false"
	}
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.SetDwindleOption(key, next)
}

func (m *Model) triggerWorkspaceDwindleForceSplitCycle() tea.Cmd {
	if !m.inWorkspacesLayout() {
		return nil
	}
	if m.workspaces.SetDwindleOption == nil {
		return m.notifyUnavailable("Workspaces")
	}
	// force_split: 0 = auto, 1 = always left/top, 2 = always
	// right/bottom. Cycle through all three.
	next := (m.state.Workspaces.Dwindle.ForceSplit + 1) % 3
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.SetDwindleOption("force_split", strconv.Itoa(next))
}

func (m *Model) triggerWorkspaceMasterStatusCycle() tea.Cmd {
	if !m.inWorkspacesLayout() {
		return nil
	}
	if m.workspaces.SetMasterOption == nil {
		return m.notifyUnavailable("Workspaces")
	}
	order := []string{"master", "slave", "inherit"}
	next := nextInCycle(order, m.state.Workspaces.Master.NewStatus)
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.SetMasterOption("new_status", next)
}

// Behavior toggles — one per key on the Behavior sub-section.

func (m *Model) triggerWorkspaceCursorWarpToggle() tea.Cmd {
	if !m.inWorkspacesBehavior() {
		return nil
	}
	if m.workspaces.SetCursorWarp == nil {
		return m.notifyUnavailable("Workspaces")
	}
	target := !m.state.Workspaces.CursorWarp
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.SetCursorWarp(target)
}

func (m *Model) triggerWorkspaceAnimationsToggle() tea.Cmd {
	if !m.inWorkspacesBehavior() {
		return nil
	}
	if m.workspaces.SetAnimations == nil {
		return m.notifyUnavailable("Workspaces")
	}
	target := !m.state.Workspaces.AnimationsEnabled
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.SetAnimations(target)
}

func (m *Model) triggerWorkspaceHideSpecialToggle() tea.Cmd {
	if !m.inWorkspacesBehavior() {
		return nil
	}
	if m.workspaces.SetHideSpecial == nil {
		return m.notifyUnavailable("Workspaces")
	}
	target := !m.state.Workspaces.HideSpecialOnChange
	m.state.WorkspacesBusy = true
	m.state.WorkspacesActionError = ""
	return m.workspaces.SetHideSpecial(target)
}
