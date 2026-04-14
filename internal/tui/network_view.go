package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/network"
)

func renderNetwork(s *core.State, width, height int) string {
	if !s.NetworkLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading network state…"),
		)
	}
	if len(s.Network.Interfaces) == 0 {
		title := contentTitle.Render("Network")
		body := placeholderStyle.Render("No network interfaces detected.")
		return renderContentPane(width, height,
			lipgloss.JoinVertical(lipgloss.Left, title, body),
		)
	}

	secs := core.NetworkSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.NetworkContentFocused
	sidebar := renderInnerSidebarFocused(s, entries, s.NetworkSectionIdx, height, sidebarFocused)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActiveNetworkSection()
	var content string
	switch sec.ID {
	case "interfaces":
		content = renderNetworkInterfacesSection(s, contentWidth, height)
	case "dns":
		content = renderNetworkDNSSection(s, contentWidth, height)
	case "routes":
		content = renderNetworkRoutesSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

// ── Interfaces section ──────────────────────────────────────────────

func renderNetworkInterfacesSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	focused := s.NetworkContentFocused
	selected := s.NetworkSelected
	if selected >= len(s.Network.Interfaces) {
		selected = 0
	}

	airplaneBox := renderNetworkAirplane(s, innerWidth)

	interfacesBox := groupBoxSections("Interfaces",
		[]string{renderNetworkInterfacesTable(s.Network.Interfaces, selected, focused)},
		innerWidth, borderForFocus(focused))

	blocks := []string{airplaneBox, interfacesBox}

	if iface, ok := s.SelectedNetworkInterface(); ok {
		detailsTitle := "Details"
		if iface.Name != "" {
			detailsTitle = "Details · " + iface.Name
		}
		detailsBox := groupBoxSections(detailsTitle,
			[]string{renderNetworkInterfaceDetails(iface)}, innerWidth, colorBorder)
		blocks = append(blocks, "", detailsBox)
	}

	if s.NetworkBusy {
		blocks = append(blocks, "", statusBusyStyle.Render("working…"))
	}

	blocks = append(blocks, renderNetworkInterfacesFocusHint(s, focused))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

// ── DNS section ─────────────────────────────────────────────────────

func renderNetworkDNSSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	dns := s.Network.DNS
	if len(dns.Servers) == 0 && len(dns.Search) == 0 {
		body := placeholderStyle.Render("No DNS information available.")
		return renderContentPane(width, height, body)
	}

	rows := [][2]string{}
	if len(dns.Servers) > 0 {
		rows = append(rows, [2]string{"Servers", strings.Join(dns.Servers, ", ")})
	}
	if len(dns.Search) > 0 {
		rows = append(rows, [2]string{"Search", strings.Join(dns.Search, ", ")})
	}
	if dns.Source != "" {
		rows = append(rows, [2]string{"Source", dns.Source})
	}

	box := groupBoxSections("DNS", []string{renderDetailRows(rows)}, innerWidth, colorBorder)

	// Also show per-interface DNS when available.
	var blocks []string
	blocks = append(blocks, box)

	for _, iface := range s.Network.Interfaces {
		if iface.Management == nil {
			continue
		}
		mi := iface.Management
		if len(mi.DNS) == 0 && len(mi.Domains) == 0 {
			continue
		}
		var ifRows [][2]string
		if len(mi.DNS) > 0 {
			ifRows = append(ifRows, [2]string{"DNS", strings.Join(mi.DNS, ", ")})
		}
		if len(mi.Domains) > 0 {
			ifRows = append(ifRows, [2]string{"Domains", strings.Join(mi.Domains, ", ")})
		}
		lw := 0
		for _, r := range ifRows {
			if w := lipgloss.Width(r[0]); w > lw {
				lw = w
			}
		}
		var ifLines []string
		for _, r := range ifRows {
			label := detailLabelStyle.Width(lw + 2).Render(r[0])
			value := detailValueStyle.Render(orDash(r[1]))
			ifLines = append(ifLines, label+value)
		}
		ifBox := groupBoxSections("Link DNS · "+iface.Name,
			[]string{strings.Join(ifLines, "\n")}, innerWidth, colorBorder)
		blocks = append(blocks, "", ifBox)
	}

	hint := statusBarStyle.Render("esc back")
	blocks = append(blocks, hint)

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

// ── Routes section ──────────────────────────────────────────────────

func renderNetworkRoutesSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	if s.NetworkRoutesOpen {
		return renderNetworkRoutesDrillIn(s, width, height, innerWidth)
	}

	var blocks []string

	// Kernel routing table.
	if len(s.Network.Routes) > 0 {
		routesBox := renderNetworkRoutesBox(s.Network.Routes, innerWidth)
		blocks = append(blocks, routesBox)
	} else {
		blocks = append(blocks, placeholderStyle.Render("No routes in kernel table."))
	}

	// Show managed routes summary per interface.
	for _, iface := range s.Network.Interfaces {
		if iface.Managed == nil || len(iface.Managed.Routes) == 0 {
			continue
		}
		table := renderNetworkManagedRoutesTable(iface.Managed.Routes, -1)
		box := groupBoxSections("Managed Routes · "+iface.Name,
			[]string{table}, innerWidth, colorBorder)
		blocks = append(blocks, "", box)
	}

	focused := s.NetworkContentFocused
	hint := renderNetworkRoutesFocusHint(s, focused)
	blocks = append(blocks, hint)

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

// renderNetworkRoutesDrillIn replaces the routes section content with
// a focused routes-management page for the highlighted interface.
func renderNetworkRoutesDrillIn(s *core.State, width, height, innerWidth int) string {
	iface, ok := s.SelectedNetworkInterface()
	if !ok {
		s.NetworkRoutesOpen = false
		return renderContentPane(width, height,
			placeholderStyle.Render("interface no longer present"),
		)
	}

	title := "Routes · " + iface.Name + " (managed by dark)"

	var body string
	if iface.Managed == nil || len(iface.Managed.Routes) == 0 {
		body = placeholderStyle.Render("no managed routes — press a to add one")
	} else {
		body = renderNetworkManagedRoutesTable(iface.Managed.Routes, s.NetworkRouteSelected)
	}

	box := groupBoxSections(title, []string{body}, innerWidth, colorAccent)

	hint := statusBarStyle.Render("a add · d delete · esc back")

	rendered := lipgloss.JoinVertical(lipgloss.Left, box, "", hint)
	return renderContentPane(width, height, rendered)
}

// ── Shared rendering helpers ────────────────────────────────────────

func renderNetworkAirplane(s *core.State, total int) string {
	lw := 18
	label := detailLabelStyle.Width(lw)

	status := lipgloss.NewStyle().Foreground(colorDim).Render("off")
	icon := "󰀝"
	if s.Network.AirplaneMode {
		status = lipgloss.NewStyle().Foreground(colorGold).Bold(true).Render("on")
		icon = "󰀞"
	}
	line := label.Render(icon+"  Airplane Mode") + status
	hint := lipgloss.NewStyle().Foreground(colorDim).Render("  A toggle")

	return groupBoxSections("", []string{line, hint}, total, colorBorder)
}

func renderNetworkInterfacesTable(ifaces []network.Interface, selected int, focused bool) string {
	type col struct {
		header string
		cell   func(network.Interface) string
		accent func(network.Interface) bool
	}
	cols := []col{
		{"Name", func(i network.Interface) string { return i.Name }, nil},
		{"Type", func(i network.Interface) string { return orDash(i.Type) }, nil},
		{"State", func(i network.Interface) string { return formatInterfaceState(i) },
			func(i network.Interface) bool { return i.State == "up" && i.Carrier }},
		{"IPv4", func(i network.Interface) string { return primaryAddress(i.IPv4) }, nil},
		{"IPv6", func(i network.Interface) string { return primaryAddress(i.IPv6) }, nil},
		{"Speed", func(i network.Interface) string { return formatLinkSpeed(i.SpeedMbps) }, nil},
		{"Rate", func(i network.Interface) string { return formatNetworkRate(i.RxRateBps, i.TxRateBps) }, nil},
	}

	colW := make([]int, len(cols))
	for i, c := range cols {
		colW[i] = lipgloss.Width(c.header)
	}
	for _, iface := range ifaces {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(iface)); w > colW[i] {
				colW[i] = w
			}
		}
	}

	const gap = "  "
	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(colW[i]).Render(c.header))
	}
	lines := []string{"  " + strings.Join(headerCells, gap)}

	for i, iface := range ifaces {
		isSel := i == selected
		var marker string
		switch {
		case isSel && focused:
			marker = tableSelectionMarker.Render("▸ ")
		case isSel:
			marker = tableSelectionMarkerDim.Render("▸ ")
		default:
			marker = "  "
		}
		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.cell(iface)
			var style lipgloss.Style
			switch {
			case isSel:
				style = tableCellSelected
			case c.accent != nil && c.accent(iface):
				style = tableCellAccent
			default:
				style = tableCellStyle
			}
			cells = append(cells, style.Width(colW[j]).Render(text))
		}
		lines = append(lines, marker+strings.Join(cells, gap))
	}
	return strings.Join(lines, "\n")
}

