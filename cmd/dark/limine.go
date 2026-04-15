package main

import (
	"encoding/json"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/services/limine"
	"github.com/automationaddict/dark/internal/tui"
)

// limineActionTimeout is generous because every write action funnels
// through pkexec, which blocks until the user interacts with the
// polkit prompt. Two minutes is well past a normal password entry.
const limineActionTimeout = 2 * time.Minute

func newLimineActions(nc *nats.Conn) tui.LimineActions {
	return tui.LimineActions{
		Create: func(description string) tea.Cmd {
			return func() tea.Msg {
				return limineRequest(nc, bus.SubjectLimineCreateCmd, map[string]any{
					"description": description,
				})
			}
		},
		Delete: func(number int) tea.Cmd {
			return func() tea.Msg {
				return limineRequest(nc, bus.SubjectLimineDeleteCmd, map[string]any{
					"number": number,
				})
			}
		},
		Sync: func() tea.Cmd {
			return func() tea.Msg {
				return limineRequest(nc, bus.SubjectLimineSyncCmd, nil)
			}
		},
		SetDefaultEntry: func(entry int) tea.Cmd {
			return func() tea.Msg {
				return limineRequest(nc, bus.SubjectLimineDefaultEntryCmd, map[string]any{
					"entry": entry,
				})
			}
		},
		SetBootConfig: func(key, value string) tea.Cmd {
			return func() tea.Msg {
				return limineRequest(nc, bus.SubjectLimineBootConfigCmd, map[string]any{
					"key": key, "value": value,
				})
			}
		},
		SetSyncConfig: func(key, value string) tea.Cmd {
			return func() tea.Msg {
				return limineRequest(nc, bus.SubjectLimineSyncConfigCmd, map[string]any{
					"key": key, "value": value,
				})
			}
		},
		SetOmarchyConfig: func(key, value string) tea.Cmd {
			return func() tea.Msg {
				return limineRequest(nc, bus.SubjectLimineOmarchyConfigCmd, map[string]any{
					"key": key, "value": value,
				})
			}
		},
		SetKernelCmdline: func(lines []string) tea.Cmd {
			return func() tea.Msg {
				return limineRequest(nc, bus.SubjectLimineKernelCmdlineCmd, map[string]any{
					"lines": lines,
				})
			}
		},
	}
}

func limineRequest(nc *nats.Conn, subject string, payload any) tui.LimineActionResultMsg {
	var data []byte
	if payload != nil {
		data, _ = json.Marshal(payload)
	}
	reply, err := nc.Request(subject, data, limineActionTimeout)
	if err != nil {
		return tui.LimineActionResultMsg{Err: err.Error()}
	}
	return parseLimineResponse(reply.Data)
}

func parseLimineResponse(data []byte) tui.LimineActionResultMsg {
	var resp struct {
		Snapshot limine.Snapshot `json:"snapshot"`
		Error    string          `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.LimineActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.LimineActionResultMsg{Err: resp.Error}
	}
	return tui.LimineActionResultMsg{Snapshot: resp.Snapshot}
}
