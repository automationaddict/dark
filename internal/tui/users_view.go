package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/users"
)

func renderUsers(s *core.State, width, height int) string {
	if !s.UsersLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading users…"))
	}
	if len(s.Users.Users) == 0 {
		return renderContentPane(width, height,
			placeholderStyle.Render("No users found."))
	}

	// User list sidebar (left) + detail pane (right).
	listWidth := 22
	detailWidth := width - listWidth - 2
	if detailWidth < 40 {
		detailWidth = 40
	}

	list := renderUserList(s, listWidth, height)
	detail := renderUserDetail(s, detailWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, list, detail)
}

func renderUserList(s *core.State, width, height int) string {
	active := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent).
		Width(width - 2).
		Padding(0, 1)
	normal := lipgloss.NewStyle().
		Foreground(colorText).
		Width(width - 2).
		Padding(0, 1)
	dimStyle := lipgloss.NewStyle().Foreground(colorDim)

	var rows []string
	for i, u := range s.Users.Users {
		label := u.Username
		if u.UID == 0 {
			label += dimStyle.Render(" (root)")
		} else if u.IsAdmin {
			label += dimStyle.Render(" *")
		}
		if u.IsLocked {
			label += dimStyle.Render(" 󰌾")
		}
		if i == s.UsersIdx && s.ContentFocused {
			rows = append(rows, active.Render(label))
		} else {
			rows = append(rows, normal.Render(label))
		}
	}

	title := lipgloss.NewStyle().
		Foreground(colorAccent).
		Bold(true).
		Padding(0, 1).
		Render("Users")

	body := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{title, ""}, rows...)...)

	container := lipgloss.NewStyle().
		Width(width).
		Height(height).
		MaxHeight(height)
	return container.Render(body)
}

func renderUserDetail(s *core.State, width, height int) string {
	u, ok := s.SelectedUser()
	if !ok {
		return renderContentPane(width, height,
			placeholderStyle.Render("Select a user."))
	}

	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}

	var blocks []string

	blocks = append(blocks, renderUserIdentity(s, u, innerWidth))
	blocks = append(blocks, renderUserAccount(s, u, innerWidth))
	blocks = append(blocks, renderUserGroups(s, u, innerWidth))
	blocks = append(blocks, renderUserSecurity(s, u, innerWidth))
	if sess := renderUserSessions(u, innerWidth); sess != "" {
		blocks = append(blocks, sess)
	}
	if logins := renderUserLogins(u, innerWidth); logins != "" {
		blocks = append(blocks, logins)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderScrollableContentPane(s, width, height, body)
}