func renderNetworkInterfaceDetails(iface network.Interface) string {
	rows := [][2]string{
		{"Name", orDash(iface.Name)},
		{"Type", orDash(iface.Type)},
		{"Driver", orDash(iface.Driver)},
		{"MAC", orDash(iface.MAC)},
		{"MTU", formatMTU(iface.MTU)},
		{"State", formatInterfaceState(iface)},
		{"Link", formatCarrier(iface.Carrier, iface.SpeedMbps, iface.Duplex)},
		{"IPv4", strings.Join(addressList(iface.IPv4), ", ")},
		{"IPv6", strings.Join(addressList(iface.IPv6), ", ")},
		{"RX bytes", formatNetworkBytes(iface.RxBytes)},
		{"TX bytes", formatNetworkBytes(iface.TxBytes)},
		{"RX packets", formatPackets(iface.RxPackets)},
		{"TX packets", formatPackets(iface.TxPackets)},
		{"Rate", formatNetworkRate(iface.RxRateBps, iface.TxRateBps)},
	}

	if iface.Management != nil {
		mi := iface.Management
		rows = append(rows,
			[2]string{"", ""},
			[2]string{"Managed by", mi.BackendName},
			[2]string{"Admin state", orDash(mi.AdminState)},
		)
		if mi.OnlineState != "" {
			rows = append(rows, [2]string{"Online state", mi.OnlineState})
		}
		if mi.Source != "" {
			rows = append(rows, [2]string{"Config", mi.Source})
		}
		if mi.DHCPv4 != "" {
			rows = append(rows, [2]string{"DHCPv4", mi.DHCPv4})
		}
		if mi.DHCPv6 != "" && mi.DHCPv6 != "stopped" {
			rows = append(rows, [2]string{"DHCPv6", mi.DHCPv6})
		}
		if len(mi.DNS) > 0 {
			rows = append(rows, [2]string{"DNS (link)", strings.Join(mi.DNS, ", ")})
		}
		if len(mi.Domains) > 0 {
			rows = append(rows, [2]string{"Domains", strings.Join(mi.Domains, ", ")})
		}
		if mi.Required != nil {
			rows = append(rows, [2]string{"Required", yesNo(*mi.Required)})
		}
	}

	return renderDetailRows(rows)
}

