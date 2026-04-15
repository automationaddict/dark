package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/appstore"
)

func renderUpdateContent(s *core.State, width, height int) string {
	secs := core.UpdateSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebar := renderInnerSidebar(s, entries, s.UpdateSectionIdx, height)
	contentWidth := width - lipgloss.Width(sidebar)

	var content string
	switch s.ActiveUpdateSection().ID {
	case "omarchy":
		content = renderOmarchyUpdateSection(s, contentWidth, height)
	case "firmware":
		content = renderFirmwareUpdateSection(s, contentWidth, height)
	case "dark":
		content = renderDarkUpdateSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderOmarchyUpdateSection(s *core.State, width, height int) string {
	if !s.UpdateLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Checking for updates…"))
	}

	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}

	title := contentTitle.Render("Omarchy Updates")

	var versionLines []string
	versionLines = append(versionLines,
		detailRow("Current Version", s.Update.CurrentVersion, innerWidth))
	if s.Update.UpdateAvailable {
		versionLines = append(versionLines,
			detailRow("Available Version", statusOnlineStyle.Render(s.Update.AvailableVersion), innerWidth))
	} else {
		versionLines = append(versionLines,
			detailRow("Available Version", s.Update.AvailableVersion, innerWidth))
	}
	versionLines = append(versionLines,
		detailRow("Channel", s.Update.Channel, innerWidth))
	versionBox := groupBoxSections("Version", versionLines, innerWidth, colorBorder)

	var statusText string
	if s.UpdateBusy {
		statusText = statusBusyStyle.Render("Updating…")
	} else if s.Update.UpdateAvailable {
		statusText = statusOnlineStyle.Render("Update available")
	} else {
		statusText = placeholderStyle.Render("System is up to date")
	}
	statusLine := detailRow("Status", statusText, innerWidth)

	var resultSection string
	if s.UpdateResult != nil {
		var stepLines []string
		for _, step := range s.UpdateResult.Steps {
			icon := "✓"
			style := statusOnlineStyle
			if step.Error != "" {
				icon = "✗"
				style = statusOfflineStyle
			}
			stepLines = append(stepLines, style.Render(icon)+" "+step.Step)
		}
		if s.UpdateResult.RebootNeeded {
			stepLines = append(stepLines, "",
				statusBusyStyle.Render("⚠ Reboot recommended"))
		}
		resultSection = groupBoxSections("Progress", []string{
			strings.Join(stepLines, "\n"),
		}, innerWidth, colorBorder)
	}

	var statusMsg string
	if s.UpdateStatusMsg != "" {
		statusMsg = placeholderStyle.Render(s.UpdateStatusMsg)
	}

	var hint string
	if s.UpdateBusy {
		hint = ""
	} else if s.Update.UpdateAvailable {
		hint = lipgloss.NewStyle().Foreground(colorDim).Render(
			"u update · c channel")
	} else {
		hint = lipgloss.NewStyle().Foreground(colorDim).Render(
			"u check for updates · c channel")
	}

	blocks := []string{title, "", statusLine, "", versionBox}
	if resultSection != "" {
		blocks = append(blocks, "", resultSection)
	}
	if statusMsg != "" {
		blocks = append(blocks, "", statusMsg)
	}
	if hint != "" {
		blocks = append(blocks, "", hint)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderDarkUpdateSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}

	title := contentTitle.Render("Dark Self-Update")

	current := s.DarkUpdate.Current
	if current == "" {
		current = "(unknown)"
	}
	latest := s.DarkUpdate.Latest
	if latest == "" {
		latest = placeholderStyle.Render("not checked")
	} else if s.DarkUpdate.UpdateAvailable {
		latest = statusOnlineStyle.Render(s.DarkUpdate.Latest)
	}

	versionLines := []string{
		detailRow("Current", current, innerWidth),
		detailRow("Latest", latest, innerWidth),
	}
	if !s.DarkUpdate.LastCheckedAt.IsZero() {
		versionLines = append(versionLines,
			detailRow("Last Checked", s.DarkUpdate.LastCheckedAt.Format("2006-01-02 15:04"), innerWidth))
	}
	if !s.DarkUpdate.LatestPublished.IsZero() {
		versionLines = append(versionLines,
			detailRow("Published", s.DarkUpdate.LatestPublished.Format("2006-01-02"), innerWidth))
	}
	if s.DarkUpdate.InstalledAt != "" {
		versionLines = append(versionLines,
			detailRow("Installed At", s.DarkUpdate.InstalledAt, innerWidth))
	}
	versionBox := groupBoxSections("Version", versionLines, innerWidth, colorBorder)

	var statusText string
	switch {
	case s.DarkUpdateApplying:
		statusText = statusBusyStyle.Render("Applying update…")
	case s.DarkUpdateChecking:
		statusText = statusBusyStyle.Render("Checking GitHub…")
	case s.DarkUpdate.UpdateAvailable:
		statusText = statusOnlineStyle.Render("Update available")
	case s.DarkUpdate.Latest != "":
		statusText = placeholderStyle.Render("dark is up to date")
	default:
		statusText = placeholderStyle.Render("Press c to check for updates")
	}
	statusLine := detailRow("Status", statusText, innerWidth)

	blocks := []string{title, "", statusLine, "", versionBox}

	if notes := strings.TrimSpace(s.DarkUpdate.LatestNotes); notes != "" {
		if len(notes) > 600 {
			notes = notes[:600] + "…"
		}
		notesBox := groupBoxSections("Release Notes", []string{notes}, innerWidth, colorBorder)
		blocks = append(blocks, "", notesBox)
	}

	if s.DarkUpdate.LastCheckError != "" {
		blocks = append(blocks, "",
			statusOfflineStyle.Render("Check error: "+s.DarkUpdate.LastCheckError))
	}
	if s.DarkUpdate.ApplyError != "" {
		blocks = append(blocks, "",
			statusOfflineStyle.Render("Apply error: "+s.DarkUpdate.ApplyError))
	}
	if s.DarkUpdateActionError != "" {
		blocks = append(blocks, "",
			statusOfflineStyle.Render(s.DarkUpdateActionError))
	}

	var hint string
	if s.DarkUpdateApplying || s.DarkUpdateChecking {
		hint = ""
	} else if s.DarkUpdate.UpdateAvailable {
		hint = lipgloss.NewStyle().Foreground(colorDim).Render(
			"u install update · c re-check")
	} else {
		hint = lipgloss.NewStyle().Foreground(colorDim).Render(
			"c check for updates")
	}
	if hint != "" {
		blocks = append(blocks, "", hint)
	}

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderFirmwareUpdateSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}

	title := contentTitle.Render("Firmware")

	if !s.FirmwareLoaded {
		body := lipgloss.JoinVertical(lipgloss.Left,
			title, "", placeholderStyle.Render("Checking firmware…"))
		return renderContentPane(width, height, body)
	}

	if !s.Firmware.Available {
		hint := lipgloss.NewStyle().Foreground(colorDim).Render("i install fwupd")
		body := lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			placeholderStyle.Render("fwupd is not installed"),
			"", hint)
		return renderContentPane(width, height, body)
	}

	if len(s.Firmware.Devices) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left,
			title, "", placeholderStyle.Render("No updatable firmware devices found"))
		return renderContentPane(width, height, body)
	}

	var deviceLines []string
	for _, d := range s.Firmware.Devices {
		line := fmt.Sprintf("%-24s %s", d.Name, d.Version)
		if d.Vendor != "" {
			line = fmt.Sprintf("%-24s %-12s %s", d.Name, d.Version, d.Vendor)
		}
		deviceLines = append(deviceLines, line)
	}

	var statusText string
	if s.Firmware.Updates > 0 {
		statusText = statusOnlineStyle.Render(
			fmt.Sprintf("%d update(s) available", s.Firmware.Updates))
	} else {
		statusText = placeholderStyle.Render("All firmware is up to date")
	}

	devBox := groupBoxSections(
		fmt.Sprintf("Devices (%d)", len(s.Firmware.Devices)),
		[]string{strings.Join(deviceLines, "\n")},
		innerWidth, colorBorder)

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "", statusText, "", devBox)
	return renderContentPane(width, height, body)
}

