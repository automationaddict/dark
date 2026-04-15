package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	powersvc "github.com/automationaddict/dark/internal/services/power"
	"github.com/automationaddict/dark/internal/tui"
)

func newPowerActions(nc *nats.Conn) tui.PowerActions {
	return tui.PowerActions{
		SetProfile: func(profile string) tea.Cmd {
			return func() tea.Msg {
				return powerRequest(nc, bus.SubjectPowerProfileCmd, map[string]string{
					"profile": profile,
				})
			}
		},
		SetGovernor: func(gov string) tea.Cmd {
			return func() tea.Msg {
				return powerRequest(nc, bus.SubjectPowerGovernorCmd, map[string]string{
					"governor": gov,
				})
			}
		},
		SetEPP: func(epp string) tea.Cmd {
			return func() tea.Msg {
				return powerRequest(nc, bus.SubjectPowerEPPCmd, map[string]string{
					"epp": epp,
				})
			}
		},
		SetButton: func(key, value string) tea.Cmd {
			return func() tea.Msg {
				return powerRequest(nc, bus.SubjectPowerButtonCmd, map[string]string{
					"button_key": key,
					"button_val": value,
				})
			}
		},
		SetIdle: func(kind string, sec int) tea.Cmd {
			return func() tea.Msg {
				return powerRequest(nc, bus.SubjectPowerIdleCmd, map[string]any{
					"idle_kind": kind,
					"idle_sec":  sec,
				})
			}
		},
		SetIdleRunning: func(running bool) tea.Cmd {
			return func() tea.Msg {
				return powerRequest(nc, bus.SubjectPowerIdleRunningCmd, map[string]any{
					"idle_running": running,
				})
			}
		},
	}
}

func powerRequest(nc *nats.Conn, subject string, payload any) tui.PowerActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.PowerActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot powersvc.Snapshot `json:"snapshot"`
		Error    string           `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.PowerActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.PowerActionResultMsg{Err: resp.Error}
	}
	return tui.PowerActionResultMsg{Snapshot: resp.Snapshot}
}
