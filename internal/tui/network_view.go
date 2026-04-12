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
		return contentStyle.Width(width).Height(height).Render(
			placeholderStyle.Render("loading network state…"),
		)
	}
	if len(s.Network.Interfaces) == 0 {
		title := contentTitle.Render("Network")
		body := placeholderStyle.Render("No network interfaces detected.")
		return contentStyle.Width(width).Height(height).Render(
			lipgloss.JoinVertical(lipgloss.Left, title, body),
		)
	}

	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	focused := s.ContentFocused
	selected := s.NetworkSelected
	if selected >= len(s.Network.Interfaces) {
		selected = 0
	}

	interfacesBorder := colorBorder
	if focused {
		interfacesBorder = colorAccent
	}
	interfacesBox := groupBoxSections("Interfaces",
		[]string{renderNetworkInterfacesTable(s.Network.Interfaces, selected, focused)},
		innerWidth, interfacesBorder)

	blocks := []string{interfacesBox}

	if iface, ok := s.SelectedNetworkInterface(); ok {
		detailsTitle := "Details"
		if iface.Name != "" {
			detailsTitle = "Details · " + iface.Name
		}
		detailsBox := groupBoxSections(detailsTitle,
			[]string{renderNetworkInterfaceDetails(iface)}, innerWidth, colorBorder)
		blocks = append(blocks, "", detailsBox)
	}

	if dnsBox, ok := renderNetworkDNSBox(s.Network.DNS, innerWidth); ok {
		blocks = append(blocks, "", dnsBox)
	}

	if routesBox, ok := renderNetworkRoutesBox(s.Network.Routes, innerWidth); ok {
		blocks = append(blocks, "", routesBox)
	}

	blocks = append(blocks, renderNetworkFocusHint(focused, len(s.Network.Interfaces)))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return contentStyle.Width(width).Height(height).Render(body)
}

// renderNetworkInterfacesTable builds the main interfaces table.
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

// renderNetworkInterfaceDetails is the label/value grid for the
// currently highlighted interface.
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
	labelWidth := 0
	for _, r := range rows {
		if w := lipgloss.Width(r[0]); w > labelWidth {
			labelWidth = w
		}
	}
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		label := detailLabelStyle.Width(labelWidth + 2).Render(r[0])
		value := detailValueStyle.Render(orDash(r[1]))
		lines = append(lines, label+value)
	}
	return strings.Join(lines, "\n")
}

func renderNetworkDNSBox(dns network.DNS, total int) (string, bool) {
	if len(dns.Servers) == 0 && len(dns.Search) == 0 {
		return "", false
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

	labelWidth := 0
	for _, r := range rows {
		if w := lipgloss.Width(r[0]); w > labelWidth {
			labelWidth = w
		}
	}
	lines := make([]string, 0, len(rows))
	for _, r := range rows {
		label := detailLabelStyle.Width(labelWidth + 2).Render(r[0])
		value := detailValueStyle.Render(orDash(r[1]))
		lines = append(lines, label+value)
	}
	return groupBoxSections("DNS", []string{strings.Join(lines, "\n")}, total, colorBorder), true
}

func renderNetworkRoutesBox(routes []network.Route, total int) (string, bool) {
	if len(routes) == 0 {
		return "", false
	}
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
	return groupBoxSections("Routes", []string{strings.Join(lines, "\n")}, total, colorBorder), true
}

func renderNetworkFocusHint(focused bool, ifaceCount int) string {
	if ifaceCount == 0 {
		return ""
	}
	var text string
	if focused {
		text = "j/k select interface · esc back · (read-only — Tier 3 will add config)"
	} else {
		text = "enter to select an interface"
	}
	return statusBarStyle.Render(text)
}

// --- formatters ---

func formatInterfaceState(iface network.Interface) string {
	if iface.State == "" {
		return "—"
	}
	if iface.State == "up" && !iface.Carrier {
		return "up · no link"
	}
	return iface.State
}

// primaryAddress returns the first global-scope address for the
// interface, falling back to whatever's available, or "—" when none.
func primaryAddress(addrs []network.Address) string {
	for _, a := range addrs {
		if a.Scope == "global" {
			return a.Address
		}
	}
	if len(addrs) > 0 {
		return addrs[0].Address
	}
	return "—"
}

// addressList returns the formatted CIDR strings for an address slice.
func addressList(addrs []network.Address) []string {
	if len(addrs) == 0 {
		return []string{"—"}
	}
	out := make([]string, 0, len(addrs))
	for _, a := range addrs {
		if a.Scope != "" && a.Scope != "global" {
			out = append(out, a.CIDR+" ("+a.Scope+")")
		} else {
			out = append(out, a.CIDR)
		}
	}
	return out
}

func formatLinkSpeed(mbps int) string {
	if mbps <= 0 {
		return "—"
	}
	if mbps >= 1000 {
		return fmt.Sprintf("%.1f Gbps", float64(mbps)/1000.0)
	}
	return fmt.Sprintf("%d Mbps", mbps)
}

func formatCarrier(carrier bool, mbps int, duplex string) string {
	if !carrier {
		return placeholderStyle.Render("no link")
	}
	speed := formatLinkSpeed(mbps)
	if duplex != "" && duplex != "unknown" {
		return fmt.Sprintf("%s  %s-duplex", speed, duplex)
	}
	return speed
}

func formatMTU(mtu int) string {
	if mtu <= 0 {
		return "—"
	}
	return fmt.Sprintf("%d", mtu)
}

func formatPackets(n uint64) string {
	if n == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", n)
}

// formatNetworkBytes mirrors the wifi formatBytes function.
func formatNetworkBytes(b uint64) string {
	if b == 0 {
		return "—"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

func formatNetworkRate(rx, tx uint64) string {
	if rx == 0 && tx == 0 {
		return "—"
	}
	return fmt.Sprintf("%s ↓  %s ↑", formatBitsPerSec(rx), formatBitsPerSec(tx))
}
