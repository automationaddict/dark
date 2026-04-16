package tui

import (
	"fmt"
	"path/filepath"
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

// renderSSHInnerSidebar renders the six subsection rows using the
// shared inner sidebar component. Focus follows the standard
// pattern: the sidebar border is accented when it owns focus and
// dimmed when the detail pane is active.
func renderSSHInnerSidebar(s *core.State, height int) string {
	secs := core.SSHSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.SSHContentFocused
	return renderInnerSidebarFocused(s, entries, int(s.SSHSelection.Subsection), height, sidebarFocused)
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
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	focused := s.SSHContentFocused

	if len(s.SSH.Keys) == 0 {
		emptyBox := groupBoxSections("Keys",
			[]string{placeholderStyle.Render("No keys found in ~/.ssh.")},
			innerWidth, borderForFocus(focused))
		hint := sshActionHint(s, "g generate")
		body := lipgloss.JoinVertical(lipgloss.Left, emptyBox, "", hint)
		return renderContentPane(width, height, body)
	}

	var keySections []string
	for i, k := range s.SSH.Keys {
		selected := focused && i == s.SSHKeysIdx
		keySections = append(keySections, sshKeyBlock(k, selected))
	}
	keysBox := groupBoxSections("Keys", keySections, innerWidth, borderForFocus(focused))

	blocks := []string{keysBox}

	if len(s.SSH.Certificates) > 0 {
		var certSections []string
		for _, c := range s.SSH.Certificates {
			certSections = append(certSections, sshCertBlock(c))
		}
		certsBox := groupBoxSections("Certificates", certSections, innerWidth, colorBorder)
		blocks = append(blocks, "", certsBox)
	}

	hint := "enter focus · g generate"
	if focused {
		hint = "j/k nav · g gen · d del · p passphrase · a agent · c copy · esc back"
	}
	blocks = append(blocks, "", sshActionHint(s, hint))
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
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
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	focused := s.SSHContentFocused
	ag := s.SSH.Agent

	// Status details box (always dimmed border — informational).
	var statusRows strings.Builder
	statusRows.WriteString(sshDetailRow("Running", sshStatusBool(ag.Running, "yes", "no")))
	statusRows.WriteString(sshDetailRow("Systemd managed", sshStatusBool(ag.SystemdManaged, "yes", "no")))
	statusRows.WriteString(sshDetailRow("Unit installed", sshStatusBool(ag.SystemdUnitExists, "yes", "no")))
	statusRows.WriteString(sshDetailRow("Forwarded", sshStatusBool(ag.Forwarded, "yes (from remote)", "no")))
	if ag.SocketPath != "" {
		statusRows.WriteString(sshDetailRow("Socket", ag.SocketPath))
	}
	if ag.Pid > 0 {
		statusRows.WriteString(sshDetailRow("PID", fmt.Sprintf("%d", ag.Pid)))
	}
	statusBox := groupBoxSections("Status", []string{statusRows.String()}, innerWidth, colorBorder)

	// Loaded keys box (accent border when focused).
	var keysContent string
	if len(ag.LoadedKeys) == 0 {
		keysContent = placeholderStyle.Render("(no keys loaded)")
	} else {
		typeW := 12
		commentW := 16
		fpW := innerWidth - typeW - commentW - 8
		if fpW < 20 {
			fpW = 20
		}
		selectedCell := lipgloss.NewStyle().Foreground(colorBg).Background(colorAccent)
		var data [][]string
		for _, lk := range ag.LoadedKeys {
			data = append(data, []string{lk.Type, lk.Fingerprint, lk.Comment})
		}
		keysContent = renderTable(
			[]string{"Type", "Fingerprint", "Comment"},
			[]int{typeW, fpW, commentW},
			data,
			s.SSHAgentIdx, focused, selectedCell,
		)
	}
	keysBox := groupBoxSections("Loaded Keys", []string{keysContent}, innerWidth, borderForFocus(focused))

	hint := "enter focus · s start · x stop"
	if focused {
		hint = "j/k navigate · s start · x stop · d remove · D remove all · esc back"
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		statusBox, "", keysBox, "", sshActionHint(s, hint))
	return renderContentPane(width, height, body)
}

// ─── Client config ──────────────────────────────────────────────

func renderSSHClientConfig(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	focused := s.SSHContentFocused
	hosts := s.SSH.ClientConfig.Hosts

	var hostsContent string
	if len(hosts) == 0 {
		hostsContent = placeholderStyle.Render("(no Host blocks)")
	} else {
		var hostSections []string
		for i, h := range hosts {
			selected := focused && i == s.SSHHostsIdx
			hostSections = append(hostSections, sshHostBlock(h, selected))
		}
		hostsContent = strings.Join(hostSections, "\n")
	}

	title := "Hosts"
	if s.SSH.ClientConfig.Path != "" {
		title = "Hosts · " + s.SSH.ClientConfig.Path
	}
	hostsBox := groupBoxSections(title, []string{hostsContent}, innerWidth, borderForFocus(focused))

	hint := "enter focus · n new host"
	if focused {
		hint = "j/k nav · n new · e edit · d del · R restore .bak · esc back"
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		hostsBox, "", sshActionHint(s, hint))
	return renderContentPane(width, height, body)
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
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	focused := s.SSHContentFocused

	var content string
	if len(s.SSH.KnownHosts) == 0 {
		content = placeholderStyle.Render("~/.ssh/known_hosts is empty.")
	} else {
		hostW := 30
		typeW := 12
		fpW := innerWidth - hostW - typeW - 8
		if fpW < 16 {
			fpW = 16
		}
		selectedCell := lipgloss.NewStyle().Foreground(colorBg).Background(colorAccent)
		var data [][]string
		for _, kh := range s.SSH.KnownHosts {
			data = append(data, []string{kh.Hostname, kh.KeyType, kh.Fingerprint})
		}
		content = renderTable(
			[]string{"Host", "Type", "Fingerprint"},
			[]int{hostW, typeW, fpW},
			data,
			s.SSHKnownHostsIdx, focused, selectedCell,
		)
	}
	box := groupBoxSections("Known Hosts", []string{content}, innerWidth, borderForFocus(focused))

	hint := "enter focus · s scan host"
	if focused {
		hint = "j/k navigate · s scan · d remove · esc back"
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		box, "", sshActionHint(s, hint))
	return renderContentPane(width, height, body)
}

// ─── Authorized keys ────────────────────────────────────────────

func renderSSHAuthorizedKeys(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	focused := s.SSHContentFocused

	var content string
	if len(s.SSH.AuthorizedKeys) == 0 {
		content = placeholderStyle.Render("(none)")
	} else {
		typeW := 12
		commentW := 16
		fpW := innerWidth - typeW - commentW - 8
		if fpW < 20 {
			fpW = 20
		}
		selectedCell := lipgloss.NewStyle().Foreground(colorBg).Background(colorAccent)
		var data [][]string
		for _, ak := range s.SSH.AuthorizedKeys {
			data = append(data, []string{ak.KeyType, ak.Fingerprint, ak.Comment})
		}
		content = renderTable(
			[]string{"Type", "Fingerprint", "Comment"},
			[]int{typeW, fpW, commentW},
			data,
			s.SSHAuthorizedIdx, focused, selectedCell,
		)
	}
	box := groupBoxSections("Authorized Keys", []string{content}, innerWidth, borderForFocus(focused))

	hint := "enter focus · n add key"
	if focused {
		hint = "j/k nav · n add · d remove · R restore .bak · esc back"
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		box, "", sshActionHint(s, hint))
	return renderContentPane(width, height, body)
}

// ─── Server config ──────────────────────────────────────────────

func renderSSHServerConfig(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	focused := s.SSHContentFocused
	sc := s.SSH.ServerConfig

	if !sc.Readable {
		content := sshDetailRow("Path", sc.Path) + "\n" +
			placeholderStyle.Render(sc.ParseError)
		box := groupBoxSections("Server Config", []string{content}, innerWidth, colorBorder)
		body := lipgloss.JoinVertical(lipgloss.Left,
			box, "", sshActionHint(s, "— cannot read sshd_config"))
		return renderContentPane(width, height, body)
	}

	var rows strings.Builder
	rows.WriteString(sshDetailRow("Path", sc.Path))
	if sc.Port > 0 {
		rows.WriteString(sshDetailRow("Port", fmt.Sprintf("%d", sc.Port)))
	}
	if sc.PermitRootLogin != "" {
		rows.WriteString(sshDetailRow("PermitRootLogin", sc.PermitRootLogin))
	}
	rows.WriteString(sshDetailRow("PasswordAuthentication", boolLabel(sc.PasswordAuthentication, "yes", "no")))
	rows.WriteString(sshDetailRow("PubkeyAuthentication", boolLabel(sc.PubkeyAuthentication, "yes", "no")))
	if sc.X11Forwarding {
		rows.WriteString(sshDetailRow("X11Forwarding", "yes"))
	}
	if len(sc.AllowUsers) > 0 {
		rows.WriteString(sshDetailRow("AllowUsers", strings.Join(sc.AllowUsers, " ")))
	}
	if len(sc.AllowGroups) > 0 {
		rows.WriteString(sshDetailRow("AllowGroups", strings.Join(sc.AllowGroups, " ")))
	}
	box := groupBoxSections("Server Config", []string{rows.String()}, innerWidth, borderForFocus(focused))

	hint := "enter focus · e edit"
	if focused {
		hint = "e edit (sshd -t validates) · R restore .bak · esc back"
	}

	body := lipgloss.JoinVertical(lipgloss.Left,
		box, "", sshActionHint(s, hint))
	return renderContentPane(width, height, body)
}

// ─── Helpers ────────────────────────────────────────────────────

func sshCertBlock(c core.SSHCertificate) string {
	var b strings.Builder
	label := c.KeyID
	if label == "" {
		label = filepath.Base(c.CertPath)
	}
	header := lipgloss.NewStyle().Bold(true)
	if c.Expired {
		header = header.Foreground(lipgloss.Color("9"))
		label += " (EXPIRED)"
	}
	b.WriteString(header.Render("  " + label))
	b.WriteString("\n")
	b.WriteString(sshDetailRow("Type", c.Type+" certificate"))
	b.WriteString(sshDetailRow("Key", c.KeyFingerprint))
	b.WriteString(sshDetailRow("CA", c.CAFingerprint))
	if len(c.Principals) > 0 {
		b.WriteString(sshDetailRow("Principals", strings.Join(c.Principals, ", ")))
	}
	if !c.ValidAfter.IsZero() {
		b.WriteString(sshDetailRow("Valid from", c.ValidAfter.Format("2006-01-02 15:04")))
	}
	if !c.ValidBefore.IsZero() {
		b.WriteString(sshDetailRow("Valid until", c.ValidBefore.Format("2006-01-02 15:04")))
	} else {
		b.WriteString(sshDetailRow("Valid until", "forever"))
	}
	return b.String()
}

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

// sshStatusBool renders a boolean as a styled status indicator matching
// the rest of the app: green bold "✓ yes" / red bold "no".
func sshStatusBool(v bool, yes, no string) string {
	if v {
		return statusOnlineStyle.Render("✓ " + yes)
	}
	return statusOfflineStyle.Render(no)
}

func sshActionHint(_ *core.State, keys string) string {
	return lipgloss.NewStyle().Foreground(colorDim).Italic(true).Render(keys)
}

