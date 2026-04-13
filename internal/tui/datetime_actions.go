package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/datetime"
)

type DateTimeActions struct {
	SetTimezone       func(tz string) tea.Cmd
	SetNTP            func(enabled bool) tea.Cmd
	ToggleClockFormat func() tea.Cmd
	SetTime           func(timeStr string) tea.Cmd
	SetLocalRTC       func(local bool) tea.Cmd
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
	zones := m.state.DateTime.Timezones

	if len(zones) > 0 {
		m.dialog = NewDialog("Set Timezone", []DialogFieldSpec{
			{Key: "tz", Label: "Timezone", Kind: DialogFieldSelect, Options: zones, Value: current},
		}, func(result DialogResult) tea.Cmd {
			tz := result["tz"]
			if tz == "" || tz == current {
				return nil
			}
			return dtRef.SetTimezone(tz)
		})
	} else {
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
}

func (m *Model) triggerDTNTPToggle() tea.Cmd {
	if m.dateTime.SetNTP == nil || !m.inDateTimeContent() {
		return nil
	}
	if !m.state.DateTime.CanNTP {
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

func (m *Model) triggerDTSetTimeDialog() {
	if m.dateTime.SetTime == nil || !m.inDateTimeContent() {
		return
	}
	if m.state.DateTime.NTPEnabled {
		return
	}
	now := time.Now().Format("2006-01-02 15:04:05")
	dtRef := m.dateTime
	m.dialog = NewDialog("Set System Time", []DialogFieldSpec{
		{Key: "time", Label: "Date & Time (YYYY-MM-DD HH:MM:SS)", Value: now},
	}, func(result DialogResult) tea.Cmd {
		t := result["time"]
		if t == "" {
			return nil
		}
		return dtRef.SetTime(t)
	})
}

func (m *Model) triggerDTRTCToggle() tea.Cmd {
	if m.dateTime.SetLocalRTC == nil || !m.inDateTimeContent() {
		return nil
	}
	// RTCInUTC == true means LocalRTC is false; toggle it
	return m.dateTime.SetLocalRTC(m.state.DateTime.RTCInUTC)
}
