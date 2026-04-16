package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
)

// renderSSHTab is F4's top-level renderer. Layout mirrors F5: an
// outer sidebar on the far left, an inner sub-nav column listing
// the six SSH subsections vertically, and a detail pane on the
// right showing whichever subsection is selected. Focus moves
// between the inner sub-nav and the detail pane with enter/esc.
func renderSSHTab(s *core.State, width, height int) string {
	sidebar := renderF4Sidebar(s, height)
	contentWidth := width - lipgloss.Width(sidebar)
	content := renderSSHContent(s, contentWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderF4Sidebar(s *core.State, height int) string {
	sections := core.F4Sections()
	entries := make([]sidebarEntry, len(sections))
	for i, sec := range sections {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	return renderSidebarGeneric(s, entries, 0, height)
}

func renderSSHContent(s *core.State, width, height int) string {
	if !s.SSHLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Loading SSH state…"))
	}
	if !s.SSH.InstalledOK {
		return renderContentPane(width, height, sshMissingPanel())
	}
	inner := renderSSHInnerSidebar(s, height)
	detailWidth := width - lipgloss.Width(inner)
	detail := renderSSHSubsection(s, detailWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, inner, detail)
}

// renderSSHInnerSidebar lays out the six subsection rows as a
// vertical list. When the user is focused on the detail pane
// (ContentFocused=true) the active row shows as "selected but
// unfocused" so the user can see which subsection they came from.
func renderSSHInnerSidebar(s *core.State, height int) string {
	subsections := []struct {
		kind  core.SSHSubsection
		label string
	}{
		{core.SSHSubKeys, "Keys"},
		{core.SSHSubAgent, "Agent"},
		{core.SSHSubClientConfig, "Client"},
		{core.SSHSubKnownHosts, "Known Hosts"},
		{core.SSHSubAuthorizedKeys, "Authorized"},
		{core.SSHSubServerConfig, "Server"},
	}
	itemWidth := s.SidebarItemWidth
	item := sidebarItem.Width(itemWidth)
	active := sidebarItemActive.Width(itemWidth)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	outerFocused := !s.SSHContentFocused
	var rows []string
	for _, sub := range subsections {
		selected := sub.kind == s.SSHSelection.Subsection
		switch {
		case selected && outerFocused:
			rows = append(rows, active.Render(sub.label))
		case selected:
			rows = append(rows, item.Render(dim.Render(sub.label)))
		default:
			rows = append(rows, item.Render(sub.label))
		}
	}
	return renderSidebarPane(height, strings.Join(rows, "\n"), outerFocused)
}

func renderSSHSubsection(s *core.State, width, height int) string {
	switch s.SSHSelection.Subsection {
	case core.SSHSubKeys:
		return renderSSHKeys(s, width, height)
	case core.SSHSubAgent:
		return renderSSHAgent(s, width, height)
	case core.SSHSubClientConfig:
		return renderSSHClientConfig(s, width, height)
	case core.SSHSubKnownHosts:
		return renderSSHKnownHosts(s, width, height)
	case core.SSHSubAuthorizedKeys:
		return renderSSHAuthorizedKeys(s, width, height)
	case core.SSHSubServerConfig:
		return renderSSHServerConfig(s, width, height)
	}
	return renderContentPane(width, height,
		placeholderStyle.Render("Nothing selected."))
}

func sshMissingPanel() string {
	var b strings.Builder
	b.WriteString(contentTitle.Render("SSH"))
	b.WriteString("\n\n")
	b.WriteString(placeholderStyle.Render(
		"openssh-client is not installed on this system.\n\n" +
			"Install it with:\n\n" +
			"  sudo pacman -S openssh\n\n" +
			"Then restart dark to pick up the new tools."))
	return b.String()
}

// ─── Keys ───────────────────────────────────────────────────────

func renderSSHKeys(s *core.State, width, height int) string {
	var b strings.Builder
	b.WriteString(contentTitle.Render("SSH Keys"))
	b.WriteString("\n\n")
	if len(s.SSH.Keys) == 0 {
		b.WriteString(placeholderStyle.Render("No keys found in ~/.ssh.\n\n"))
		b.WriteString(sshActionHint(s, "g generate"))
		return renderContentPane(width, height, b.String())
	}
	for i, k := range s.SSH.Keys {
		selected := s.SSHContentFocused && i == s.SSHKeysIdx
		b.WriteString(sshKeyBlock(k, selected))
		b.WriteString("\n")
	}
	hint := "enter focus · g generate"
	if s.SSHContentFocused {
		hint = "j/k nav · g gen · d del · p passphrase · a agent · c copy · esc back"
	}
	b.WriteString(sshActionHint(s, hint))
	return renderContentPane(width, height, b.String())
}

func sshKeyBlock(k core.SSHKey, selected bool) string {
	var b strings.Builder
	marker := "  "
	if k.InAgent {
		marker = "● "
	}
	nameStyle := lipgloss.NewStyle().Bold(true)
	if selected {
		nameStyle = nameStyle.Foreground(colorAccent)
		marker = "▶ "
	}
	b.WriteString(nameStyle.Render(marker + sshKeyBaseName(k)))
	b.WriteString("\n")
	b.WriteString(sshDetailRow("Type", strings.ToUpper(k.Type)))
	if k.Bits > 0 {
		b.WriteString(sshDetailRow("Bits", fmt.Sprintf("%d", k.Bits)))
	}
	if k.Comment != "" {
		b.WriteString(sshDetailRow("Comment", k.Comment))
	}
	b.WriteString(sshDetailRow("Fingerprint", k.Fingerprint))
	b.WriteString(sshDetailRow("Passphrase", boolLabel(k.HasPassphrase, "yes", "no")))
	b.WriteString(sshDetailRow("In agent", boolLabel(k.InAgent, "yes", "no")))
	return b.String()
}

// ─── Agent ──────────────────────────────────────────────────────

func renderSSHAgent(s *core.State, width, height int) string {
	var b strings.Builder
	b.WriteString(contentTitle.Render("SSH Agent"))
	b.WriteString("\n\n")
	ag := s.SSH.Agent
	b.WriteString(sshDetailRow("Running", boolLabel(ag.Running, "yes", "no")))
	b.WriteString(sshDetailRow("Systemd managed", boolLabel(ag.SystemdManaged, "yes", "no")))
	b.WriteString(sshDetailRow("Unit installed", boolLabel(ag.SystemdUnitExists, "yes", "no")))
	b.WriteString(sshDetailRow("Forwarded", boolLabel(ag.Forwarded, "yes (from remote ssh session)", "no")))
	if ag.SocketPath != "" {
		b.WriteString(sshDetailRow("Socket", ag.SocketPath))
	}
	if ag.Pid > 0 {
		b.WriteString(sshDetailRow("PID", fmt.Sprintf("%d", ag.Pid)))
	}
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Loaded keys"))
	b.WriteString("\n\n")
	if len(ag.LoadedKeys) == 0 {
		b.WriteString(placeholderStyle.Render("  (no keys loaded)\n"))
	}
	for i, lk := range ag.LoadedKeys {
		prefix := "  "
		if s.SSHContentFocused && i == s.SSHAgentIdx {
			prefix = "▶ "
		}
		b.WriteString(fmt.Sprintf("%s%s  %s  %s\n", prefix, lk.Type, lk.Fingerprint, lk.Comment))
	}
	b.WriteString("\n")
	hint := "enter focus · s start · x stop"
	if s.SSHContentFocused {
		hint = "j/k navigate · s start · x stop · d remove · D remove all · esc back"
	}
	b.WriteString(sshActionHint(s, hint))
	return renderContentPane(width, height, b.String())
}

// ─── Client config ──────────────────────────────────────────────

func renderSSHClientConfig(s *core.State, width, height int) string {
	var b strings.Builder
	b.WriteString(contentTitle.Render("Client config"))
	b.WriteString("\n")
	b.WriteString(placeholderStyle.Render("  " + s.SSH.ClientConfig.Path))
	b.WriteString("\n\n")
	hosts := s.SSH.ClientConfig.Hosts
	if len(hosts) == 0 {
		b.WriteString(placeholderStyle.Render("(no Host blocks)\n\n"))
		b.WriteString(sshActionHint(s, "n new host"))
		return renderContentPane(width, height, b.String())
	}
	for i, h := range hosts {
		selected := s.SSHContentFocused && i == s.SSHHostsIdx
		b.WriteString(sshHostBlock(h, selected))
		b.WriteString("\n")
	}
	hint := "enter focus · n new host"
	if s.SSHContentFocused {
		hint = "j/k nav · n new · e edit · d del · R restore .bak · esc back"
	}
	b.WriteString(sshActionHint(s, hint))
	return renderContentPane(width, height, b.String())
}

func sshHostBlock(h core.SSHHostEntry, selected bool) string {
	var b strings.Builder
	nameStyle := lipgloss.NewStyle().Bold(true)
	prefix := ""
	if selected {
		nameStyle = nameStyle.Foreground(colorAccent)
		prefix = "▶ "
	}
	b.WriteString(nameStyle.Render(prefix + h.Pattern))
	b.WriteString("\n")
	if h.HostName != "" {
		b.WriteString(sshDetailRow("HostName", h.HostName))
	}
	if h.User != "" {
		b.WriteString(sshDetailRow("User", h.User))
	}
	if h.Port > 0 {
		b.WriteString(sshDetailRow("Port", fmt.Sprintf("%d", h.Port)))
	}
	if h.IdentityFile != "" {
		b.WriteString(sshDetailRow("IdentityFile", h.IdentityFile))
	}
	if h.ProxyJump != "" {
		b.WriteString(sshDetailRow("ProxyJump", h.ProxyJump))
	}
	if h.ForwardAgent {
		b.WriteString(sshDetailRow("ForwardAgent", "yes"))
	}
	if h.StrictHostKeyChecking != "" {
		b.WriteString(sshDetailRow("StrictHostKeyChecking", h.StrictHostKeyChecking))
	}
	return b.String()
}

// ─── Known hosts ────────────────────────────────────────────────

func renderSSHKnownHosts(s *core.State, width, height int) string {
	var b strings.Builder
	b.WriteString(contentTitle.Render("Known hosts"))
	b.WriteString("\n\n")
	if len(s.SSH.KnownHosts) == 0 {
		b.WriteString(placeholderStyle.Render("~/.ssh/known_hosts is empty.\n\n"))
	}
	for i, kh := range s.SSH.KnownHosts {
		prefix := "  "
		if s.SSHContentFocused && i == s.SSHKnownHostsIdx {
			prefix = "▶ "
		}
		b.WriteString(fmt.Sprintf("%s%-32s  %-12s  %s\n",
			prefix, sshTruncate(kh.Hostname, 32), kh.KeyType, kh.Fingerprint))
	}
	b.WriteString("\n")
	hint := "enter focus · s scan host"
	if s.SSHContentFocused {
		hint = "j/k navigate · s scan · d remove · esc back"
	}
	b.WriteString(sshActionHint(s, hint))
	return renderContentPane(width, height, b.String())
}

// ─── Authorized keys ────────────────────────────────────────────

func renderSSHAuthorizedKeys(s *core.State, width, height int) string {
	var b strings.Builder
	b.WriteString(contentTitle.Render("Authorized keys"))
	b.WriteString("\n")
	b.WriteString(placeholderStyle.Render(
		"  Incoming connection allow-list (~/.ssh/authorized_keys)."))
	b.WriteString("\n\n")
	if len(s.SSH.AuthorizedKeys) == 0 {
		b.WriteString(placeholderStyle.Render("(none)\n"))
	}
	for i, ak := range s.SSH.AuthorizedKeys {
		prefix := "  "
		if s.SSHContentFocused && i == s.SSHAuthorizedIdx {
			prefix = "▶ "
		}
		b.WriteString(fmt.Sprintf("%s%-12s  %s  %s\n", prefix, ak.KeyType, ak.Fingerprint, ak.Comment))
	}
	b.WriteString("\n")
	hint := "enter focus · n add key"
	if s.SSHContentFocused {
		hint = "j/k nav · n add · d remove · R restore .bak · esc back"
	}
	b.WriteString(sshActionHint(s, hint))
	return renderContentPane(width, height, b.String())
}

// ─── Server config ──────────────────────────────────────────────

func renderSSHServerConfig(s *core.State, width, height int) string {
	sc := s.SSH.ServerConfig
	var b strings.Builder
	b.WriteString(contentTitle.Render("Server config"))
	b.WriteString("\n\n")
	b.WriteString(sshDetailRow("Path", sc.Path))
	if !sc.Readable {
		b.WriteString(placeholderStyle.Render("\n" + sc.ParseError + "\n"))
		b.WriteString("\n")
		b.WriteString(sshActionHint(s, "— cannot read sshd_config"))
		return renderContentPane(width, height, b.String())
	}
	if sc.Port > 0 {
		b.WriteString(sshDetailRow("Port", fmt.Sprintf("%d", sc.Port)))
	}
	if sc.PermitRootLogin != "" {
		b.WriteString(sshDetailRow("PermitRootLogin", sc.PermitRootLogin))
	}
	b.WriteString(sshDetailRow("PasswordAuthentication", boolLabel(sc.PasswordAuthentication, "yes", "no")))
	b.WriteString(sshDetailRow("PubkeyAuthentication", boolLabel(sc.PubkeyAuthentication, "yes", "no")))
	if sc.X11Forwarding {
		b.WriteString(sshDetailRow("X11Forwarding", "yes"))
	}
	if len(sc.AllowUsers) > 0 {
		b.WriteString(sshDetailRow("AllowUsers", strings.Join(sc.AllowUsers, " ")))
	}
	if len(sc.AllowGroups) > 0 {
		b.WriteString(sshDetailRow("AllowGroups", strings.Join(sc.AllowGroups, " ")))
	}
	b.WriteString("\n")
	hint := "enter focus · e edit"
	if s.SSHContentFocused {
		hint = "e edit (sshd -t validates) · R restore .bak · esc back"
	}
	b.WriteString(sshActionHint(s, hint))
	return renderContentPane(width, height, b.String())
}

// ─── Helpers ────────────────────────────────────────────────────

func sshKeyBaseName(k core.SSHKey) string {
	if k.Path != "" {
		parts := strings.Split(k.Path, "/")
		return parts[len(parts)-1]
	}
	if k.PublicPath != "" {
		parts := strings.Split(k.PublicPath, "/")
		return strings.TrimSuffix(parts[len(parts)-1], ".pub")
	}
	return "(orphan)"
}

func sshDetailRow(label, value string) string {
	return fmt.Sprintf("  %-22s %s\n",
		lipgloss.NewStyle().Foreground(colorDim).Render(label),
		value)
}

func boolLabel(v bool, yes, no string) string {
	if v {
		return yes
	}
	return no
}

// sshActionHint prepends the subsection's key hint with any pending
// action error so failures stay visible until the next successful
// mutation clears SSHActionError.
func sshActionHint(s *core.State, keys string) string {
	style := lipgloss.NewStyle().Foreground(colorDim).Italic(true)
	if s.SSHActionError != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		return errStyle.Render("error: "+s.SSHActionError) + "\n" + style.Render(keys)
	}
	return style.Render(keys)
}

func sshTruncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max < 4 {
		return s[:max]
	}
	return s[:max-1] + "…"
}
