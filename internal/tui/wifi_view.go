package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/wifi"
)

type wifiColumn struct {
	header string
	value  func(wifi.Adapter) string
	accent func(wifi.Adapter) bool
}

func wifiColumns() []wifiColumn {
	return []wifiColumn{
		{"Name", func(a wifi.Adapter) string { return a.Name }, nil},
		{"Mode", func(a wifi.Adapter) string { return orDash(a.Mode) }, nil},
		{"Powered", func(a wifi.Adapter) string { return onOff(a.Powered) }, func(a wifi.Adapter) bool { return a.Powered }},
		{"State", func(a wifi.Adapter) string { return titleCase(orDash(a.State)) }, func(a wifi.Adapter) bool { return a.State == "connected" }},
		{"Scanning", func(a wifi.Adapter) string { return yesNo(a.Scanning) }, nil},
		{"Frequency", func(a wifi.Adapter) string { return formatFreq(a.FrequencyMHz) }, nil},
		{"Security", func(a wifi.Adapter) string { return orDash(a.Security) }, nil},
	}
}

func renderWifi(s *core.State, width, height int) string {
	if !s.WifiLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("loading wireless adapters…"),
		)
	}
	adapters := s.Wifi.Adapters
	if len(adapters) == 0 {
		title := contentTitle.Render("Wi-Fi")
		body := placeholderStyle.Render("No wireless adapters detected.")
		return renderContentPane(width, height,
			lipgloss.JoinVertical(lipgloss.Left, title, body),
		)
	}

	// contentStyle has Padding(1, 3), so usable inner width is width - 6.
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	selected := s.WifiSelected
	if selected >= len(adapters) {
		selected = 0
	}
	focused := s.ContentFocused

	tableSection := renderAdaptersTable(adapters, selected, focused)

	adaptersBorder := colorBorder
	if focused && s.WifiFocus == core.WifiFocusAdapters {
		adaptersBorder = colorAccent
	}
	adaptersBox := groupBoxSections("Adapters", []string{tableSection}, innerWidth, adaptersBorder)

	selAdapter := adapters[selected]
	toggleLine := renderWifiToggle(selAdapter, s.WifiBusy)
	blocks := []string{toggleLine, "", adaptersBox}

	detailsTitle := "Details"
	if selAdapter.Name != "" {
		detailsTitle = "Details · " + selAdapter.Name
	}
	detailsBox := groupBoxSections(detailsTitle,
		[]string{renderAdapterDetailsInline(selAdapter, s.RSSIHistory[selAdapter.Name])}, innerWidth, colorBorder)
	blocks = append(blocks, "", detailsBox)

	networksBorder := colorBorder
	if focused && s.WifiFocus == core.WifiFocusNetworks {
		networksBorder = colorAccent
	}
	networksBox := renderNetworksBox(s, selAdapter, innerWidth, networksBorder)
	blocks = append(blocks, "", networksBox)

	knownBorder := colorBorder
	if focused && s.WifiFocus == core.WifiFocusKnown {
		knownBorder = colorAccent
	}
	knownBox := renderKnownNetworksBox(s, innerWidth, knownBorder)
	blocks = append(blocks, "", knownBox)

	if adapterSupportsAP(adapters[selected]) {
		apBox := renderAccessPointBox(adapters[selected], innerWidth)
		blocks = append(blocks, "", apBox)
	}

	blocks = append(blocks, renderWifiFocusHint(s, focused, true, len(adapters)))
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)

	return renderContentPane(width, height, body)
}

// renderAdaptersTable builds the header + data rows for the Adapters table.
// The selected row gets a leading ▸ marker and accent coloring on all
// cells. When the content region is unfocused, the marker stays but is
// rendered in the dim color so it reads as "selected but not active".
func renderAdaptersTable(adapters []wifi.Adapter, selected int, focused bool) string {
	cols := wifiColumns()
	colW := make([]int, len(cols))
	for i, c := range cols {
		colW[i] = lipgloss.Width(c.header)
	}
	for _, a := range adapters {
		for i, c := range cols {
			if w := lipgloss.Width(c.value(a)); w > colW[i] {
				colW[i] = w
			}
		}
	}

	var lines []string
	lines = append(lines, buildHeaderRow(cols, colW))
	for i, a := range adapters {
		isSel := i == selected
		lines = append(lines, buildDataRow(cols, colW, a, isSel, focused))
	}
	return strings.Join(lines, "\n")
}

