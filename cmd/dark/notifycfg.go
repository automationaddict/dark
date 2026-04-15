package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/notifycfg"
	"github.com/automationaddict/dark/internal/tui"
)

func newNotifyCfgActions(nc *nats.Conn) tui.NotifyConfigActions {
	return tui.NotifyConfigActions{
		ToggleDND: func() tea.Cmd {
			return func() tea.Msg {
				return notifyCfgSimpleRequest(nc, bus.SubjectNotifyDNDCmd)
			}
		},
		DismissAll: func() tea.Cmd {
			return func() tea.Msg {
				return notifyCfgSimpleRequest(nc, bus.SubjectNotifyDismissCmd)
			}
		},
		SetAnchor: func(anchor string) tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifyAnchorCmd, map[string]any{"anchor": anchor})
			}
		},
		SetTimeout: func(ms int) tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifyTimeoutCmd, map[string]any{"timeout": ms})
			}
		},
		SetWidth: func(px int) tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifyWidthCmd, map[string]any{"timeout": px})
			}
		},
		SetLayer: func(layer string) tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifyLayerCmd, map[string]any{"anchor": layer})
			}
		},
		SetSound: func(soundPath string) tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifySoundCmd, map[string]any{"criteria": soundPath})
			}
		},
		AddAppRule: func(appName string, hide bool) tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifyAddRuleCmd, map[string]any{"app_name": appName, "hide": hide})
			}
		},
		RemoveRule: func(criteria string) tea.Cmd {
			return func() tea.Msg {
				return notifyCfgRequest(nc, bus.SubjectNotifyRemoveRuleCmd, map[string]any{"criteria": criteria})
			}
		},
	}
}

func notifyCfgSimpleRequest(nc *nats.Conn, subject string) tui.NotifyCfgActionResultMsg {
	reply, err := nc.Request(subject, nil, core.TimeoutNormal)
	if err != nil {
		return tui.NotifyCfgActionResultMsg{Err: err.Error()}
	}
	return parseNotifyCfgResponse(reply.Data)
}

func notifyCfgRequest(nc *nats.Conn, subject string, payload any) tui.NotifyCfgActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.NotifyCfgActionResultMsg{Err: err.Error()}
	}
	return parseNotifyCfgResponse(reply.Data)
}

func parseNotifyCfgResponse(data []byte) tui.NotifyCfgActionResultMsg {
	var resp struct {
		Snapshot notifycfg.Snapshot `json:"snapshot"`
		Error    string            `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.NotifyCfgActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.NotifyCfgActionResultMsg{Err: resp.Error}
	}
	return tui.NotifyCfgActionResultMsg{Snapshot: resp.Snapshot}
}
