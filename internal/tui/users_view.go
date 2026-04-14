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

	secs := core.UsersSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.UsersContentFocused
	sidebar := renderInnerSidebarFocused(s, entries, s.UsersSectionIdx, height, sidebarFocused)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActiveUsersSection()
	var content string
	switch sec.ID {
	case "identity":
		content = renderUsersIdentitySection(s, contentWidth, height)
	case "account":
		content = renderUsersAccountSection(s, contentWidth, height)
	case "security":
		content = renderUsersSecuritySection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── Identity section ────────────────────────────────────────────────

func renderUsersIdentitySection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}

	focused := s.UsersContentFocused
	var blocks []string

	blocks = append(blocks, renderUserListBox(s, innerWidth, focused))

	if u, ok := s.SelectedUser(); ok {
		blocks = append(blocks, renderUserIdentity(u, innerWidth))
	}

	blocks = append(blocks, renderUsersIdentityHint(focused))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderUsersIdentityHint(focused bool) string {
	if !focused {
		return statusBarStyle.Render("enter to select")
	}
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("c")+" rename")
	hints = append(hints, accent.Render("s")+" shell")
	hints = append(hints, accent.Render("a")+" add user")
	hints = append(hints, accent.Render("d")+" remove")
	hints = append(hints, accent.Render("esc"))
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Account section ─────────────────────────────────────────────────

func renderUsersAccountSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}

	focused := s.UsersContentFocused
	var blocks []string

	blocks = append(blocks, renderUserListBox(s, innerWidth, focused))

	if u, ok := s.SelectedUser(); ok {
		blocks = append(blocks, renderUserAccount(u, innerWidth))
		blocks = append(blocks, renderUserGroups(u, innerWidth))
	}

	blocks = append(blocks, renderUsersAccountHint(focused))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderUsersAccountHint(focused bool) string {
	if !focused {
		return statusBarStyle.Render("enter to select")
	}
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("w")+" admin toggle")
	hints = append(hints, accent.Render("l")+" lock toggle")
	hints = append(hints, accent.Render("g")+" add group")
	hints = append(hints, accent.Render("G")+" remove group")
	hints = append(hints, accent.Render("esc"))
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Security section ────────────────────────────────────────────────

func renderUsersSecuritySection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}

	focused := s.UsersContentFocused
	var blocks []string

	blocks = append(blocks, renderUserListBox(s, innerWidth, focused))

	if u, ok := s.SelectedUser(); ok {
		blocks = append(blocks, renderUserSecurity(u, innerWidth))
		if sess := renderUserSessions(u, innerWidth); sess != "" {
			blocks = append(blocks, sess)
		}
		if logins := renderUserLogins(u, innerWidth); logins != "" {
			blocks = append(blocks, logins)
		}
	}

	blocks = append(blocks, renderUsersSecurityHint(focused))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderUsersSecurityHint(focused bool) string {
	if !focused {
		return statusBarStyle.Render("enter to select")
	}
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("p")+" password")
	hints = append(hints, accent.Render("esc"))
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Shared rendering helpers ────────────────────────────────────────

func renderUserListBox(s *core.State, total int, focused bool) string {
	type col struct {
		header string
		cell   func(users.User) string
		accent func(users.User) bool
	}
	cols := []col{
		{"Username", func(u users.User) string { return u.Username }, nil},
		{"Name", func(u users.User) string { return orDash(u.FullName) }, nil},
		{"UID", func(u users.User) string { return fmt.Sprintf("%d", u.UID) }, nil},
		{"Admin", func(u users.User) string {
			if u.IsAdmin {
				return "yes"
			}
			return ""
		}, func(u users.User) bool { return u.IsAdmin }},
		{"Locked", func(u users.User) string {
			if u.IsLocked {
				return "󰌾"
			}
			return ""
		}, nil},
	}

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, u := range s.Users.Users {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(u)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = "  "
	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines := []string{"  " + strings.Join(headerCells, gap)}

	for i, u := range s.Users.Users {
		isSel := i == s.UsersIdx
		var marker string
		switch {
		case isSel && focused:
			marker = tableSelectionMarker.Render("▸ ")
		case isSel:
			marker = tableSelectionMarkerDim.Render("▸ ")
		default:
			marker = "  "
		}
		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.cell(u)
			var style lipgloss.Style
			switch {
			case isSel:
				style = tableCellSelected
			case c.accent != nil && c.accent(u):
				style = tableCellAccent
			default:
				style = tableCellStyle
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		lines = append(lines, marker+strings.Join(cells, gap))
	}

	return groupBoxSections("Users", []string{strings.Join(lines, "\n")}, total, borderForFocus(focused))
}

func renderUserIdentity(u users.User, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string
	lines = append(lines, label.Render("Username")+accent.Render(u.Username))
	lines = append(lines, label.Render("Full Name")+value.Render(u.FullName))
	lines = append(lines, label.Render("UID")+value.Render(fmt.Sprintf("%d", u.UID)))
	lines = append(lines, label.Render("GID")+value.Render(fmt.Sprintf("%d (%s)", u.GID, u.Primary)))
	lines = append(lines, label.Render("Home")+value.Render(u.HomeDir))
	lines = append(lines, label.Render("Shell")+value.Render(u.Shell))

	return groupBoxSections("Identity", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderUserAccount(u users.User, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)

	var lines []string

	adminStr := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("yes (wheel)")
	if !u.IsAdmin {
		adminStr = lipgloss.NewStyle().Foreground(colorDim).Render("no")
	}
	lines = append(lines, label.Render("Administrator")+adminStr)

	var lockStr string
	if !u.ShadowAvail {
		lockStr = lipgloss.NewStyle().Foreground(colorDim).Render("requires elevation")
	} else if u.IsLocked {
		lockStr = lipgloss.NewStyle().Foreground(colorRed).Render("locked")
	} else {
		lockStr = lipgloss.NewStyle().Foreground(colorGreen).Render("active")
	}
	lines = append(lines, label.Render("Status")+lockStr)
	lines = append(lines, label.Render("Processes")+
		detailValueStyle.Render(fmt.Sprintf("%d", u.ProcessCount)))

	return groupBoxSections("Account", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderUserGroups(u users.User, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	groupList := strings.Join(u.Groups, ", ")
	if groupList == "" {
		groupList = dim.Render("none")
	}
	lines = append(lines, label.Render("Groups")+value.Render(groupList))
	lines = append(lines, label.Render("Group Count")+value.Render(fmt.Sprintf("%d", len(u.Groups))))

	return groupBoxSections("Groups", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderUserSecurity(u users.User, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	var passStr string
	if !u.ShadowAvail {
		passStr = dim.Render("requires elevation")
	} else if u.HasPasswd {
		passStr = lipgloss.NewStyle().Foreground(colorGreen).Render("set")
	} else {
		passStr = lipgloss.NewStyle().Foreground(colorGold).Render("not set")
	}
	lines = append(lines, label.Render("Password")+passStr)

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
