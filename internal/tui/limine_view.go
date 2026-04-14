package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
)

func renderLimineSection(s *core.State, width, height int) string {
	secs := core.LimineSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.LimineContentFocused
	sidebar := renderInnerSidebarFocused(s, entries, s.LimineSubIdx, height, sidebarFocused)
	innerWidth := width - lipgloss.Width(sidebar)

	if !s.LimineLoaded {
		return lipgloss.JoinHorizontal(lipgloss.Top, sidebar,
			renderContentPane(innerWidth, height,
				placeholderStyle.Render("Loading limine state…")))
	}
	if !s.Limine.Available {
		msg := "Limine is not available on this host."
		if s.Limine.Error != "" {
			msg += "\n\n" + s.Limine.Error
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, sidebar,
			renderContentPane(innerWidth, height, placeholderStyle.Render(msg)))
	}

	var content string
	switch s.ActiveLimineSection().ID {
	case "snapshots":
		content = renderLimineSnapshots(s, innerWidth, height)
	case "boot":
		content = renderLimineBootConfig(s, innerWidth, height)
	case "sync":
		content = renderLimineSyncConfig(s, innerWidth, height)
	case "omarchy":
		content = renderLimineOmarchyConfig(s, innerWidth, height)
	default:
		content = renderContentPane(innerWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderLimineSnapshots(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 40 {
		innerWidth = 40
	}
	title := contentTitle.Render("Limine Boot Snapshots")

	if len(s.Limine.Snapshots) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			placeholderStyle.Render("No snapshot entries in /boot/limine.conf."), "",
			lipgloss.NewStyle().Foreground(colorDim).Render(
				"a create new · s sync now"),
		)
		return renderContentPane(width, height, body)
	}

	numW := 6
	tsW := 22
	typeW := 10
	kernelW := 12
	subvolW := innerWidth - numW - tsW - typeW - kernelW - 5
	if subvolW < 16 {
		subvolW = 16
	}

	selectedCell := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	var data [][]string
	for _, snap := range s.Limine.Snapshots {
		num := "-"
		if snap.Number > 0 {
			num = fmt.Sprintf("%d", snap.Number)
		}
		data = append(data, []string{
			num,
			snap.Timestamp,
			snap.Type,
			snap.Kernel,
			snap.Subvol,
		})
	}

	table := renderTable(
		[]string{"#", "Timestamp", "Type", "Kernel", "Subvol"},
		[]int{numW, tsW, typeW, kernelW, subvolW},
		data,
		s.LimineSnapshotIdx, s.LimineContentFocused, selectedCell,
	)

	hint := lipgloss.NewStyle().Foreground(colorDim).Render(
		"a create · s sync · d delete")

	summary := lipgloss.NewStyle().Foreground(colorDim).Render(
		fmt.Sprintf("%d snapshot entries · default_entry %d", len(s.Limine.Snapshots), s.Limine.DefaultEntry))

	bodyParts := []string{title, "", table, "", hint, "", summary}
	if s.LimineActionError != "" {
		errStyle := lipgloss.NewStyle().Foreground(colorAccent)
		bodyParts = append(bodyParts, "", errStyle.Render("error: "+s.LimineActionError))
	}
	if s.LimineBusy {
		bodyParts = append(bodyParts, "", placeholderStyle.Render("working… polkit prompt may be open"))
	}
	body := lipgloss.JoinVertical(lipgloss.Left, bodyParts...)
	return renderContentPane(width, height, strings.TrimRight(body, "\n"))
}

func renderLimineBootConfig(s *core.State, width, height int) string {
	rows := s.LimineBootConfigRows()
	body := lipgloss.JoinVertical(lipgloss.Left,
		contentTitle.Render("Limine Boot Config"), "",
		lipgloss.NewStyle().Foreground(colorDim).Render("path: "+s.Limine.BootConfig.Path), "",
		renderEditableRows(rows, s.LimineBootCfgIdx, s.LimineContentFocused),
		"",
		lipgloss.NewStyle().Foreground(colorDim).Render(
			"enter edit · keys at the top of /boot/limine.conf — entry blocks below are left untouched"),
	)
	return renderContentPane(width, height, body)
}

func renderLimineSyncConfig(s *core.State, width, height int) string {
	rows := s.LimineSyncConfigRows()
	body := lipgloss.JoinVertical(lipgloss.Left,
		contentTitle.Render("Limine ↔ Snapper Sync Config"), "",
		lipgloss.NewStyle().Foreground(colorDim).Render("path: "+s.Limine.SyncConfig.Path), "",
		renderEditableRows(rows, s.LimineSyncCfgIdx, s.LimineContentFocused),
		"",
		lipgloss.NewStyle().Foreground(colorDim).Render(
			"enter edit · controls how limine-snapper-sync writes snapshot entries"),
	)
	return renderContentPane(width, height, body)
}

func renderLimineOmarchyConfig(s *core.State, width, height int) string {
	rows := s.LimineOmarchyConfigRows()
	cmdLine := core.LimineConfigRow{Label: "kernel_cmdline", Key: "KERNEL_CMDLINE"}
	all := append([]core.LimineConfigRow(nil), rows...)
	all = append(all, cmdLine)

	rowsView := renderEditableRows(all, s.LimineOmarchyCfgIdx, s.LimineContentFocused)

	parts := []string{
		contentTitle.Render("Omarchy Limine Defaults"), "",
		lipgloss.NewStyle().Foreground(colorDim).Render("path: "+s.Limine.OmarchyConfig.Path), "",
		rowsView,
	}
	if len(s.Limine.OmarchyConfig.KernelCmdline) > 0 {
		parts = append(parts, "", lipgloss.NewStyle().Foreground(colorAccent).Render("KERNEL_CMDLINE[default]:"))
		for _, line := range s.Limine.OmarchyConfig.KernelCmdline {
			parts = append(parts, "  "+strings.TrimSpace(line))
		}
	}
	parts = append(parts, "",
		lipgloss.NewStyle().Foreground(colorDim).Render(
			"enter edit · /etc/default/limine, consumed by limine-mkinitcpio-hook"),
	)
	body := lipgloss.JoinVertical(lipgloss.Left, parts...)
	return renderContentPane(width, height, body)
}

// renderEditableRows draws a focusable list of key/value pairs,
// highlighting the selected row when the content pane owns focus.
func renderEditableRows(rows []core.LimineConfigRow, selected int, focused bool) string {
	keyStyle := lipgloss.NewStyle().Foreground(colorAccent).Width(24)
	valStyle := lipgloss.NewStyle().Foreground(colorDim)
	selectedStyle := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent).
		Width(24)
	selectedVal := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	var lines []string
	for i, r := range rows {
		val := r.Value
		if val == "" {
			val = "—"
		}
		if i == selected && focused {
			lines = append(lines, selectedStyle.Render(r.Label)+" "+selectedVal.Render(" "+val+" "))
		} else {
			lines = append(lines, keyStyle.Render(r.Label)+" "+valStyle.Render(val))
		}
	}
	return strings.Join(lines, "\n")
}
