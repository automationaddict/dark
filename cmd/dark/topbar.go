package main

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/services/topbar"
	"github.com/automationaddict/dark/internal/tui"
)

// topBarTimeout is generous because restart actions cycle waybar
// (kill + wait + spawn) and the daemon sleeps 150ms between the
// mutation and the snapshot read.
const topBarTimeout = 10 * time.Second

func newTopBarActions(nc *nats.Conn) tui.TopBarActions {
	return tui.TopBarActions{
		SetRunning: func(running bool) tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarSetRunningCmd, map[string]any{
					"running": running,
				})
			}
		},
		Restart: func() tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarRestartCmd, nil)
			}
		},
		Reset: func() tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarResetCmd, nil)
			}
		},
		SetPosition: func(value string) tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarSetPositionCmd, map[string]any{
					"value": value,
				})
			}
		},
		SetLayer: func(value string) tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarSetLayerCmd, map[string]any{
					"value": value,
				})
			}
		},
		SetHeight: func(value int) tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarSetHeightCmd, map[string]any{
					"value": value,
				})
			}
		},
		SetSpacing: func(value int) tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarSetSpacingCmd, map[string]any{
					"value": value,
				})
			}
		},
		SetConfig: func(content string) tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarSetConfigCmd, map[string]any{
					"content": content,
				})
			}
		},
		SetStyle: func(content string) tea.Cmd {
			return func() tea.Msg {
				return topBarRequest(nc, bus.SubjectTopBarSetStyleCmd, map[string]any{
					"content": content,
				})
			}
		},
	}
}

func topBarRequest(nc *nats.Conn, subject string, payload any) tui.TopBarActionResultMsg {
	var data []byte
	if payload != nil {
		data, _ = json.Marshal(payload)
	}
	reply, err := nc.Request(subject, data, topBarTimeout)
	if err != nil {
		return tui.TopBarActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot topbar.Snapshot `json:"snapshot"`
		Error    string          `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.TopBarActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.TopBarActionResultMsg{Snapshot: resp.Snapshot, Err: resp.Error}
	}
	return tui.TopBarActionResultMsg{Snapshot: resp.Snapshot}
}
