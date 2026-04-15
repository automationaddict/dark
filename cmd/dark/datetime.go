package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/datetime"
	"github.com/automationaddict/dark/internal/tui"
)

func newDateTimeActions(nc *nats.Conn) tui.DateTimeActions {
	return tui.DateTimeActions{
		SetTimezone: func(tz string) tea.Cmd {
			return func() tea.Msg {
				return dateTimeRequest(nc, bus.SubjectDateTimeTZCmd, map[string]any{"timezone": tz})
			}
		},
		SetNTP: func(enabled bool) tea.Cmd {
			return func() tea.Msg {
				return dateTimeRequest(nc, bus.SubjectDateTimeNTPCmd, map[string]any{"enabled": enabled})
			}
		},
		ToggleClockFormat: func() tea.Cmd {
			return func() tea.Msg {
				return dateTimeSimpleRequest(nc, bus.SubjectDateTimeFormatCmd)
			}
		},
		SetTime: func(timeStr string) tea.Cmd {
			return func() tea.Msg {
				return dateTimeRequest(nc, bus.SubjectDateTimeSetTimeCmd, map[string]any{"time": timeStr})
			}
		},
		SetLocalRTC: func(local bool) tea.Cmd {
			return func() tea.Msg {
				return dateTimeRequest(nc, bus.SubjectDateTimeRTCCmd, map[string]any{"local": local})
			}
		},
	}
}

func dateTimeRequest(nc *nats.Conn, subject string, payload any) tui.DateTimeActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.DateTimeActionResultMsg{Err: err.Error()}
	}
	return parseDateTimeResponse(reply.Data)
}

func dateTimeSimpleRequest(nc *nats.Conn, subject string) tui.DateTimeActionResultMsg {
	reply, err := nc.Request(subject, nil, core.TimeoutNormal)
	if err != nil {
		return tui.DateTimeActionResultMsg{Err: err.Error()}
	}
	return parseDateTimeResponse(reply.Data)
}

func parseDateTimeResponse(data []byte) tui.DateTimeActionResultMsg {
	var resp struct {
		Snapshot datetime.Snapshot `json:"snapshot"`
		Error    string           `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.DateTimeActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.DateTimeActionResultMsg{Err: resp.Error}
	}
	return tui.DateTimeActionResultMsg{Snapshot: resp.Snapshot}
}
