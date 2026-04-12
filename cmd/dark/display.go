package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	displaysvc "github.com/johnnelson/dark/internal/services/display"
	"github.com/johnnelson/dark/internal/tui"
)

func newDisplayActions(nc *nats.Conn) tui.DisplayActions {
	return tui.DisplayActions{
		SetResolution: func(name string, width, height int, refreshRate float64) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayResolutionCmd, map[string]any{
					"name": name, "width": width, "height": height, "refresh_rate": refreshRate,
				})
			}
		},
		SetScale: func(name string, scale float64) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayScaleCmd, map[string]any{
					"name": name, "scale": scale,
				})
			}
		},
		SetTransform: func(name string, transform int) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayTransformCmd, map[string]any{
					"name": name, "transform": transform,
				})
			}
		},
		SetPosition: func(name string, x, y int) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayPositionCmd, map[string]any{
					"name": name, "x": x, "y": y,
				})
			}
		},
		SetDpms: func(name string, on bool) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayDpmsCmd, map[string]any{
					"name": name, "on": on,
				})
			}
		},
		SetVrr: func(name string, mode int) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayVrrCmd, map[string]any{
					"name": name, "mode": mode,
				})
			}
		},
		SetMirror: func(name, mirrorOf string) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayMirrorCmd, map[string]any{
					"name": name, "mirror_of": mirrorOf,
				})
			}
		},
		ToggleEnabled: func(name string) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayToggleCmd, map[string]any{
					"name": name,
				})
			}
		},
		Identify: func() tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayIdentifyCmd, map[string]any{})
			}
		},
		SetBrightness: func(pct int) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayBrightnessCmd, map[string]any{
					"pct": pct,
				})
			}
		},
		SetKbdBrightness: func(pct int) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayKbdBrightnessCmd, map[string]any{
					"pct": pct,
				})
			}
		},
		SetNightLight: func(enable bool, tempK int, gamma int) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayNightLightCmd, map[string]any{
					"enable": enable, "temperature": tempK, "gamma": gamma,
				})
			}
		},
		SetGamma: func(pct int) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayGammaCmd, map[string]any{
					"pct": pct,
				})
			}
		},
		SaveProfile: func(name string) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplaySaveProfileCmd, map[string]any{
					"profile": name,
				})
			}
		},
		ApplyProfile: func(name string) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayApplyProfileCmd, map[string]any{
					"profile": name,
				})
			}
		},
		DeleteProfile: func(name string) tea.Cmd {
			return func() tea.Msg {
				return displayRequest(nc, bus.SubjectDisplayDeleteProfileCmd, map[string]any{
					"profile": name,
				})
			}
		},
	}
}

func displayRequest(nc *nats.Conn, subject string, payload map[string]any) tui.DisplayActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.DisplayActionResultMsg{Err: err.Error()}
	}
	return decodeDisplayReply(reply.Data)
}

func decodeDisplayReply(data []byte) tui.DisplayActionResultMsg {
	var resp struct {
		Snapshot displaysvc.Snapshot `json:"snapshot"`
		Error    string              `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.DisplayActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.DisplayActionResultMsg{Err: resp.Error}
	}
	return tui.DisplayActionResultMsg{Snapshot: resp.Snapshot}
}

func requestInitialDisplay(nc *nats.Conn) (displaysvc.Snapshot, bool) {
	reply, err := nc.Request(bus.SubjectDisplayMonitorsCmd, nil, core.TimeoutFast)
	if err != nil {
		return displaysvc.Snapshot{}, false
	}
	var snap displaysvc.Snapshot
	if err := json.Unmarshal(reply.Data, &snap); err != nil {
		return displaysvc.Snapshot{}, false
	}
	return snap, true
}