func renderNetworkManagedRoutesTable(routes []network.RouteConfig, selected int) string {
	type col struct {
		header string
		cell   func(network.RouteConfig) string
	}
	cols := []col{
		{"Destination", func(r network.RouteConfig) string { return orDash(r.Destination) }},
		{"Gateway", func(r network.RouteConfig) string {
			if r.Gateway == "" {
				return "(on-link)"
			}
			return r.Gateway
		}},
		{"Metric", func(r network.RouteConfig) string {
			if r.Metric == 0 {
				return "—"
			}
			return fmt.Sprintf("%d", r.Metric)
		}},
	}

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, r := range routes {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(r)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = "  "
	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines := []string{"  " + strings.Join(headerCells, gap)}

	for i, r := range routes {
		isSel := i == selected
		var marker string
		if isSel {
			marker = tableSelectionMarker.Render("▸ ")
		} else {
			marker = "  "
		}
		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.cell(r)
			style := tableCellStyle
			if isSel {
				style = tableCellSelected
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		lines = append(lines, marker+strings.Join(cells, gap))
	}
	return strings.Join(lines, "\n")
}

func renderNetworkRoutesBox(routes []network.Route, total int) string {
	type col struct {
		header string
		cell   func(network.Route) string
	}
	cols := []col{
		{"Destination", func(r network.Route) string { return orDash(r.Destination) }},
		{"Gateway", func(r network.Route) string { return orDash(r.Gateway) }},
		{"Interface", func(r network.Route) string { return orDash(r.Interface) }},
		{"Metric", func(r network.Route) string {
			if r.Metric == 0 {
				return "—"
			}
			return fmt.Sprintf("%d", r.Metric)
		}},
		{"Family", func(r network.Route) string { return r.Family }},
	}
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, r := range routes {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(r)); w > widths[i] {
				widths[i] = w
			}
		}
	}
	const gap = "  "
	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines := []string{"  " + strings.Join(headerCells, gap)}
	for _, r := range routes {
		cells := make([]string, 0, len(cols))
		isDefault := r.Destination == "default"
		for j, c := range cols {
			text := c.cell(r)
			style := tableCellStyle
			if isDefault {
				style = tableCellAccent
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		lines = append(lines, "  "+strings.Join(cells, gap))
	}
	return groupBoxSections("Routes", []string{strings.Join(lines, "\n")}, total, colorBorder)
}

// ── Focus hints ─────────────────────────────────────────────────────

func renderNetworkInterfacesFocusHint(s *core.State, focused bool) string {
	if len(s.Network.Interfaces) == 0 {
		return ""
	}
	backend := s.Network.Backend
	if backend == "" || backend == network.BackendNone {
		backend = "no manager detected (read-only)"
	}
	var text string
	if focused {
		text = "j/k · r reconfig · h DHCP · e edit · R reset · A airplane · esc · backend: " + backend
	} else {
		text = "enter to select · A airplane · backend: " + backend
	}
	return statusBarStyle.Render(text)
}

func renderNetworkRoutesFocusHint(s *core.State, focused bool) string {
	if !focused {
		return statusBarStyle.Render("enter to select · t manage routes")
	}
	return statusBarStyle.Render("t manage routes · esc")
}
