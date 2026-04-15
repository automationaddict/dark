package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/keybind"
	"github.com/automationaddict/dark/internal/tui"
)

func newKeybindActions(nc *nats.Conn) tui.KeybindActions {
	return tui.KeybindActions{
		Add: func(b keybind.Binding) tea.Cmd {
			return func() tea.Msg {
				return keybindMutationRequest(nc, bus.SubjectKeybindAddCmd, map[string]any{
					"mods": b.Mods, "key": b.Key, "desc": b.Desc,
					"dispatcher": b.Dispatcher, "args": b.Args,
					"source": string(b.Source), "category": b.Category,
					"bind_type": b.BindType,
				})
			}
		},
		Update: func(old, new keybind.Binding) tea.Cmd {
			return func() tea.Msg {
				return keybindMutationRequest(nc, bus.SubjectKeybindUpdateCmd, map[string]any{
					"mods": new.Mods, "key": new.Key, "desc": new.Desc,
					"dispatcher": new.Dispatcher, "args": new.Args,
					"source": string(new.Source), "category": new.Category,
					"bind_type": new.BindType,
					"old_mods": old.Mods, "old_key": old.Key, "old_desc": old.Desc,
					"old_dispatcher": old.Dispatcher, "old_args": old.Args,
					"old_source": string(old.Source), "old_category": old.Category,
					"old_bind_type": old.BindType,
				})
			}
		},
		Remove: func(b keybind.Binding) tea.Cmd {
			return func() tea.Msg {
				return keybindMutationRequest(nc, bus.SubjectKeybindRemoveCmd, map[string]any{
					"mods": b.Mods, "key": b.Key, "desc": b.Desc,
					"dispatcher": b.Dispatcher, "args": b.Args,
					"source": string(b.Source), "category": b.Category,
					"bind_type": b.BindType,
				})
			}
		},
	}
}

func keybindMutationRequest(nc *nats.Conn, subject string, payload any) tui.KeybindActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.KeybindActionResultMsg{Err: err.Error()}
	}
	return parseKeybindResponse(reply.Data)
}

func parseKeybindResponse(data []byte) tui.KeybindActionResultMsg {
	var resp struct {
		Snapshot keybind.Snapshot `json:"snapshot"`
		Error    string          `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.KeybindActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.KeybindActionResultMsg{Err: resp.Error}
	}
	return tui.KeybindActionResultMsg{Snapshot: resp.Snapshot}
}