func detailRow(label, value string, _ int) string {
	l := detailLabelStyle.Width(20).Render(label)
	return l + value
}

func (m *Model) triggerOmarchyUpdate() tea.Cmd {
	if m.state.UpdateBusy || m.update.Run == nil {
		return nil
	}
	if m.state.Update.UpdateAvailable {
		m.dialog = NewDialog("Run Omarchy update?", nil, func(_ DialogResult) tea.Cmd {
			m.state.MarkUpdateBusy()
			return m.update.Run()
		})
		return nil
	}
	// No update available — just re-check
	m.state.MarkUpdateBusy()
	return m.update.Run()
}

func (m *Model) triggerChannelChange() tea.Cmd {
	if m.state.UpdateBusy || m.update.ChangeChannel == nil {
		return nil
	}
	m.dialog = NewDialog("Update channel…", []DialogFieldSpec{
		{
			Key:     "channel",
			Label:   "Channel",
			Kind:    DialogFieldSelect,
			Value:   m.state.Update.Channel,
			Options: []string{"stable", "rc", "edge", "dev"},
		},
	}, func(result DialogResult) tea.Cmd {
		ch := result["channel"]
		if ch == "" || ch == m.state.Update.Channel {
			return nil
		}
		return m.update.ChangeChannel(ch)
	})
	return nil
}

