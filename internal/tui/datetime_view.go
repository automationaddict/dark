package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/datetime"
)

func renderDateTime(s *core.State, width, height int) string {
	if !s.DateTimeLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading date & time…"))
	}

	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	blocks = append(blocks, renderDTCurrent(s.DateTime, innerWidth))
	blocks = append(blocks, renderDTTimezone(s, innerWidth))
	blocks = append(blocks, renderDTNTP(s.DateTime, innerWidth))
	blocks = append(blocks, renderDTFormat(s, innerWidth))
	blocks = append(blocks, renderDTHardware(s.DateTime, innerWidth))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderScrollableContentPane(s, width, height, body)
}

func renderDTCurrent(dt datetime.Snapshot, total int) string {
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
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string
	tzLine := label.Render("Timezone") + accent.Render(dt.Timezone)
	if s.ContentFocused {
		tzLine += dim.Render("  (") + accent.Render("z") + dim.Render(" change)")
	}
	lines = append(lines, tzLine)
	lines = append(lines, label.Render("Abbreviation")+value.Render(dt.TZAbbrev))
	lines = append(lines, label.Render("UTC Offset")+value.Render(dt.UTCOffset))

	return groupBoxSections("Timezone", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderDTNTP(dt datetime.Snapshot, total int) string {
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
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	formatLine := label.Render("Clock Format") + accent.Render(dt.ClockFormat)
	if s.ContentFocused {
		formatLine += dim.Render("  (") + accent.Render("f") + dim.Render(" toggle 12h/24h)")
	}
	lines = append(lines, formatLine)

	if dt.Locale != "" {
		lines = append(lines, label.Render("Locale")+value.Render(dt.Locale))
	}

	return groupBoxSections("Format & Locale", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderDTHardware(dt datetime.Snapshot, total int) string {
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
		return ""
	}
	return groupBoxSections("Hardware Clock", []string{strings.Join(lines, "\n")}, total, colorBorder)
}
