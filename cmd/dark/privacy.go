package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/privacy"
	"github.com/johnnelson/dark/internal/tui"
)

func newPrivacyActions(nc *nats.Conn) tui.PrivacyActions {
	return tui.PrivacyActions{
		SetIdleTimeout: func(field string, seconds int) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacyIdleCmd, map[string]any{
					"field": field, "seconds": seconds,
				})
			}
		},
		SetDNSOverTLS: func(value string) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacyDNSTLSCmd, map[string]any{"value": value})
			}
		},
		SetDNSSEC: func(value string) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacyDNSSECCmd, map[string]any{"value": value})
			}
		},
		SetFirewall: func(enable bool) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacyFirewallCmd, map[string]any{"enabled": enable})
			}
		},
		SetSSH: func(enable bool) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacySSHCmd, map[string]any{"enabled": enable})
			}
		},
		ClearRecent: func() tea.Cmd {
			return func() tea.Msg {
				return privacySimpleRequest(nc, bus.SubjectPrivacyClearCmd)
			}
		},
		SetLocation: func(enable bool) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacyLocationCmd, map[string]any{"enabled": enable})
			}
		},
		SetMACRandom: func(value string) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacyMACCmd, map[string]any{"value": value})
			}
		},
		SetIndexer: func(enable bool) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacyIndexerCmd, map[string]any{"enabled": enable})
			}
		},
		SetCoredumpStorage: func(value string) tea.Cmd {
			return func() tea.Msg {
				return privacyRequest(nc, bus.SubjectPrivacyCoredumpCmd, map[string]any{"value": value})
			}
		},
	}
}

func privacyRequest(nc *nats.Conn, subject string, payload any) tui.PrivacyActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.PrivacyActionResultMsg{Err: err.Error()}
	}
	return parsePrivacyResponse(reply.Data)
}

func privacySimpleRequest(nc *nats.Conn, subject string) tui.PrivacyActionResultMsg {
	reply, err := nc.Request(subject, nil, core.TimeoutNormal)
	if err != nil {
		return tui.PrivacyActionResultMsg{Err: err.Error()}
	}
	return parsePrivacyResponse(reply.Data)
}

func parsePrivacyResponse(data []byte) tui.PrivacyActionResultMsg {
	var resp struct {
		Snapshot privacy.Snapshot `json:"snapshot"`
		Error    string          `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.PrivacyActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.PrivacyActionResultMsg{Err: resp.Error}
	}
	return tui.PrivacyActionResultMsg{Snapshot: resp.Snapshot}
}
