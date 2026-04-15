package tui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/users"
)

type UsersActions struct {
	AddUser         func(username, fullName, shell string, admin bool) tea.Cmd
	RemoveUser      func(username string, removeHome bool) tea.Cmd
	SetShell        func(username, shell string) tea.Cmd
	SetFullName     func(username, fullName string) tea.Cmd
	LockUser        func(username string) tea.Cmd
	UnlockUser      func(username string) tea.Cmd
	AddToGroup      func(username, group string) tea.Cmd
	RemoveFromGroup func(username, group string) tea.Cmd
	ToggleAdmin     func(username string, admin bool) tea.Cmd
	SetPassword     func(username, currentPass, newPass string) tea.Cmd
	ElevateShadow   func(targetUser string) tea.Cmd
}

type UsersMsg users.Snapshot

type UsersActionResultMsg struct {
	Snapshot users.Snapshot
	Err      string
}

// UsersElevatedMsg is sent after a successful pkexec elevation.
// The model handles it by opening the password dialog for the target user.
type UsersElevatedMsg struct {
	Snapshot users.Snapshot
	Username string
	Err      string
}

func (m *Model) inUsersContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "users"
}

func (m *Model) inUsersDetails() bool {
	return m.inUsersContent() && m.state.UsersContentFocused
}

func (m *Model) triggerUserAdd() {
	if m.users.AddUser == nil || !m.inUsersDetails() {
		return
	}
	shells := m.state.Users.Shells
	usersRef := m.users
	m.dialog = NewDialog("Add user", []DialogFieldSpec{
		{Key: "username", Label: "Username"},
		{Key: "fullname", Label: "Full name"},
		{Key: "shell", Label: "Shell", Kind: DialogFieldSelect, Options: shells, Value: "/bin/bash"},
		{Key: "admin", Label: "Admin (yes/no)", Value: "no"},
	}, func(result DialogResult) tea.Cmd {
		username := strings.TrimSpace(result["username"])
		if username == "" {
			return nil
		}
		fullName := strings.TrimSpace(result["fullname"])
		shell := result["shell"]
		admin := strings.ToLower(strings.TrimSpace(result["admin"])) == "yes"
		return usersRef.AddUser(username, fullName, shell, admin)
	})
}

func (m *Model) triggerUserRemove() {
	if m.users.RemoveUser == nil || !m.inUsersDetails() {
		return
	}
	u, ok := m.state.SelectedUser()
	if !ok || u.UID == 0 {
		return
	}
	usersRef := m.users
	username := u.Username
	m.dialog = NewDialog("Remove "+username+"?", []DialogFieldSpec{
		{Key: "home", Label: "Remove home directory? (yes/no)", Value: "no"},
	}, func(result DialogResult) tea.Cmd {
		removeHome := strings.ToLower(strings.TrimSpace(result["home"])) == "yes"
		return usersRef.RemoveUser(username, removeHome)
	})
}

func (m *Model) triggerUserShellChange() {
	if m.users.SetShell == nil || !m.inUsersDetails() {
		return
	}
	u, ok := m.state.SelectedUser()
	if !ok {
		return
	}
	shells := m.state.Users.Shells
	usersRef := m.users
	username := u.Username
	m.dialog = NewDialog("Change shell for "+username, []DialogFieldSpec{
		{Key: "shell", Label: "Shell", Kind: DialogFieldSelect, Options: shells, Value: u.Shell},
	}, func(result DialogResult) tea.Cmd {
		shell := result["shell"]
		if shell == "" || shell == u.Shell {
			return nil
		}
		return usersRef.SetShell(username, shell)
	})
}

func (m *Model) triggerUserRename() {
	if m.users.SetFullName == nil || !m.inUsersDetails() {
		return
	}
	u, ok := m.state.SelectedUser()
	if !ok {
		return
	}
	usersRef := m.users
	username := u.Username
	m.dialog = NewDialog("Change name for "+username, []DialogFieldSpec{
		{Key: "name", Label: "Full name", Value: u.FullName},
	}, func(result DialogResult) tea.Cmd {
		name := strings.TrimSpace(result["name"])
		return usersRef.SetFullName(username, name)
	})
}

