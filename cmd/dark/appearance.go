package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	appearancesvc "github.com/johnnelson/dark/internal/services/appearance"
	"github.com/johnnelson/dark/internal/tui"
)

func newAppearanceActions(nc *nats.Conn) tui.AppearanceActions {
	return tui.AppearanceActions{
		SetTheme: func(name string) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceThemeCmd, map[string]any{
					"theme": name,
				})
			}
		},
		SetGapsIn: func(val int) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceGapsInCmd, map[string]any{
					"value": val,
				})
			}
		},
		SetGapsOut: func(val int) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceGapsOutCmd, map[string]any{
					"value": val,
				})
			}
		},
		SetBorder: func(val int) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceBorderCmd, map[string]any{
					"value": val,
				})
			}
		},
		SetRounding: func(val int) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceRoundingCmd, map[string]any{
					"value": val,
				})
			}
		},
		SetBlur: func(enabled bool) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceBlurCmd, map[string]any{
					"enabled": enabled,
				})
			}
		},
		SetBlurSize: func(val int) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceBlurSizeCmd, map[string]any{
					"value": val,
				})
			}
		},
		SetBlurPass: func(val int) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceBlurPassCmd, map[string]any{
					"value": val,
				})
			}
		},
		SetAnim: func(enabled bool) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceAnimCmd, map[string]any{
					"enabled": enabled,
				})
			}
		},
		SetFont: func(name string) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceFontCmd, map[string]any{
					"font": name,
				})
			}
		},
		SetFontSize: func(val int) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceFontSizeCmd, map[string]any{
					"value": val,
				})
			}
		},
		SetBackground: func(name string) tea.Cmd {
			return func() tea.Msg {
				return appearanceRequest(nc, bus.SubjectAppearanceBackgroundCmd, map[string]any{
					"background": name,
				})
			}
		},
	}
}

func appearanceRequest(nc *nats.Conn, subject string, payload any) tui.AppearanceActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.AppearanceActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot appearancesvc.Snapshot `json:"snapshot"`
		Error    string                 `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.AppearanceActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.AppearanceActionResultMsg{Err: resp.Error}
	}
	return tui.AppearanceActionResultMsg{Snapshot: resp.Snapshot}
}