func buildHeaderRow(cols []wifiColumn, widths []int) string {
	const gap = "  "
	cells := make([]string, 0, len(cols))
	for i, c := range cols {
		cells = append(cells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	// Leading "  " aligns the header with the row selection-marker column.
	return "  " + strings.Join(cells, gap)
}

func buildDataRow(cols []wifiColumn, widths []int, a wifi.Adapter, selected, focused bool) string {
	const gap = "  "
	var marker string
	switch {
	case selected && focused:
		marker = tableSelectionMarker.Render("▸ ")
	case selected:
		marker = tableSelectionMarkerDim.Render("▸ ")
	default:
		marker = "  "
	}

	parts := make([]string, 0, len(cols))
	for i, c := range cols {
		text := c.value(a)
		var style lipgloss.Style
		switch {
		case selected:
			style = tableCellSelected
		case c.accent != nil && c.accent(a):
			style = tableCellAccent
		default:
			style = tableCellStyle
		}
		parts = append(parts, style.Width(widths[i]).Render(text))
	}
	return marker + strings.Join(parts, gap)
}

// renderWifiToggle draws a single line above the Adapters box showing
// the current radio power state with a keyboard hint. It's not wrapped
// in a group box because it's an overview indicator, not a navigable
// table — the visual weight of a full border would make it look more
// important than the Adapters box beneath it.
func renderWifiToggle(a wifi.Adapter, busy bool) string {
	icon := "󰖩"
	stateText := "On"
	stateStyle := statusOnlineStyle
	if !a.Powered {
		icon = "󰖪"
		stateText = "Off"
		stateStyle = statusOfflineStyle
	}
	if busy {
		stateText = "…"
		stateStyle = statusBusyStyle
	}
	label := tableHeaderStyle.Render(icon + "  Wi-Fi")
	state := stateStyle.Render(stateText)
	hint := placeholderStyle.Render("w to toggle")
	return label + "  " + state + "    " + hint
}

// renderNetworksBox shows the scanned SSIDs for an adapter. Border color
// is caller-controlled so the outer view can light it up when this
// sub-table owns focus.
func renderNetworksBox(s *core.State, a wifi.Adapter, total int, border lipgloss.Color) string {
	title := "Networks"
	if a.Name != "" {
		title = "Networks · " + a.Name
	}

	// Errors no longer render inline — they fire desktop notifications
	// via the model's notifyError helper. Only the busy/scanning
	// indicators stay here because they're transient state, not failures.
	var statusLine string
	switch {
	case s.WifiBusy:
		statusLine = statusBusyStyle.Render("working…")
	case s.WifiScanning:
		statusLine = statusBusyStyle.Render("scanning…")
	}

	focused := s.WifiFocus == core.WifiFocusNetworks
	sel := -1
	if focused {
		sel = s.WifiNetworkSelected
	}

	if len(a.Networks) == 0 {
		body := placeholderStyle.Render("no networks — press s to scan")
		if statusLine != "" {
			body = statusLine + "\n" + body
		}
		return groupBoxSections(title, []string{body}, total, border)
	}

	table := renderNetworksTable(a.Networks, sel)
	body := table
	if statusLine != "" {
		body = statusLine + "\n" + table
	}
	return groupBoxSections(title, []string{body}, total, border)
}

// adapterSupportsAP checks whether the iwd SupportedModes list on
// the selected adapter includes "ap". When it doesn't, the Access
// Point group box is hidden from the view entirely — there's no
// useful interaction to offer.
func adapterSupportsAP(a wifi.Adapter) bool {
	for _, m := range a.SupportedModes {
		if m == "ap" {
			return true
		}
	}
	return false
}

// renderAccessPointBox shows the current AP state and the key hints.
// When no AP is running it's a single-line "stopped" message with a
// reminder to press p. When one is running it's a label/value grid
// with SSID and frequency.
func renderAccessPointBox(a wifi.Adapter, total int) string {
	title := "Access Point · " + a.Name

	if !a.APActive {
		body := placeholderStyle.Render("not running — press p to start a hotspot")
		return groupBoxSections(title, []string{body}, total, colorBorder)
	}

	rows := [][2]string{
		{"Status", statusOnlineStyle.Render("Running")},
		{"SSID", orDash(a.APSSID)},
		{"Frequency", formatFreq(a.APFrequencyMHz)},
		{"Mode", orDash(a.Mode)},
	}
	labelWidth := 0
	for _, r := range rows {
		if w := lipgloss.Width(r[0]); w > labelWidth {
			labelWidth = w
		}
	}
	lines := make([]string, 0, len(rows)+1)
	for _, r := range rows {
		label := detailLabelStyle.Width(labelWidth + 2).Render(r[0])
		lines = append(lines, label+r[1])
	}
	lines = append(lines, "")
	lines = append(lines, placeholderStyle.Render("press p to stop the hotspot"))
	return groupBoxSections(title, []string{strings.Join(lines, "\n")}, total, colorAccent)
}

// renderKnownNetworksBox renders the saved-profile list as a table.
func renderKnownNetworksBox(s *core.State, total int, border lipgloss.Color) string {
	title := "Known Networks"
	nets := s.Wifi.KnownNetworks
	if len(nets) == 0 {
		body := placeholderStyle.Render("no saved networks")
		return groupBoxSections(title, []string{body}, total, border)
	}

	focused := s.WifiFocus == core.WifiFocusKnown
	sel := -1
	if focused {
		sel = s.WifiKnownSelected
	}

	body := renderKnownNetworksTable(nets, sel)
	return groupBoxSections(title, []string{body}, total, border)
}

// renderNetworksTable builds the scanned-SSID table. The connected
// network gets a ★ prefix, the highlighted row gets a ▸ marker. When
// selected < 0 the selection marker is suppressed (used when this
// sub-table doesn't currently own focus).
func renderNetworksTable(nets []wifi.Network, selected int) string {
	type col struct {
		header string
		cell   func(wifi.Network) string
	}
	cols := []col{
		{"SSID", func(n wifi.Network) string { return orDash(n.SSID) }},
		{"Security", func(n wifi.Network) string { return formatNetSecurity(n.Security) }},
		{"Signal", func(n wifi.Network) string { return fmt.Sprintf("%d dBm", n.SignalDBm) }},
		{"Bars", func(n wifi.Network) string { return signalBars(n.SignalDBm) }},
		{"BSS", func(n wifi.Network) string { return fmt.Sprintf("%d", n.BSSCount) }},
	}

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, n := range nets {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(n)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = "  "
	lines := make([]string, 0, len(nets)+1)

	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines = append(lines, "   "+strings.Join(headerCells, gap))

	for i, n := range nets {
		cells := make([]string, 0, len(cols))
		isSel := selected >= 0 && i == selected
		for j, c := range cols {
			text := c.cell(n)
			style := tableCellStyle
			switch {
			case isSel:
				style = tableCellSelected
			case n.Connected:
				style = tableCellAccent
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		lines = append(lines, rowMarker(isSel, n.Connected)+strings.Join(cells, gap))
	}
	return strings.Join(lines, "\n")
}

// renderKnownNetworksTable builds the saved-profile table.
func renderKnownNetworksTable(nets []wifi.KnownNetwork, selected int) string {
	type col struct {
		header string
		cell   func(wifi.KnownNetwork) string
	}
	cols := []col{
		{"SSID", func(k wifi.KnownNetwork) string { return orDash(k.SSID) }},
		{"Security", func(k wifi.KnownNetwork) string { return formatNetSecurity(k.Security) }},
		{"Auto", func(k wifi.KnownNetwork) string { return yesNo(k.AutoConnect) }},
		{"Hidden", func(k wifi.KnownNetwork) string { return yesNo(k.Hidden) }},
		{"Last seen", func(k wifi.KnownNetwork) string { return formatAge(k.LastConnectedTime) }},
	}

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, n := range nets {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(n)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = "  "
	lines := make([]string, 0, len(nets)+1)

	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines = append(lines, "   "+strings.Join(headerCells, gap))

	for i, k := range nets {
		isSel := selected >= 0 && i == selected
		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.cell(k)
			style := tableCellStyle
			if isSel {
				style = tableCellSelected
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		lines = append(lines, rowMarker(isSel, false)+strings.Join(cells, gap))
	}
	return strings.Join(lines, "\n")
}

// rowMarker builds a 3-cell prefix: selection marker · status glyph ·
// trailing space. The status glyph is the Nerd Font Wi-Fi icon when the
// row represents the currently associated network, matching the icon
// used for the Wi-Fi section in the sidebar.
func rowMarker(selected, connected bool) string {
	sel := " "
	if selected {
		sel = tableSelectionMarker.Render("▸")
	}
	status := " "
	if connected {
		status = tableSelectionMarker.Render("󰖩")
	}
	return sel + status + " "
}

// formatAge renders an RFC3339 timestamp as a short "time ago" string.
// iwd reports LastConnectedTime as RFC3339 UTC. Unset / unparseable
// values return a dash.
func formatAge(rfc3339 string) string {
	if rfc3339 == "" {
		return "—"
	}
	t, err := time.Parse(time.RFC3339, rfc3339)
	if err != nil {
		return rfc3339
	}
	d := time.Since(t)
	switch {
	case d < 0:
		return "just now"
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// formatNetSecurity maps iwd's network type to a display label.
func formatNetSecurity(t string) string {
	switch t {
	case "open":
		return "Open"
	case "psk":
		return "WPA2/3"
	case "8021x":
		return "Enterprise"
	case "wep":
		return "WEP"
	case "":
		return "—"
	default:
		return t
	}
}

// signalBars returns a Unicode bar-graph glyph approximating signal
// strength from an RSSI value in dBm.
func signalBars(dbm int) string {
	switch {
	case dbm == 0:
		return "—"
	case dbm >= -50:
		return "▁▂▃▄▅▆▇█"
	case dbm >= -60:
		return "▁▂▃▄▅▆▇"
	case dbm >= -67:
		return "▁▂▃▄▅▆"
	case dbm >= -70:
		return "▁▂▃▄▅"
	case dbm >= -75:
		return "▁▂▃▄"
	case dbm >= -80:
		return "▁▂▃"
	case dbm >= -85:
		return "▁▂"
	default:
		return "▁"
	}
}

// renderAdapterDetailsInline is the details section for the currently
// selected row. Same data as before, presented as a label/value grid,
// but rendered without its own borders because the outer group box
// already provides them. history carries the recent RSSI samples for
// the sparkline row — an empty slice is fine.
func renderAdapterDetailsInline(a wifi.Adapter, history []int16) string {
	rows := [][2]string{
		{"SSID", orDash(a.SSID)},
		{"BSSID", orDash(a.BSSID)},
		{"MAC", orDash(a.MAC)},
		{"Driver", orDash(a.Driver)},
		{"Vendor", orDash(a.Vendor)},
		{"Model", orDash(a.Model)},
		{"IPv4", orDash(a.IPv4)},
		{"IPv6", orDash(a.IPv6)},
		{"Gateway", orDash(a.Gateway)},
		{"DNS", strings.Join(a.DNS, ", ")},
		{"Channel", formatChannel(a.Channel)},
		{"Signal", formatSignal(a.RSSI, a.AverageRSSI, history)},
		{"Link", formatLink(a.RxMode, a.RxBitrateKbps, a.TxMode, a.TxBitrateKbps)},
		{"Traffic", formatTraffic(a.RxBytes, a.TxBytes)},
		{"Rate", formatRate(a.RxRateBps, a.TxRateBps)},
		{"Connected", formatDuration(a.ConnectedSecs)},
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

// renderWifiFocusHint shows one line beneath the box reminding the user
// how to move focus in and out of the content region. Stays out of the
// global status bar so the wifi-specific shortcut is only visible here.
func renderWifiFocusHint(s *core.State, focused, detailsOpen bool, adapterCount int) string {
	if adapterCount == 0 {
		return ""
	}
	var text string
	switch {
	case focused && s.WifiFocus == core.WifiFocusKnown:
		text = "tab · j/k · c connect · f forget · a auto · d disconnect · h hidden · p AP · w toggle · esc"
	case focused && s.WifiFocus == core.WifiFocusAdapters:
		text = "tab · j/k select adapter · s scan · h hidden · p AP · w toggle · esc"
	case focused:
		text = "tab · j/k · c connect · s scan · h hidden · d disconnect · p AP · w toggle · esc"
	default:
		text = "enter · w toggle radio · h hidden · p start/stop hotspot"
	}
	return statusBarStyle.Render(text)
}

// --- formatters ---

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

func onOff(v bool) string {
	if v {
		return "On"
	}
	return "Off"
}

func yesNo(v bool) string {
	if v {
		return "Yes"
	}
	return "No"
}

func titleCase(s string) string {
	if s == "" || s == "—" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func formatFreq(mhz uint32) string {
	if mhz == 0 {
		return "—"
	}
	return fmt.Sprintf("%.2f GHz", float64(mhz)/1000.0)
}

func formatChannel(ch uint16) string {
	if ch == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", ch)
}

func formatRSSI(current, average int16) string {
	if current == 0 {
		return "—"
	}
	if average != 0 && average != current {
		return fmt.Sprintf("%d dBm  (avg %d)", current, average)
	}
	return fmt.Sprintf("%d dBm", current)
}

// formatSignal combines the RSSI text with a rolling sparkline of
// recent samples. With fewer than two samples the sparkline is
// omitted — a single bar doesn't tell the user anything useful.
func formatSignal(current, average int16, history []int16) string {
	text := formatRSSI(current, average)
	if len(history) < 2 {
		return text
	}
	return text + "  " + rssiSparkline(history)
}

// rssiSparkline maps a history of RSSI values (in dBm, always negative)
// to a Unicode bar-graph. The scale is pinned to -30 (excellent) and
// -90 (unusable) so changes across sessions are visually comparable
// rather than normalized to whatever's currently in the buffer.
func rssiSparkline(history []int16) string {
	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	const best = -30
	const worst = -90
	const span = best - worst // 60

	var b strings.Builder
	for _, v := range history {
		if v == 0 {
			b.WriteRune(' ')
			continue
		}
		n := int(v)
		if n > best {
			n = best
		}
		if n < worst {
			n = worst
		}
		idx := (n - worst) * (len(bars) - 1) / span
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		b.WriteRune(bars[idx])
	}
	return b.String()
}

func formatLink(rxMode string, rxKbps uint32, txMode string, txKbps uint32) string {
	if rxKbps == 0 && txKbps == 0 {
		return "—"
	}
	mode := rxMode
	if mode == "" {
		mode = txMode
	}
	return fmt.Sprintf("%s  (TX %s · RX %s)", mode, formatBitrate(txKbps), formatBitrate(rxKbps))
}

// formatTraffic renders cumulative RX/TX byte totals as a short string.
func formatTraffic(rxBytes, txBytes uint64) string {
	if rxBytes == 0 && txBytes == 0 {
		return "—"
	}
	return fmt.Sprintf("%s ↓  %s ↑", formatBytes(rxBytes), formatBytes(txBytes))
}

// formatRate renders RX/TX rates in human-friendly units.
func formatRate(rxBps, txBps uint64) string {
	if rxBps == 0 && txBps == 0 {
		return "—"
	}
	return fmt.Sprintf("%s ↓  %s ↑", formatBitsPerSec(rxBps), formatBitsPerSec(txBps))
}

// formatBytes renders a byte count as KiB/MiB/GiB/TiB.
func formatBytes(b uint64) string {
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

// formatBitsPerSec renders a byte-per-second rate as bits per second in
// kb/s or Mb/s. Network rates are conventionally reported in bits.
func formatBitsPerSec(bytesPerSec uint64) string {
	bitsPerSec := bytesPerSec * 8
	switch {
	case bitsPerSec == 0:
		return "0 bps"
	case bitsPerSec < 1_000:
		return fmt.Sprintf("%d bps", bitsPerSec)
	case bitsPerSec < 1_000_000:
		return fmt.Sprintf("%.1f kbps", float64(bitsPerSec)/1_000)
	case bitsPerSec < 1_000_000_000:
		return fmt.Sprintf("%.1f Mbps", float64(bitsPerSec)/1_000_000)
	default:
		return fmt.Sprintf("%.2f Gbps", float64(bitsPerSec)/1_000_000_000)
	}
}

func formatBitrate(kbps uint32) string {
	if kbps == 0 {
		return "—"
	}
	if kbps >= 10000 {
		return fmt.Sprintf("%.1f Mbps", float64(kbps)/1000.0)
	}
	return fmt.Sprintf("%d kbps", kbps)
}

func formatDuration(secs uint32) string {
	if secs == 0 {
		return "—"
	}
	d := int(secs)
	h := d / 3600
	m := (d % 3600) / 60
	s := d % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}