func (m *Model) triggerUserLockToggle() tea.Cmd {
	if !m.inUsersDetails() {
		return nil
	}
	u, ok := m.state.SelectedUser()
	if !ok || u.UID == 0 {
		return nil
	}
	if u.IsLocked {
		if m.users.UnlockUser == nil {
			return nil
		}
		return m.users.UnlockUser(u.Username)
	}
	if m.users.LockUser == nil {
		return nil
	}
	return m.users.LockUser(u.Username)
}

func (m *Model) triggerUserAdminToggle() tea.Cmd {
	if m.users.ToggleAdmin == nil || !m.inUsersDetails() {
		return nil
	}
	u, ok := m.state.SelectedUser()
	if !ok || u.UID == 0 {
		return nil
	}
	return m.users.ToggleAdmin(u.Username, !u.IsAdmin)
}

func (m *Model) triggerUserGroupAdd() {
	if m.users.AddToGroup == nil || !m.inUsersDetails() {
		return
	}
	u, ok := m.state.SelectedUser()
	if !ok {
		return
	}
	usersRef := m.users
	username := u.Username
	m.dialog = NewDialog("Add "+username+" to group", []DialogFieldSpec{
		{Key: "group", Label: "Group name"},
	}, func(result DialogResult) tea.Cmd {
		group := strings.TrimSpace(result["group"])
		if group == "" {
			return nil
		}
		return usersRef.AddToGroup(username, group)
	})
}

func (m *Model) triggerUserGroupRemove() {
	if m.users.RemoveFromGroup == nil || !m.inUsersDetails() {
		return
	}
	u, ok := m.state.SelectedUser()
	if !ok || len(u.Groups) == 0 {
		return
	}
	usersRef := m.users
	username := u.Username
	m.dialog = NewDialog("Remove "+username+" from group", []DialogFieldSpec{
		{Key: "group", Label: "Group", Kind: DialogFieldSelect, Options: u.Groups},
	}, func(result DialogResult) tea.Cmd {
		group := result["group"]
		if group == "" {
			return nil
		}
		return usersRef.RemoveFromGroup(username, group)
	})
}

func (m *Model) triggerUserPasswordChange() tea.Cmd {
	if m.users.SetPassword == nil || !m.inUsersDetails() {
		return nil
	}
	u, ok := m.state.SelectedUser()
	if !ok {
		return nil
	}

	// Current user: show dialog with current password + new password.
	if u.UID == os.Getuid() {
		m.showCurrentUserPasswordDialog(u.Username)
		return nil
	}

	// Other users: elevate first, then show dialog on success.
	if m.users.ElevateShadow == nil {
		return nil
	}
	return m.users.ElevateShadow(u.Username)
}

func (m *Model) showCurrentUserPasswordDialog(username string) {
	usersRef := m.users
	m.dialog = NewDialog("Change password for "+username, []DialogFieldSpec{
		{Key: "current", Label: "Current password", Kind: DialogFieldPassword},
		{Key: "password", Label: "New password", Kind: DialogFieldPassword},
		{Key: "confirm", Label: "Confirm password", Kind: DialogFieldPassword},
	}, func(result DialogResult) tea.Cmd {
		current := result["current"]
		pass := result["password"]
		confirm := result["confirm"]
		if current == "" || pass == "" || pass != confirm {
			return nil
		}
		return usersRef.SetPassword(username, current, pass)
	})
}

func (m *Model) showOtherUserPasswordDialog(username string) {
	usersRef := m.users
	m.dialog = NewDialog("Set password for "+username, []DialogFieldSpec{
		{Key: "password", Label: "New password", Kind: DialogFieldPassword},
		{Key: "confirm", Label: "Confirm password", Kind: DialogFieldPassword},
	}, func(result DialogResult) tea.Cmd {
		pass := result["password"]
		confirm := result["confirm"]
		if pass == "" || pass != confirm {
			return nil
		}
		return usersRef.SetPassword(username, "", pass)
	})
}

func (m *Model) handleUsersElevated(msg UsersElevatedMsg) {
	if msg.Err != "" {
		m.notifyError("Users", msg.Err)
		return
	}
	m.state.SetUsers(msg.Snapshot)
	m.showOtherUserPasswordDialog(msg.Username)
}
