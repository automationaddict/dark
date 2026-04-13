package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/notifycfg"
)

func renderNotifications(s *core.State, width, height int) string {
	if !s.NotifyLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading notification settings…"))
	}

	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	var blocks []string

	blocks = append(blocks, renderNotifyDaemon(s, innerWidth))
	blocks = append(blocks, renderNotifyAppearance(s.Notify, innerWidth))
	blocks = append(blocks, renderNotifyDND(s, innerWidth))
	blocks = append(blocks, renderNotifyUrgency(s, innerWidth))
	blocks = append(blocks, renderNotifySound(s, innerWidth))

	if rules := renderNotifyRules(s, innerWidth); rules != "" {
		blocks = append(blocks, rules)
	}

	if hist := renderNotifyHistory(s.Notify, innerWidth); hist != "" {
		blocks = append(blocks, hist)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderScrollableContentPane(s, width, height, body)
}

func renderNotifyDaemon(s *core.State, total int) string {
	n := s.Notify
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("running")
	if !n.Running {
		status = lipgloss.NewStyle().Foreground(colorRed).Render("stopped")
	}
	lines = append(lines, label.Render("Status")+status)
	lines = append(lines, label.Render("Daemon")+value.Render(n.Daemon))

	posLine := label.Render("Position") + value.Render(n.Anchor)
	if s.ContentFocused {
		posLine += dim.Render("  (") + accent.Render("p") + dim.Render(" cycle)")
	}
	lines = append(lines, posLine)

	timeoutSec := fmt.Sprintf("%.1fs", float64(n.TimeoutMS)/1000)
	timeoutLine := label.Render("Timeout") + value.Render(timeoutSec)
	if s.ContentFocused {
		timeoutLine += dim.Render("  (") + accent.Render("+/-") + dim.Render(" adjust)")
	}
	lines = append(lines, timeoutLine)

	widthLine := label.Render("Width") + value.Render(fmt.Sprintf("%dpx", n.Width))
	if s.ContentFocused {
		widthLine += dim.Render("  (") + accent.Render("w/W") + dim.Render(" adjust)")
	}
	lines = append(lines, widthLine)

	lines = append(lines, label.Render("Max Visible")+value.Render(fmt.Sprintf("%d", n.MaxVisible)))
	lines = append(lines, label.Render("Max History")+value.Render(fmt.Sprintf("%d", n.MaxHistory)))

	layerLine := label.Render("Layer") + value.Render(n.Layer)
	if s.ContentFocused {
		layerLine += dim.Render("  (") + accent.Render("l") + dim.Render(" toggle)")
	}
	lines = append(lines, layerLine)

	return groupBoxSections("Notification Daemon", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderNotifyAppearance(n notifycfg.Snapshot, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	if n.Font != "" {
		lines = append(lines, label.Render("Font")+value.Render(n.Font))
	}
	if n.Padding != "" {
		lines = append(lines, label.Render("Padding")+value.Render(n.Padding))
	}
	lines = append(lines, label.Render("Border")+value.Render(fmt.Sprintf("%dpx", n.BorderSize)))
	if n.BorderRadius > 0 {
		lines = append(lines, label.Render("Border Radius")+value.Render(fmt.Sprintf("%dpx", n.BorderRadius)))
	}
	lines = append(lines, label.Render("Max Icon")+value.Render(fmt.Sprintf("%dpx", n.MaxIcon)))
	lines = append(lines, label.Render("Icons")+value.Render(onOff(n.Icons)))
	lines = append(lines, label.Render("Markup")+value.Render(onOff(n.Markup)))
	lines = append(lines, label.Render("Actions")+value.Render(onOff(n.Actions)))

	if n.TextColor != "" {
		lines = append(lines, label.Render("Text Color")+renderColorSwatch(n.TextColor))
	}
	if n.BorderColor != "" {
		lines = append(lines, label.Render("Border Color")+renderColorSwatch(n.BorderColor))
	}
	if n.BgColor != "" {
		lines = append(lines, label.Render("Background")+renderColorSwatch(n.BgColor))
	}

	return groupBoxSections("Appearance", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderColorSwatch(hex string) string {
	swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("██")
	return swatch + " " + detailValueStyle.Render(hex)
}

func renderNotifyDND(s *core.State, total int) string {
	n := s.Notify
	lw := 18
	label := detailLabelStyle.Width(lw)
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	if n.DNDActive {
		lines = append(lines, label.Render("Do Not Disturb")+
			lipgloss.NewStyle().Foreground(colorGold).Bold(true).Render("active"))
	} else {
		lines = append(lines, label.Render("Do Not Disturb")+
			lipgloss.NewStyle().Foreground(colorGreen).Render("off"))
	}

	if s.ContentFocused {
		lines = append(lines, dim.Render("  "+accent.Render("d")+" toggle DND · "+
			accent.Render("D")+" dismiss all"))
	}

	return groupBoxSections("Do Not Disturb", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderNotifyRules(s *core.State, total int) string {
	n := s.Notify
	if len(n.Rules) == 0 && !s.ContentFocused {
		return ""
	}

	lw := 28
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string
	for _, r := range n.Rules {
		lines = append(lines, label.Render(r.Criteria)+value.Render(r.Action))
	}

	if s.ContentFocused {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, dim.Render("  "+accent.Render("a")+" add app rule · "+
			accent.Render("x")+" remove rule"))
	}

	if len(lines) == 0 {
		lines = append(lines, placeholderStyle.Render("No app-specific rules configured."))
	}

	return groupBoxSections("App Rules", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderNotifyUrgency(s *core.State, total int) string {
	n := s.Notify
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	formatTimeout := func(ms int) string {
		if ms < 0 {
			return "default"
		}
		if ms == 0 {
			return "never dismiss"
		}
		return fmt.Sprintf("%.1fs", float64(ms)/1000)
	}

	var lines []string

	lines = append(lines, label.Render("Low")+value.Render(formatTimeout(n.LowTimeout)))
	lines = append(lines, label.Render("Normal")+value.Render(formatTimeout(n.TimeoutMS)))

	critLine := label.Render("Critical") + value.Render(formatTimeout(n.CritTimeout))
	if n.CritLayer != "" {
		critLine += detailValueStyle.Render("  layer: " + n.CritLayer)
	}
	lines = append(lines, critLine)

	if n.GroupFormat != "" {
		lines = append(lines, label.Render("Group Format")+value.Render(n.GroupFormat))
	}

	return groupBoxSections("Urgency Timeouts", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderNotifySound(s *core.State, total int) string {
	n := s.Notify
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)
	accent := lipgloss.NewStyle().Foreground(colorAccent)

	var lines []string

	if n.NotifySound != "" {
		sound := n.NotifySound
		// Strip "exec mpv " prefix for display
		sound = strings.TrimPrefix(sound, "exec mpv ")
		lines = append(lines, label.Render("Sound")+
			lipgloss.NewStyle().Foreground(colorGreen).Render("enabled"))
		lines = append(lines, label.Render("File")+value.Render(sound))
	} else {
		lines = append(lines, label.Render("Sound")+
			lipgloss.NewStyle().Foreground(colorDim).Render("disabled"))
	}

	if s.ContentFocused {
		lines = append(lines, dim.Render("  "+accent.Render("o")+" set sound · "+
			accent.Render("O")+" disable sound"))
	}

	if len(n.Sounds) > 0 {
		lines = append(lines, "")
		lines = append(lines, detailLabelStyle.Render("Available:"))
		row := "  "
		for i, snd := range n.Sounds {
			if i > 0 {
				row += "  "
			}
			if len(row)+len(snd) > 60 {
				lines = append(lines, dim.Render(row))
				row = "  "
			}
			row += snd
		}
		if row != "  " {
			lines = append(lines, dim.Render(row))
		}
	}

	return groupBoxSections("Notification Sound", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderNotifyHistory(n notifycfg.Snapshot, total int) string {
	if len(n.History) == 0 {
		return ""
	}

	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle
	dim := lipgloss.NewStyle().Foreground(colorDim)

	var lines []string
	limit := len(n.History)
	if limit > 10 {
		limit = 10
	}
	for _, h := range n.History[:limit] {
		app := h.AppName
		if app == "" {
			app = "unknown"
		}
		summary := h.Summary
		if len(summary) > 40 {
			summary = summary[:39] + "…"
		}
		urgLabel := ""
		if h.Urgency == "critical" {
			urgLabel = lipgloss.NewStyle().Foreground(colorRed).Render(" [critical]")
		}
		lines = append(lines, label.Render(app)+value.Render(summary)+urgLabel)
		if h.Body != "" {
			body := h.Body
			if len(body) > 50 {
				body = body[:49] + "…"
			}
			lines = append(lines, dim.Render("  "+body))
		}
	}

	return groupBoxSections("Recent History", []string{strings.Join(lines, "\n")}, total, colorBorder)
}
