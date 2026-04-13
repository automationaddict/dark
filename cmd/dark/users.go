package main

import (
	"encoding/json"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/users"
	"github.com/johnnelson/dark/internal/tui"
)

func newUsersActions(nc *nats.Conn) tui.UsersActions {
	return tui.UsersActions{
		AddUser: func(username, fullName, shell string, admin bool) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersAddCmd, map[string]any{
					"username": username, "full_name": fullName, "shell": shell, "admin": admin,
				})
			}
		},
		RemoveUser: func(username string, removeHome bool) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersRemoveCmd, map[string]any{
					"username": username, "remove_home": removeHome,
				})
			}
		},
		SetShell: func(username, shell string) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersShellCmd, map[string]any{
					"username": username, "shell": shell,
				})
			}
		},
		SetFullName: func(username, fullName string) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersCommentCmd, map[string]any{
					"username": username, "full_name": fullName,
				})
			}
		},
		LockUser: func(username string) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersLockCmd, map[string]any{
					"username": username, "admin": false,
				})
			}
		},
		UnlockUser: func(username string) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersLockCmd, map[string]any{
					"username": username, "admin": true,
				})
			}
		},
		AddToGroup: func(username, group string) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersGroupCmd, map[string]any{
					"username": username, "group": group, "admin": true,
				})
			}
		},
		RemoveFromGroup: func(username, group string) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersGroupCmd, map[string]any{
					"username": username, "group": group, "admin": false,
				})
			}
		},
		ToggleAdmin: func(username string, admin bool) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersAdminCmd, map[string]any{
					"username": username, "admin": admin,
				})
			}
		},
		SetPassword: func(username, currentPass, newPass string) tea.Cmd {
			return func() tea.Msg {
				return usersRequest(nc, bus.SubjectUsersPasswdCmd, map[string]any{
					"username": username, "current_pass": currentPass, "password": newPass,
				})
			}
		},
		ElevateShadow: func(targetUser string) tea.Cmd {
			return func() tea.Msg {
				return usersElevateRequest(nc, targetUser)
			}
		},
	}
}

func usersRequest(nc *nats.Conn, subject string, payload any) tui.UsersActionResultMsg {
	data, _ := json.Marshal(payload)
	reply, err := nc.Request(subject, data, core.TimeoutNormal)
	if err != nil {
		return tui.UsersActionResultMsg{Err: err.Error()}
	}
	return parseUsersResponse(reply.Data)
}

func parseUsersResponse(data []byte) tui.UsersActionResultMsg {
	var resp struct {
		Snapshot users.Snapshot `json:"snapshot"`
		Error    string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return tui.UsersActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.UsersActionResultMsg{Err: resp.Error}
	}
	return tui.UsersActionResultMsg{Snapshot: resp.Snapshot}
}

func usersElevateRequest(nc *nats.Conn, targetUser string) tui.UsersElevatedMsg {
	data, _ := json.Marshal(map[string]any{})
	reply, err := nc.Request(bus.SubjectUsersElevateCmd, data, core.TimeoutPair)
	if err != nil {
		return tui.UsersElevatedMsg{Err: err.Error(), Username: targetUser}
	}
	var resp struct {
		Snapshot users.Snapshot `json:"snapshot"`
		Error    string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.UsersElevatedMsg{Err: err.Error(), Username: targetUser}
	}
	if resp.Error != "" {
		return tui.UsersElevatedMsg{Err: resp.Error, Username: targetUser}
	}
	return tui.UsersElevatedMsg{Snapshot: resp.Snapshot, Username: targetUser}
}
