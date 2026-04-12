package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	inputsvc "github.com/johnnelson/dark/internal/services/input"
	"github.com/johnnelson/dark/internal/tui"
)

func newInputActions(nc *nats.Conn) tui.InputActions {
	return tui.InputActions{
		SetRepeatRate: func(rate int) tea.Cmd {
			return func() tea.Msg {
				return inputRequest(nc, bus.SubjectInputRepeatRateCmd, map[string]any{"rate": rate})
			}
		},
		SetRepeatDelay: func(delay int) tea.Cmd {
			return func() tea.Msg {
				return inputRequest(nc, bus.SubjectInputRepeatDelayCmd, map[string]any{"delay": delay})
			}
		},
		SetSensitivity: func(sens float64) tea.Cmd {
			return func() tea.Msg {
				return inputRequest(nc, bus.SubjectInputSensitivityCmd, map[string]any{"sens": sens})
			}
		},
		SetNaturalScroll: func(enabled bool) tea.Cmd {
			return func() tea.Msg {
				return inputRequest(nc, bus.SubjectInputNatScrollCmd, map[string]any{"enabled": enabled})
			}
		},
		SetScrollFactor: func(factor float64) tea.Cmd {
			return func() tea.Msg {
				return inputRequest(nc, bus.SubjectInputScrollFactorCmd, map[string]any{"factor": factor})
			}
		},
		SetKBLayout: func(layout string) tea.Cmd {
			return func() tea.Msg {
				return inputRequest(nc, bus.SubjectInputKBLayoutCmd, map[string]any{"layout": layout})
			}
		},
	}
}

func inputRequest(nc *nats.Conn, subject string, payload any) tui.InputActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.InputActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot inputsvc.Snapshot `json:"snapshot"`
		Error    string           `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.InputActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.InputActionResultMsg{Err: resp.Error}
	}
	return tui.InputActionResultMsg{Snapshot: resp.Snapshot}
}
