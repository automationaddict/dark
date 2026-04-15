package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/sysinfo"
)

func renderAbout(s *core.State, width, height int) string {
	if !s.SysInfoLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading system info…"))
	}

	secs := core.AboutSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebar := renderInnerSidebar(s, entries, s.AboutSectionIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActiveAboutSection()
	var content string
	switch sec.ID {
	case "system":
		content = renderAboutSystemSection(s, contentWidth, height)
	case "hardware":
		content = renderAboutHardwareSection(s, contentWidth, height)
	case "session":
		content = renderAboutSessionSection(s, contentWidth, height)
	case "omarchy":
		content = renderAboutOmarchySection(s, contentWidth, height)
	case "dark":
		content = renderAboutDarkSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── System section ──────────────────────────────────────────────────

func renderAboutSystemSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	info := s.SysInfo
	fields := [][2]string{
		{"Host", info.Hostname},
		{"OS", info.OSPretty},
		{"Kernel", info.Kernel},
		{"Arch", info.Arch},
		{"Uptime", sysinfo.FormatDuration(info.Uptime)},
	}
	body := renderAboutBox("System", fields, innerWidth)
	return renderContentPane(width, height, body)
}

// ── Hardware section ────────────────────────────────────────────────

func renderAboutHardwareSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	info := s.SysInfo
	fields := [][2]string{
		{"CPU", info.CPUModel},
		{"Cores", fmt.Sprintf("%d", info.CPUCores)},
		{"Memory", formatMem(info.MemUsed, info.MemTotal)},
		{"Swap", formatMem(info.SwapUsed, info.SwapTotal)},
	}
	body := renderAboutBox("Hardware", fields, innerWidth)
	return renderContentPane(width, height, body)
}

// ── Session section ─────────────────────────────────────────────────

func renderAboutSessionSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	info := s.SysInfo
	fields := [][2]string{
		{"User", info.User},
		{"Shell", info.Shell},
		{"Terminal", info.Terminal},
		{"Desktop", info.Desktop},
	}
	body := renderAboutBox("Session", fields, innerWidth)
	return renderContentPane(width, height, body)
}

// ── Omarchy section ─────────────────────────────────────────────────

func renderAboutOmarchySection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	info := s.SysInfo

	softwareFields := [][2]string{
		{"Version", info.OmarchyVersion},
		{"Branch", info.OmarchyBranch},
		{"Channel", info.OmarchyChannel},
		{"Kernel", info.Kernel},
		{"Compositor", info.Compositor},
		{"Terminal", info.Terminal},
		{"Packages", fmt.Sprintf("%d (pacman)", info.PackageCount)},
		{"Theme", info.OmarchyTheme},
		{"Font", info.Font},
	}
	softwareBox := renderAboutBox("Software", softwareFields, innerWidth)

	ageFields := [][2]string{
		{"OS Age", sysinfo.FormatDuration(info.InstallAge)},
		{"Uptime", sysinfo.FormatDuration(info.Uptime)},
	}
	if !info.LastUpdate.IsZero() {
		ageFields = append(ageFields, [2]string{"Last Update", info.LastUpdate.Format("Mon, Jan 2 2006 at 15:04")})
	}
	ageBox := renderAboutBox("Age / Uptime / Update", ageFields, innerWidth)

	// Update status
	var updateLine string
	if s.UpdateLoaded {
		if s.Update.UpdateAvailable {
			updateLine = statusOnlineStyle.Render("Update available: " + s.Update.AvailableVersion)
		} else {
			updateLine = placeholderStyle.Render("System is up to date")
		}
	}

	aboutFields := [][2]string{
		{"Author", info.OmarchyAuthor},
		{"Website", "omarchy.org"},
		{"License", "MIT"},
	}
	aboutBox := renderAboutBox("Omarchy", aboutFields, innerWidth)

	blocks := []string{softwareBox, "", ageBox}
	if updateLine != "" {
		blocks = append(blocks, "", updateLine)
	}
	blocks = append(blocks, "", aboutBox)

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

// ── dark section ────────────────────────────────────────────────────

func renderAboutDarkSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}
	info := s.SysInfo
	fields := [][2]string{
		{"Version", info.DarkVersion},
		{"Go", info.GoVersion},
		{"Binary", info.BinaryPath},
		{"Built", info.BinaryMTime.Format("2006-01-02 15:04:05")},
	}
	body := renderAboutBox("dark", fields, innerWidth)
	return renderContentPane(width, height, body)
}

// ── Shared helpers ──────────────────────────────────────────────────

func renderAboutBox(title string, fields [][2]string, total int) string {
	lw := 14
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	for _, f := range fields {
		lines = append(lines, label.Render(f[0])+value.Render(displayValue(f[1])))
	}
	return groupBoxSections(title, []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func displayValue(v string) string {
	if strings.TrimSpace(v) == "" {
		return placeholderStyle.Render("—")
	}
	return v
}

func formatMem(used, total uint64) string {
	if total == 0 {
		return "—"
	}
	pct := float64(used) / float64(total) * 100
	return fmt.Sprintf("%s / %s  (%.0f%%)", sysinfo.FormatBytes(used), sysinfo.FormatBytes(total), pct)
}
