package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/limine"
)

// LimineActions is the set of asynchronous commands the TUI can dispatch
// at darkd to drive the limine service.
type LimineActions struct {
	Create           func(description string) tea.Cmd
	Delete           func(number int) tea.Cmd
	Sync             func() tea.Cmd
	SetDefaultEntry  func(entry int) tea.Cmd
	SetBootConfig    func(key, value string) tea.Cmd
	SetSyncConfig    func(key, value string) tea.Cmd
	SetOmarchyConfig func(key, value string) tea.Cmd
	SetKernelCmdline func(lines []string) tea.Cmd
}

// LimineMsg carries a snapshot published by darkd.
type LimineMsg limine.Snapshot

// LimineActionResultMsg is dispatched when a create/delete/sync command
// completes. The TUI clears the busy indicator and, on success, replaces
// the cached snapshot with the reply's updated one.
type LimineActionResultMsg struct {
	Snapshot limine.Snapshot
	Err      string
}

func (m *Model) triggerLimineCreate() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 {
		return nil
	}
	if m.limine.Create == nil {
		return m.notifyUnavailable("Limine")
	}
	m.dialog = NewDialog("Create snapper snapshot", []DialogFieldSpec{
		{Key: "description", Label: "Description", Value: "dark manual snapshot"},
	}, func(result DialogResult) tea.Cmd {
		desc := result["description"]
		m.state.LimineBusy = true
		m.state.LimineActionError = ""
		return m.limine.Create(desc)
	})
	return nil
}

func (m *Model) triggerLimineSync() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 {
		return nil
	}
	if m.limine.Sync == nil {
		return m.notifyUnavailable("Limine")
	}
	m.dialog = NewDialog("Run limine-snapper-sync now?", nil, func(_ DialogResult) tea.Cmd {
		m.state.LimineBusy = true
		m.state.LimineActionError = ""
		return m.limine.Sync()
	})
	return nil
}

func (m *Model) triggerLimineDelete() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.LimineContentFocused {
		return nil
	}
	if m.limine.Delete == nil {
		return m.notifyUnavailable("Limine")
	}
	snap, ok := m.state.SelectedLimineSnapshot()
	if !ok || snap.Number <= 0 {
		return nil
	}
	num := snap.Number
	label := snap.Timestamp
	m.dialog = NewDialog("Delete snapshot "+label+"?", nil, func(_ DialogResult) tea.Cmd {
		m.state.LimineBusy = true
		m.state.LimineActionError = ""
		return m.limine.Delete(num)
	})
	return nil
}

// triggerLimineEditRow routes enter-on-a-config-row to the right
// setter. For scalar rows it opens a pre-filled single-field dialog;
// for the virtual kernel_cmdline row it opens a multi-field dialog
// covering every current cmdline line plus a couple of blanks for
// additions.
func (m *Model) triggerLimineEditRow() tea.Cmd {
	if m.state.ActiveTab != core.TabF3 || !m.state.LimineContentFocused {
		return nil
	}
	row, kind, ok := m.state.SelectedLimineConfigRow()
	if !ok {
		return nil
	}
	switch kind {
	case "boot":
		return m.openLimineScalarEdit("Edit "+row.Label, row, m.limine.SetBootConfig)
	case "sync":
		return m.openLimineScalarEdit("Edit "+row.Label, row, m.limine.SetSyncConfig)
	case "omarchy":
		return m.openLimineScalarEdit("Edit "+row.Label, row, m.limine.SetOmarchyConfig)
	case "omarchy_cmdline":
		return m.openLimineCmdlineEdit()
	}
	return nil
}

func (m *Model) openLimineScalarEdit(title string, row core.LimineConfigRow, setter func(key, value string) tea.Cmd) tea.Cmd {
	if setter == nil {
		return m.notifyUnavailable("Limine")
	}
	m.dialog = NewDialog(title, []DialogFieldSpec{
		{Key: "value", Label: row.Key, Value: row.Value},
	}, func(result DialogResult) tea.Cmd {
		val := result["value"]
		m.state.LimineBusy = true
		m.state.LimineActionError = ""
		if row.Key == "default_entry" && m.limine.SetDefaultEntry != nil {
			if n, err := strconv.Atoi(val); err == nil {
				return m.limine.SetDefaultEntry(n)
			}
		}
		return setter(row.Key, val)
	})
	return nil
}

func (m *Model) openLimineCmdlineEdit() tea.Cmd {
	if m.limine.SetKernelCmdline == nil {
		return m.notifyUnavailable("Limine")
	}
	existing := m.state.Limine.OmarchyConfig.KernelCmdline
	fields := make([]DialogFieldSpec, 0, len(existing)+2)
	for i, line := range existing {
		fields = append(fields, DialogFieldSpec{
			Key:   fmt.Sprintf("line%d", i),
			Label: fmt.Sprintf("line %d", i+1),
			Value: strings.TrimSpace(line),
		})
	}
	// Two extra blank slots so users can append without re-editing
	// every existing row. Blank submissions are dropped.
	for i := 0; i < 2; i++ {
		fields = append(fields, DialogFieldSpec{
			Key:   fmt.Sprintf("new%d", i),
			Label: "(add)",
		})
	}
	m.dialog = NewDialog("Edit KERNEL_CMDLINE[default]", fields, func(result DialogResult) tea.Cmd {
		var lines []string
		for i := range existing {
			if v := strings.TrimSpace(result[fmt.Sprintf("line%d", i)]); v != "" {
				lines = append(lines, v)
			}
		}
		for i := 0; i < 2; i++ {
			if v := strings.TrimSpace(result[fmt.Sprintf("new%d", i)]); v != "" {
				lines = append(lines, v)
			}
		}
		m.state.LimineBusy = true
		m.state.LimineActionError = ""
		return m.limine.SetKernelCmdline(lines)
	})
	return nil
}
