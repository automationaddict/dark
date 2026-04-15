package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
)

func renderPrivacy(s *core.State, width, height int) string {
	if !s.PrivacyLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading privacy settings…"))
	}

	secs := core.PrivacySections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebar := renderInnerSidebar(s, entries, s.PrivacySectionIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActivePrivacySection()
	var content string
	switch sec.ID {
	case "screenlock":
		content = renderPrivacyScreenLockSection(s, contentWidth, height)
	case "network":
		content = renderPrivacyNetworkSection(s, contentWidth, height)
	case "data":
		content = renderPrivacyDataSection(s, contentWidth, height)
	case "location":
		content = renderPrivacyLocationSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── Screen Lock section ─────────────────────────────────────────────

func renderPrivacyScreenLockSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderPrivacyIdle(s, innerWidth))
	blocks = append(blocks, renderPrivacyScreenLockHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderPrivacyScreenLockHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("1")+" screensaver")
	hints = append(hints, accent.Render("2")+" lock screen")
	hints = append(hints, accent.Render("3")+" screen off")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Network section ─────────────────────────────────────────────────

func renderPrivacyNetworkSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderPrivacyDNS(s, innerWidth))
	blocks = append(blocks, renderPrivacyFirewall(s, innerWidth))
	blocks = append(blocks, renderPrivacySSH(s, innerWidth))
	blocks = append(blocks, renderPrivacyMAC(s, innerWidth))
	blocks = append(blocks, renderPrivacyNetworkHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderPrivacyNetworkHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("t")+" DNS-over-TLS")
	hints = append(hints, accent.Render("e")+" DNSSEC")
	hints = append(hints, accent.Render("f")+" firewall")
	hints = append(hints, accent.Render("s")+" SSH")
	hints = append(hints, accent.Render("m")+" MAC random")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Data section ────────────────────────────────────────────────────

func renderPrivacyDataSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderPrivacyFiles(s, innerWidth))
	blocks = append(blocks, renderPrivacyIndexer(s, innerWidth))
	blocks = append(blocks, renderPrivacyCoredump(s, innerWidth))
	blocks = append(blocks, renderPrivacyDataHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderPrivacyDataHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("x")+" clear recent")
	hints = append(hints, accent.Render("i")+" indexer toggle")
	hints = append(hints, accent.Render("o")+" coredump cycle")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Location section ────────────────────────────────────────────────

func renderPrivacyLocationSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderPrivacyLocation(s, innerWidth))
	blocks = append(blocks, renderPrivacyLocationHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderPrivacyLocationHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	return dim.Render("  " + accent.Render("l") + " toggle location")
}

// ── Shared rendering helpers ────────────────────────────────────────

