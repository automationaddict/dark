package tui

import (
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

	blocks = append(blocks, renderNotifyDaemon(s.Notify, innerWidth))
	blocks = append(blocks, renderNotifyAppearance(s.Notify, innerWidth))
	blocks = append(blocks, renderNotifyDND(s.Notify, innerWidth))

	if rules := renderNotifyRules(s.Notify, innerWidth); rules != "" {
		blocks = append(blocks, rules)
	}

	if hist := renderNotifyHistory(s.Notify, innerWidth); hist != "" {
		blocks = append(blocks, hist)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderScrollableContentPane(s, width, height, body)
}

func renderNotifyDaemon(n notifycfg.Snapshot, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	status := lipgloss.NewStyle().Foreground(colorGreen).Bold(true).Render("running")
	if !n.Running {
		status = lipgloss.NewStyle().Foreground(colorRed).Render("stopped")
	}
	lines = append(lines, label.Render("Status")+status)
	lines = append(lines, label.Render("Daemon")+value.Render(n.Daemon))

	if n.Anchor != "" {
		lines = append(lines, label.Render("Position")+value.Render(n.Anchor))
	}
	if n.Timeout != "" {
		lines = append(lines, label.Render("Timeout")+value.Render(n.Timeout))
	}

	return groupBoxSections("Notification Daemon", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderNotifyAppearance(n notifycfg.Snapshot, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string

	if n.Width != "" {
		lines = append(lines, label.Render("Width")+value.Render(n.Width))
	}
	if n.Padding != "" {
		lines = append(lines, label.Render("Padding")+value.Render(n.Padding))
	}
	if n.BorderSize != "" {
		lines = append(lines, label.Render("Border")+value.Render(n.BorderSize))
	}
	if n.Font != "" {
		lines = append(lines, label.Render("Font")+value.Render(n.Font))
	}
	if n.MaxIcon != "" {
		lines = append(lines, label.Render("Max Icon")+value.Render(n.MaxIcon))
	}

	if n.TextColor != "" {
		lines = append(lines, label.Render("Text Color")+
			renderColorSwatch(n.TextColor))
	}
	if n.BorderColor != "" {
		lines = append(lines, label.Render("Border Color")+
			renderColorSwatch(n.BorderColor))
	}
	if n.BgColor != "" {
		lines = append(lines, label.Render("Background")+
			renderColorSwatch(n.BgColor))
	}

	if len(lines) == 0 {
		return ""
	}
	return groupBoxSections("Appearance", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderColorSwatch(hex string) string {
	swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(hex)).Render("██")
	return swatch + " " + detailValueStyle.Render(hex)
}

func renderNotifyDND(n notifycfg.Snapshot, total int) string {
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

	lines = append(lines, dim.Render("  "+accent.Render("d")+" toggle DND · "+
		accent.Render("D")+" dismiss all"))

	return groupBoxSections("Do Not Disturb", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

func renderNotifyRules(n notifycfg.Snapshot, total int) string {
	if len(n.Rules) == 0 {
		return ""
	}

	lw := 28
	label := detailLabelStyle.Width(lw)
	value := detailValueStyle

	var lines []string
	for _, r := range n.Rules {
		lines = append(lines, label.Render(r.Criteria)+value.Render(r.Action))
	}

	return groupBoxSections("App Rules", []string{strings.Join(lines, "\n")}, total, colorBorder)
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
