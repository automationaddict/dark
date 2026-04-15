package main

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	workspacessvc "github.com/automationaddict/dark/internal/services/workspaces"
	"github.com/automationaddict/dark/internal/tui"
)

// workspacesActionTimeout is short because every mutation is a
// fast hyprctl invocation. No pkexec, no long-running work.
const workspacesActionTimeout = 5 * time.Second

func newWorkspacesActions(nc *nats.Conn) tui.WorkspacesActions {
	return tui.WorkspacesActions{
		Switch: func(id string) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesSwitchCmd, map[string]any{
					"id": id,
				})
			}
		},
		Rename: func(id int, name string) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesRenameCmd, map[string]any{
					"id":   id,
					"name": name,
				})
			}
		},
		MoveToMonitor: func(id int, monitor string) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesMoveToMonitorCmd, map[string]any{
					"id":      id,
					"monitor": monitor,
				})
			}
		},
		SetLayout: func(id int, layout string) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesSetLayoutCmd, map[string]any{
					"id":     id,
					"layout": layout,
				})
			}
		},
		SetDefaultLayout: func(layout string) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesSetDefaultLayoutCmd, map[string]any{
					"layout": layout,
				})
			}
		},
		SetDwindleOption: func(key, value string) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesSetDwindleOptionCmd, map[string]any{
					"key":   key,
					"value": value,
				})
			}
		},
		SetMasterOption: func(key, value string) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesSetMasterOptionCmd, map[string]any{
					"key":   key,
					"value": value,
				})
			}
		},
		SetCursorWarp: func(enabled bool) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesSetCursorWarpCmd, map[string]any{
					"enabled": enabled,
				})
			}
		},
		SetAnimations: func(enabled bool) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesSetAnimationsCmd, map[string]any{
					"enabled": enabled,
				})
			}
		},
		SetHideSpecial: func(enabled bool) tea.Cmd {
			return func() tea.Msg {
				return workspacesRequest(nc, bus.SubjectWorkspacesSetHideSpecialCmd, map[string]any{
					"enabled": enabled,
				})
			}
		},
	}
}

func workspacesRequest(nc *nats.Conn, subject string, payload any) tui.WorkspacesActionResultMsg {
	var data []byte
	if payload != nil {
		data, _ = json.Marshal(payload)
	}
	reply, err := nc.Request(subject, data, workspacesActionTimeout)
	if err != nil {
		return tui.WorkspacesActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot workspacessvc.Snapshot `json:"snapshot"`
		Error    string                 `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.WorkspacesActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.WorkspacesActionResultMsg{Snapshot: resp.Snapshot, Err: resp.Error}
	}
	return tui.WorkspacesActionResultMsg{Snapshot: resp.Snapshot}
}