func renderPrivacyIdle(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	lines = append(lines, label.Render("Screensaver")+value.Render(formatIdleDuration(p.ScreensaverTimeout)))
	lines = append(lines, label.Render("Lock Screen")+value.Render(formatIdleDuration(p.LockTimeout)))
	lines = append(lines, label.Render("Screen Off")+value.Render(formatIdleDuration(p.ScreenOffTimeout)))

	sleepStr := lipgloss.NewStyle().Foreground(colorGreen).Render("yes")
	if !p.LockOnSleep {
		sleepStr = lipgloss.NewStyle().Foreground(colorDim).Render("no")
	}
	lines = append(lines, label.Render("Lock on Sleep")+sleepStr)

	return groupBoxSections("Screen Lock & Idle", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyDNS(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	if p.DNSServer != "" {
		lines = append(lines, label.Render("DNS Server")+value.Render(p.DNSServer))
	}

	tlsColor := colorRed
	if p.DNSOverTLS == "yes" {
		tlsColor = colorGreen
	} else if p.DNSOverTLS == "opportunistic" {
		tlsColor = colorGold
	}
	lines = append(lines, label.Render("DNS-over-TLS")+
		lipgloss.NewStyle().Foreground(tlsColor).Render(p.DNSOverTLS))

	secColor := colorRed
	if p.DNSSEC == "yes" {
		secColor = colorGreen
	} else if p.DNSSEC == "allow-downgrade" {
		secColor = colorGold
	}
	lines = append(lines, label.Render("DNSSEC")+
		lipgloss.NewStyle().Foreground(secColor).Render(p.DNSSEC))

	if p.DNSProtocols != "" {
		lines = append(lines, label.Render("Protocols")+value.Render(p.DNSProtocols))
	}
	if p.FallbackDNS != "" {
		lines = append(lines, label.Render("Fallback")+value.Render(p.FallbackDNS))
	}

	return groupBoxSections("DNS Privacy", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyFirewall(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	if !p.FirewallInstalled {
		lines = append(lines, label.Render("Status")+dim.Render("ufw not installed"))
	} else {
		status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("active")
		if !p.FirewallActive {
			status = lipgloss.NewStyle().Foreground(colorRed).Render("inactive")
		}
		lines = append(lines, label.Render("Status")+status)

		if p.FirewallActive && len(p.FirewallRules) > 0 {
			lines = append(lines, label.Render("Rules")+
				detailValueStyle.Render(fmt.Sprintf("%d", len(p.FirewallRules))))
			for _, rule := range p.FirewallRules {
				lines = append(lines, "  "+dim.Render(rule))
			}
		}
	}

	return groupBoxSections("Firewall", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacySSH(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	if !p.SSHInstalled {
		lines = append(lines, label.Render("Status")+dim.Render("openssh not installed"))
	} else {
		status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("running")
		if !p.SSHActive {
			status = lipgloss.NewStyle().Foreground(colorDim).Render("stopped")
		}
		lines = append(lines, label.Render("SSH Server")+status)

		enabledStr := "enabled"
		if !p.SSHEnabled {
			enabledStr = "disabled"
		}
		lines = append(lines, label.Render("On Boot")+detailValueStyle.Render(enabledStr))
	}

	return groupBoxSections("SSH Server", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyFiles(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)

	var lines []string
	countStr := fmt.Sprintf("%d entries", p.RecentFileCount)
	lines = append(lines, label.Render("Recent Files")+detailValueStyle.Render(countStr))

	return groupBoxSections("File History", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyLocation(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	if !p.LocationInstalled {
		lines = append(lines, label.Render("Status")+dim.Render("geoclue not installed"))
	} else {
		status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("active")
		if !p.LocationActive {
			status = lipgloss.NewStyle().Foreground(colorDim).Render("disabled")
		}
		lines = append(lines, label.Render("Location")+status)
	}

	return groupBoxSections("Location Services", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyMAC(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)

	var lines []string

	macColor := colorRed
	switch p.MACRandomization {
	case "once":
		macColor = colorGold
	case "network":
		macColor = colorGreen
	}
	lines = append(lines, label.Render("MAC Randomization")+
		lipgloss.NewStyle().Foreground(macColor).Render(p.MACRandomization))

	return groupBoxSections("WiFi Privacy", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyIndexer(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string

	if !p.IndexerInstalled {
		lines = append(lines, label.Render("Status")+dim.Render("not installed"))
	} else {
		status := lipgloss.NewStyle().Foreground(colorGreen).Render("active")
		if !p.IndexerActive {
			status = lipgloss.NewStyle().Foreground(colorDim).Render("stopped")
		}
		lines = append(lines, label.Render("File Indexer")+status)
	}

	return groupBoxSections("File Indexing", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyCoredump(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	storageColor := colorGold
	if p.CoredumpStorage == "none" {
		storageColor = colorGreen
	} else if p.CoredumpStorage == "external" {
		storageColor = colorRed
	}
	lines = append(lines, label.Render("Crash Dumps")+
		lipgloss.NewStyle().Foreground(storageColor).Render(p.CoredumpStorage))

	if p.JournalSize != "" {
		lines = append(lines, label.Render("Journal Size")+value.Render(p.JournalSize))
	}

	return groupBoxSections("Diagnostics", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func formatIdleDuration(seconds int) string {
	if seconds == 0 {
		return "disabled"
	}
	m := seconds / 60
	s := seconds % 60
	if m > 0 && s > 0 {
		return fmt.Sprintf("%dm %ds (%ds)", m, s, seconds)
	}
	if m > 0 {
		return fmt.Sprintf("%dm (%ds)", m, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}