func renderUserIdentity(s *core.State, u users.User, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	nameLine := label.Render("Username") + accent.Render(u.Username)
	lines = append(lines, nameLine)

	fullLine := label.Render("Full Name") + value.Render(u.FullName)
	if s.ContentFocused && u.UID != 0 {
		fullLine += dim.Render("  (") + accent.Render("c") + dim.Render(" change)")
	}
	lines = append(lines, fullLine)

	lines = append(lines, label.Render("UID")+value.Render(fmt.Sprintf("%d", u.UID)))
	lines = append(lines, label.Render("GID")+value.Render(fmt.Sprintf("%d (%s)", u.GID, u.Primary)))
	lines = append(lines, label.Render("Home")+value.Render(u.HomeDir))

	shellLine := label.Render("Shell") + value.Render(u.Shell)
	if s.ContentFocused && u.UID != 0 {
		shellLine += dim.Render("  (") + accent.Render("s") + dim.Render(" change)")
	}
	lines = append(lines, shellLine)

	return groupBoxSections("Identity", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderUserAccount(s *core.State, u users.User, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	// Admin status.
	adminStr := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("yes (wheel)")
	if !u.IsAdmin {
		adminStr = lipgloss.NewStyle().Foreground(colorDim).Render("no")
	}
	adminLine := label.Render("Administrator") + adminStr
	if s.ContentFocused && u.UID != 0 {
		adminLine += dim.Render("  (") + accent.Render("w") + dim.Render(" toggle)")
	}
	lines = append(lines, adminLine)

	// Lock status.
	var lockStr string
	if !u.ShadowAvail {
		lockStr = dim.Render("requires elevation")
	} else if u.IsLocked {
		lockStr = lipgloss.NewStyle().Foreground(colorRed).Render("locked")
	} else {
		lockStr = lipgloss.NewStyle().Foreground(colorGreen).Render("active")
	}
	lockLine := label.Render("Status") + lockStr
	if s.ContentFocused && u.UID != 0 {
		lockLine += dim.Render("  (") + accent.Render("l") + dim.Render(" toggle)")
	}
	lines = append(lines, lockLine)

	// Process count.
	lines = append(lines, label.Render("Processes")+
		detailValueStyle.Render(fmt.Sprintf("%d", u.ProcessCount)))

	return groupBoxSections("Account", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderUserGroups(s *core.State, u users.User, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	groupList := strings.Join(u.Groups, ", ")
	if groupList == "" {
		groupList = dim.Render("none")
	}
	lines = append(lines, label.Render("Groups")+value.Render(groupList))
	lines = append(lines, label.Render("Group Count")+value.Render(fmt.Sprintf("%d", len(u.Groups))))

	if s.ContentFocused && u.UID != 0 {
		hintLine := dim.Render("  (") + accent.Render("g") + dim.Render(" add · ") +
			accent.Render("G") + dim.Render(" remove)")
		lines = append(lines, hintLine)
	}

	return groupBoxSections("Groups", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderUserSecurity(s *core.State, u users.User, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	var passStr string
	if !u.ShadowAvail {
		passStr = dim.Render("requires elevation")
	} else if u.HasPasswd {
		passStr = lipgloss.NewStyle().Foreground(colorGreen).Render("set")
	} else {
		passStr = lipgloss.NewStyle().Foreground(colorGold).Render("not set")
	}
	passLine := label.Render("Password") + passStr
	if s.ContentFocused && u.UID != 0 {
		passLine += dim.Render("  (") + accent.Render("p") + dim.Render(" change)")
	}
	lines = append(lines, passLine)

	if u.LastChanged != "" {
		lines = append(lines, label.Render("Last Changed")+value.Render(u.LastChanged))
	}
	if u.PasswordAge != "" {
		lines = append(lines, label.Render("Password Age")+value.Render(u.PasswordAge))
	}
	if u.MaxDays > 0 {
		lines = append(lines, label.Render("Max Age")+value.Render(fmt.Sprintf("%d days", u.MaxDays)))
	}
	if u.MinDays > 0 {
		lines = append(lines, label.Render("Min Age")+value.Render(fmt.Sprintf("%d days", u.MinDays)))
	}
	if u.WarnDays > 0 {
		lines = append(lines, label.Render("Warn Before")+value.Render(fmt.Sprintf("%d days", u.WarnDays)))
	}
	if u.InactiveDays > 0 {
		lines = append(lines, label.Render("Inactive After")+value.Render(fmt.Sprintf("%d days", u.InactiveDays)))
	}
	if u.Expires != "" {
		lines = append(lines, label.Render("Expires")+value.Render(u.Expires))
	}

	return groupBoxSections("Security", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderUserSessions(u users.User, total int) string {
	if len(u.Sessions) == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	for _, sess := range u.Sessions {
		stateStr := lipgloss.NewStyle().Foreground(colorGreen).Render(sess.State)
		lines = append(lines, label.Render("Session "+sess.ID)+stateStr)
		if sess.TTY != "" {
			lines = append(lines, label.Render("  TTY")+value.Render(sess.TTY))
		}
		if sess.Remote != "" && sess.Remote != "no" {
			lines = append(lines, label.Render("  Remote")+value.Render(sess.Remote))
		}
		if sess.Since != "" {
			lines = append(lines, label.Render("  Since")+value.Render(sess.Since))
		}
	}

	return groupBoxSections("Active Sessions", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderUserLogins(u users.User, total int) string {
	if len(u.LastLogins) == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	for i, login := range u.LastLogins {
		timeStr := login.Time
		if login.Duration != "" {
			timeStr += "  (" + login.Duration + ")"
		}
		lines = append(lines, label.Render(fmt.Sprintf("Login %d", i+1))+value.Render(timeStr))
		if login.TTY != "" {
			lines = append(lines, label.Render("  TTY")+value.Render(login.TTY))
		}
		if login.Host != "" {
			lines = append(lines, label.Render("  From")+value.Render(login.Host))
		}
	}

	return groupBoxSections("Recent Logins", []string{strings.Join(lines, "\n")}, total, colorBorder)
}
