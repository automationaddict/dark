package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
)

func renderDateTime(s *core.State, width, height int) string {
	if !s.DateTimeLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading date & time…"))
	}

	secs := core.DateTimeSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebar := renderInnerSidebar(s, entries, s.DateTimeSectionIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActiveDateTimeSection()
	var content string
	switch sec.ID {
	case "time":
		content = renderDTTimeSection(s, contentWidth, height)
	case "sync":
		content = renderDTSyncSection(s, contentWidth, height)
	case "hardware":
		content = renderDTHardwareSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── Time section ────────────────────────────────────────────────────

func renderDTTimeSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderDTCurrent(s, innerWidth))
	blocks = append(blocks, renderDTTimezone(s, innerWidth))
	blocks = append(blocks, renderDTTimeHint(s))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDTTimeHint(s *core.State) string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("z")+" timezone")
	if !s.DateTime.NTPEnabled {
		hints = append(hints, accent.Render("t")+" set time")
	}
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Sync section ────────────────────────────────────────────────────

func renderDTSyncSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderDTNTP(s, innerWidth))
	blocks = append(blocks, renderDTFormat(s, innerWidth))
	blocks = append(blocks, renderDTSyncHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDTSyncHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	var hints []string
	hints = append(hints, accent.Render("n")+" NTP toggle")
	hints = append(hints, accent.Render("f")+" clock format")
	return dim.Render("  " + strings.Join(hints, "  "))
}

// ── Hardware section ────────────────────────────────────────────────

func renderDTHardwareSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string
	blocks = append(blocks, renderDTHardware(s, innerWidth))
	blocks = append(blocks, renderDTHardwareHint())

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDTHardwareHint() string {
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)
	return dim.Render("  " + accent.Render("r") + " toggle UTC/Local")
}

// ── Shared rendering helpers ────────────────────────────────────────

func renderDTCurrent(s *core.State, total int) string {
	dt := s.DateTime
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	lines = append(lines, label.Render("Local Time")+value.Render(dt.LocalTime))
	lines = append(lines, label.Render("UTC Time")+value.Render(dt.UTCTime))
	if dt.Uptime != "" {
		lines = append(lines, label.Render("Uptime")+value.Render(dt.Uptime))
	}

	return groupBoxSections("Current Time", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderDTTimezone(s *core.State, total int) string {
	dt := s.DateTime
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string
	lines = append(lines, label.Render("Timezone")+accent.Render(dt.Timezone))
	lines = append(lines, label.Render("Abbreviation")+value.Render(dt.TZAbbrev))
	lines = append(lines, label.Render("UTC Offset")+value.Render(dt.UTCOffset))

	return groupBoxSections("Timezone", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderDTNTP(s *core.State, total int) string {
	dt := s.DateTime
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	ntpStatus := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("enabled")
	if !dt.NTPEnabled {
		ntpStatus = lipgloss.NewStyle().Foreground(colorRed).Render("disabled")
	}
	lines = append(lines, label.Render("NTP Service")+ntpStatus)

	syncStatus := lipgloss.NewStyle().Foreground(colorGreen).Render("synchronized")
	if !dt.NTPSynced {
		syncStatus = lipgloss.NewStyle().Foreground(colorGold).Render("not synchronized")
	}
	lines = append(lines, label.Render("Sync Status")+syncStatus)

	if dt.NTPServer != "" {
		lines = append(lines, label.Render("Server")+value.Render(dt.NTPServer))
	}
	if dt.PollInterval != "" {
		lines = append(lines, label.Render("Poll Interval")+value.Render(dt.PollInterval))
	}
	if dt.Jitter != "" {
		lines = append(lines, label.Render("Jitter")+value.Render(dt.Jitter))
	}

	return groupBoxSections("NTP Synchronization", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderDTFormat(s *core.State, total int) string {
	dt := s.DateTime
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string
	lines = append(lines, label.Render("Clock Format")+accent.Render(dt.ClockFormat))

	if dt.Locale != "" {
		lines = append(lines, label.Render("Locale")+value.Render(dt.Locale))
	}

	return groupBoxSections("Format & Locale", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderDTHardware(s *core.State, total int) string {
	dt := s.DateTime
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	if dt.RTCDate != "" && dt.RTCTime != "" {
		lines = append(lines, label.Render("RTC Time")+value.Render(dt.RTCDate+" "+dt.RTCTime))
	}

	rtcMode := "UTC"
	if !dt.RTCInUTC {
		rtcMode = "Local"
	}
	lines = append(lines, label.Render("RTC Mode")+value.Render(rtcMode))

	if len(lines) == 0 {
		lines = append(lines, placeholderStyle.Render("No hardware clock information available."))
	}
	return groupBoxSections("Hardware Clock", []string{strings.Join(lines, "\n")}, total, colorBorder)
}