func renderFirmwareSection(s *core.State, innerWidth int) string {
	title := contentTitle.Render("Firmware")

	if !s.FirmwareLoaded {
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "", placeholderStyle.Render("Checking firmware…"))
	}

	if !s.Firmware.Available {
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "", placeholderStyle.Render("fwupd is not installed"))
	}

	if len(s.Firmware.Devices) == 0 {
		return lipgloss.JoinVertical(lipgloss.Left,
			title, "", placeholderStyle.Render("No updatable firmware devices found"))
	}

	var deviceLines []string
	for _, d := range s.Firmware.Devices {
		line := fmt.Sprintf("%-24s %s", d.Name, d.Version)
		if d.Vendor != "" {
			line = fmt.Sprintf("%-24s %-12s %s", d.Name, d.Version, d.Vendor)
		}
		deviceLines = append(deviceLines, line)
	}

	var statusText string
	if s.Firmware.Updates > 0 {
		statusText = statusOnlineStyle.Render(
			fmt.Sprintf("%d update(s) available", s.Firmware.Updates))
	} else {
		statusText = placeholderStyle.Render("All firmware is up to date")
	}

	devBox := groupBoxSections(
		fmt.Sprintf("Devices (%d)", len(s.Firmware.Devices)),
		[]string{strings.Join(deviceLines, "\n")},
		innerWidth, colorBorder)

	return lipgloss.JoinVertical(lipgloss.Left,
		title, "", statusText, "", devBox)
}

func (m *Model) inF2Updates() bool {
	return m.state.ActiveTab == core.TabF2 && m.state.F2OnUpdates()
}

func (m *Model) inF2Firmware() bool {
	return m.inF2Updates() && m.state.ActiveUpdateSection().ID == "firmware"
}

func (m *Model) triggerFwupdInstall() tea.Cmd {
	if m.state.Firmware.Available {
		return nil
	}
	if m.appstore.Install == nil {
		return nil
	}
	m.dialog = NewDialog("Install fwupd?", nil, func(_ DialogResult) tea.Cmd {
		m.state.AppstoreBusy = true
		return m.appstore.Install(appstore.InstallRequest{
			Names:  []string{"fwupd"},
			Origin: appstore.OriginPacman,
		})
	})
	return nil
}
