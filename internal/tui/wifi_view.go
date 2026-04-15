package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/wifi"
)

func wifiColumns() []accentColumn[wifi.Adapter] {
	return []accentColumn[wifi.Adapter]{
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

	secs := core.WifiSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.WifiContentFocused
	sidebar := renderInnerSidebarFocused(s, entries, s.WifiSectionIdx, height, sidebarFocused)
	contentWidth := width - lipgloss.Width(sidebar)

	sec := s.ActiveWifiSection()
	var content string
	switch sec.ID {
	case "adapters":
		content = renderWifiAdaptersSection(s, contentWidth, height)
	case "networks":
		content = renderWifiNetworksSection(s, contentWidth, height)
	case "known":
		content = renderWifiKnownSection(s, contentWidth, height)
	case "ap":
		content = renderWifiAPSection(s, contentWidth, height)
	default:
		content = renderContentPane(contentWidth, height,
			placeholderStyle.Render("Not implemented."))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderWifiAdaptersSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	adapters := s.Wifi.Adapters
	selected := s.WifiSelected
	if selected >= len(adapters) {
		selected = 0
	}

	tableSection := renderAdaptersTable(adapters, selected, s.WifiContentFocused)
	adaptersBox := groupBoxSections("Adapters", []string{tableSection}, innerWidth,
		borderForFocus(s.WifiContentFocused))

	selAdapter := adapters[selected]
	toggleLine := renderWifiToggle(selAdapter, s.WifiBusy)

	detailsTitle := "Details"
	if selAdapter.Name != "" {
		detailsTitle = "Details · " + selAdapter.Name
	}
	detailsBox := groupBoxSections(detailsTitle,
		[]string{renderAdapterDetailsInline(selAdapter, s.RSSIHistory[selAdapter.Name])},
		innerWidth, colorBorder)

	blocks := []string{toggleLine, "", adaptersBox, "", detailsBox}
	blocks = append(blocks, renderWifiFocusHint(s, s.WifiContentFocused, true, len(adapters)))
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderWifiNetworksSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	adapters := s.Wifi.Adapters
	selected := s.WifiSelected
	if selected >= len(adapters) {
		selected = 0
	}
	selAdapter := adapters[selected]

	networksBox := renderNetworksBox(s, selAdapter, innerWidth,
		borderForFocus(s.WifiContentFocused))

	blocks := []string{networksBox}
	blocks = append(blocks, renderWifiFocusHint(s, s.WifiContentFocused, true, len(adapters)))
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderWifiKnownSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	knownBox := renderKnownNetworksBox(s, innerWidth, borderForFocus(s.WifiContentFocused))

	blocks := []string{knownBox}
	blocks = append(blocks, renderWifiFocusHint(s, s.WifiContentFocused, true, len(s.Wifi.Adapters)))
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return renderContentPane(width, height, body)
}

func renderWifiAPSection(s *core.State, width, height int) string {
	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	adapters := s.Wifi.Adapters
	selected := s.WifiSelected
	if selected >= len(adapters) {
		selected = 0
	}

	if !adapterSupportsAP(adapters[selected]) {
		return renderContentPane(width, height,
			placeholderStyle.Render("Selected adapter does not support Access Point mode."))
	}

	apBox := renderAccessPointBox(adapters[selected], innerWidth)
	body := lipgloss.JoinVertical(lipgloss.Left, apBox)
	return renderContentPane(width, height, body)
}

// renderInnerSidebar renders a sub-section sidebar for sections that
// have their own navigation (wifi, bluetooth, power). Highlights when
// the content pane has focus.
func renderInnerSidebar(s *core.State, entries []sidebarEntry, selected int, height int) string {
	return renderInnerSidebarFocused(s, entries, selected, height, s.ContentFocused)
}

// renderInnerSidebarFocused is like renderInnerSidebar but takes an
// explicit focused flag for cases where the highlight depends on more
// than just ContentFocused (e.g. keybindings table focus).
func renderInnerSidebarFocused(s *core.State, entries []sidebarEntry, selected int, height int, focused bool) string {
	itemWidth := s.SidebarItemWidth
	item := sidebarItem.Width(itemWidth)
	active := sidebarItemActive.Width(itemWidth)

	var rows []string
	for i, e := range entries {
		line := e.Icon + "  " + e.Label
		if i == selected {
			rows = append(rows, active.Render(line))
		} else {
			rows = append(rows, item.Render(line))
		}
	}
	body := strings.Join(rows, "\n")
	return renderSidebarPane(height, body, focused)
}

func borderForFocus(focused bool) lipgloss.Color {
	if focused {
		return colorAccent
	}
	return colorBorder
}

// renderAdaptersTable builds the header + data rows for the Adapters table.
// The selected row gets a leading ▸ marker and accent coloring on all
// cells. When the content region is unfocused, the marker stays but is
// rendered in the dim color so it reads as "selected but not active".
func renderAdaptersTable(adapters []wifi.Adapter, selected int, focused bool) string {
	return renderAccentTable(wifiColumns(), adapters, selected, focused, nil, nil)
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

	sel := -1
	if s.WifiContentFocused {
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

	sel := -1
	if s.WifiContentFocused {
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
	cols := []accentColumn[wifi.Network]{
		{"SSID", func(n wifi.Network) string { return orDash(n.SSID) }, nil},
		{"Security", func(n wifi.Network) string { return formatNetSecurity(n.Security) }, nil},
		{"Signal", func(n wifi.Network) string { return fmt.Sprintf("%d dBm", n.SignalDBm) }, nil},
		{"Bars", func(n wifi.Network) string { return signalBars(n.SignalDBm) }, nil},
		{"BSS", func(n wifi.Network) string { return fmt.Sprintf("%d", n.BSSCount) }, nil},
	}
	marker := func(sel, _ bool, n wifi.Network) string { return rowMarker(sel, n.Connected) }
	return renderAccentTable(cols, nets, selected, true,
		func(n wifi.Network) bool { return n.Connected },
		marker)
}

// renderKnownNetworksTable builds the saved-profile table.
func renderKnownNetworksTable(nets []wifi.KnownNetwork, selected int) string {
	cols := []accentColumn[wifi.KnownNetwork]{
		{"SSID", func(k wifi.KnownNetwork) string { return orDash(k.SSID) }, nil},
		{"Security", func(k wifi.KnownNetwork) string { return formatNetSecurity(k.Security) }, nil},
		{"Auto", func(k wifi.KnownNetwork) string { return yesNo(k.AutoConnect) }, nil},
		{"Hidden", func(k wifi.KnownNetwork) string { return yesNo(k.Hidden) }, nil},
		{"Last seen", func(k wifi.KnownNetwork) string { return formatAge(k.LastConnectedTime) }, nil},
	}
	marker := func(sel, _ bool, _ wifi.KnownNetwork) string { return rowMarker(sel, false) }
	return renderAccentTable(cols, nets, selected, true, nil, marker)
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
	return renderDetailRows(rows)
}

// renderWifiFocusHint shows one line beneath the box reminding the user
// how to move focus in and out of the content region. Stays out of the
// global status bar so the wifi-specific shortcut is only visible here.
func renderWifiFocusHint(s *core.State, focused, detailsOpen bool, adapterCount int) string {
	if adapterCount == 0 {
		return ""
	}
	if !focused {
		return statusBarStyle.Render("enter to select · w toggle radio")
	}
	sec := s.ActiveWifiSection()
	var text string
	switch sec.ID {
	case "adapters":
		text = "j/k select adapter · s scan · w toggle · esc"
	case "networks":
		text = "j/k · c connect · s scan · d disconnect · h hidden · esc"
	case "known":
		text = "j/k · c connect · f forget · a autoconnect · esc"
	case "ap":
		text = "p start/stop hotspot · esc"
	default:
		text = "esc"
	}
	return statusBarStyle.Render(text)
}

