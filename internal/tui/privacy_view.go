package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

func renderPrivacy(s *core.State, width, height int) string {
	if !s.PrivacyLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading privacy settings…"))
	}

	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderPrivacyIdle(s, innerWidth))
	blocks = append(blocks, renderPrivacyDNS(s, innerWidth))
	blocks = append(blocks, renderPrivacyFirewall(s, innerWidth))
	blocks = append(blocks, renderPrivacySSH(s, innerWidth))
	blocks = append(blocks, renderPrivacyLocation(s, innerWidth))
	blocks = append(blocks, renderPrivacyMAC(s, innerWidth))
	blocks = append(blocks, renderPrivacyFiles(s, innerWidth))
	blocks = append(blocks, renderPrivacyIndexer(s, innerWidth))
	blocks = append(blocks, renderPrivacyCoredump(s, innerWidth))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderScrollableContentPane(s, width, height, body)
}

func renderPrivacyIdle(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	ssLine := label.Render("Screensaver") + value.Render(formatIdleDuration(p.ScreensaverTimeout))
	if s.ContentFocused {
		ssLine += dim.Render("  (") + accent.Render("1") + dim.Render(" set)")
	}
	lines = append(lines, ssLine)

	lockLine := label.Render("Lock Screen") + value.Render(formatIdleDuration(p.LockTimeout))
	if s.ContentFocused {
		lockLine += dim.Render("  (") + accent.Render("2") + dim.Render(" set)")
	}
	lines = append(lines, lockLine)

	offLine := label.Render("Screen Off") + value.Render(formatIdleDuration(p.ScreenOffTimeout))
	if s.ContentFocused {
		offLine += dim.Render("  (") + accent.Render("3") + dim.Render(" set)")
	}
	lines = append(lines, offLine)

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
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

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
	tlsLine := label.Render("DNS-over-TLS") +
		lipgloss.NewStyle().Foreground(tlsColor).Render(p.DNSOverTLS)
	if s.ContentFocused {
		tlsLine += dim.Render("  (") + accent.Render("t") + dim.Render(" cycle)")
	}
	lines = append(lines, tlsLine)

	secColor := colorRed
	if p.DNSSEC == "yes" {
		secColor = colorGreen
	} else if p.DNSSEC == "allow-downgrade" {
		secColor = colorGold
	}
	secLine := label.Render("DNSSEC") +
		lipgloss.NewStyle().Foreground(secColor).Render(p.DNSSEC)
	if s.ContentFocused {
		secLine += dim.Render("  (") + accent.Render("e") + dim.Render(" cycle)")
	}
	lines = append(lines, secLine)

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
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	if !p.FirewallInstalled {
		lines = append(lines, label.Render("Status")+dim.Render("ufw not installed"))
	} else {
		status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("active")
		if !p.FirewallActive {
			status = lipgloss.NewStyle().Foreground(colorRed).Render("inactive")
		}
		fwLine := label.Render("Status") + status
		if s.ContentFocused {
			fwLine += dim.Render("  (") + accent.Render("f") + dim.Render(" toggle)")
		}
		lines = append(lines, fwLine)

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
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	if !p.SSHInstalled {
		lines = append(lines, label.Render("Status")+dim.Render("openssh not installed"))
	} else {
		status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("running")
		if !p.SSHActive {
			status = lipgloss.NewStyle().Foreground(colorDim).Render("stopped")
		}
		sshLine := label.Render("SSH Server") + status
		if s.ContentFocused {
			sshLine += dim.Render("  (") + accent.Render("s") + dim.Render(" toggle)")
		}
		lines = append(lines, sshLine)

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
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	countStr := fmt.Sprintf("%d entries", p.RecentFileCount)
	fileLine := label.Render("Recent Files") + detailValueStyle.Render(countStr)
	if s.ContentFocused && p.RecentFileCount > 0 {
		fileLine += dim.Render("  (") + accent.Render("x") + dim.Render(" clear)")
	}
	lines = append(lines, fileLine)

	return groupBoxSections("File History", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyLocation(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	if !p.LocationInstalled {
		lines = append(lines, label.Render("Status")+dim.Render("geoclue not installed"))
	} else {
		status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("active")
		if !p.LocationActive {
			status = lipgloss.NewStyle().Foreground(colorDim).Render("disabled")
		}
		locLine := label.Render("Location") + status
		if s.ContentFocused {
			locLine += dim.Render("  (") + accent.Render("l") + dim.Render(" toggle)")
		}
		lines = append(lines, locLine)
	}

	return groupBoxSections("Location Services", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyMAC(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	macColor := colorRed
	switch p.MACRandomization {
	case "once":
		macColor = colorGold
	case "network":
		macColor = colorGreen
	}
	macLine := label.Render("MAC Randomization") +
		lipgloss.NewStyle().Foreground(macColor).Render(p.MACRandomization)
	if s.ContentFocused {
		macLine += dim.Render("  (") + accent.Render("m") + dim.Render(" cycle)")
	}
	lines = append(lines, macLine)

	return groupBoxSections("WiFi Privacy", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyIndexer(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	if !p.IndexerInstalled {
		lines = append(lines, label.Render("Status")+dim.Render("not installed"))
	} else {
		status := lipgloss.NewStyle().Foreground(colorGreen).Render("active")
		if !p.IndexerActive {
			status = lipgloss.NewStyle().Foreground(colorDim).Render("stopped")
		}
		idxLine := label.Render("File Indexer") + status
		if s.ContentFocused {
			idxLine += dim.Render("  (") + accent.Render("i") + dim.Render(" toggle)")
		}
		lines = append(lines, idxLine)
	}

	return groupBoxSections("File Indexing", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderPrivacyCoredump(s *core.State, total int) string {
	p := s.Privacy
	lw := 20
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	storageColor := colorGold
	if p.CoredumpStorage == "none" {
		storageColor = colorGreen
	} else if p.CoredumpStorage == "external" {
		storageColor = colorRed
	}
	cdLine := label.Render("Crash Dumps") +
		lipgloss.NewStyle().Foreground(storageColor).Render(p.CoredumpStorage)
	if s.ContentFocused {
		cdLine += dim.Render("  (") + accent.Render("o") + dim.Render(" cycle)")
	}
	lines = append(lines, cdLine)

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
