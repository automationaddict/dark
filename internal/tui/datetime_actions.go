package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/datetime"
)

type DateTimeActions struct {
	SetTimezone       func(tz string) tea.Cmd
	SetNTP            func(enabled bool) tea.Cmd
	ToggleClockFormat func() tea.Cmd
}

type DateTimeMsg datetime.Snapshot

type DateTimeActionResultMsg struct {
	Snapshot datetime.Snapshot
	Err      string
}

func (m *Model) inDateTimeContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "datetime"
}

func (m *Model) triggerDTTimezoneDialog() {
	if m.dateTime.SetTimezone == nil || !m.inDateTimeContent() {
		return
	}
	current := m.state.DateTime.Timezone
	dtRef := m.dateTime
	m.dialog = NewDialog("Set Timezone", []DialogFieldSpec{
		{Key: "tz", Label: "Timezone (e.g. America/New_York)", Value: current},
	}, func(result DialogResult) tea.Cmd {
		tz := result["tz"]
		if tz == "" || tz == current {
			return nil
		}
		return dtRef.SetTimezone(tz)
	})
}

func (m *Model) triggerDTNTPToggle() tea.Cmd {
	if m.dateTime.SetNTP == nil || !m.inDateTimeContent() {
		return nil
	}
	return m.dateTime.SetNTP(!m.state.DateTime.NTPEnabled)
}

func (m *Model) triggerDTClockFormatToggle() tea.Cmd {
	if m.dateTime.ToggleClockFormat == nil || !m.inDateTimeContent() {
		return nil
	}
	return m.dateTime.ToggleClockFormat()
}
