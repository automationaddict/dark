package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/bluetooth"
)

func renderBluetooth(s *core.State, width, height int) string {
	if !s.BluetoothLoaded {
		return contentStyle.Width(width).Height(height).Render(
			placeholderStyle.Render("loading bluetooth adapters…"),
		)
	}
	adapters := s.Bluetooth.Adapters
	if len(adapters) == 0 {
		title := contentTitle.Render("Bluetooth")
		body := placeholderStyle.Render("No bluetooth adapters detected.")
		return contentStyle.Width(width).Height(height).Render(
			lipgloss.JoinVertical(lipgloss.Left, title, body),
		)
	}

	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	selected := s.BluetoothSelected
	if selected >= len(adapters) {
		selected = 0
	}
	focused := s.ContentFocused
	detailsOpen := s.BluetoothDetailsOpen
	selAdapter := adapters[selected]

	toggle := renderBluetoothToggle(selAdapter, s.BluetoothBusy)

	adaptersBorder := colorBorder
	if focused && s.BluetoothFocus == core.BluetoothFocusAdapters {
		adaptersBorder = colorAccent
	}
	adaptersBox := groupBoxSections("Adapters",
		[]string{renderBluetoothAdaptersTable(adapters, selected, focused)},
		innerWidth, adaptersBorder)

	blocks := []string{toggle, "", adaptersBox}

	if detailsOpen {
		detailsTitle := "Details"
		if selAdapter.Name != "" {
			detailsTitle = "Details · " + selAdapter.Name
		}
		detailsBox := groupBoxSections(detailsTitle,
			[]string{renderBluetoothAdapterDetails(selAdapter)}, innerWidth, colorBorder)
		blocks = append(blocks, "", detailsBox)

		devicesBorder := colorBorder
		if focused && s.BluetoothFocus == core.BluetoothFocusDevices {
			devicesBorder = colorAccent
		}
		if s.BluetoothDeviceInfoOpen {
			dev, ok := s.SelectedBluetoothDevice()
			if ok {
				infoBox := renderBluetoothDeviceInfoBox(dev, innerWidth)
				blocks = append(blocks, "", infoBox)
			} else {
				devicesBox := renderBluetoothDevicesBox(s, selAdapter, innerWidth, devicesBorder)
				blocks = append(blocks, "", devicesBox)
			}
		} else {
			devicesBox := renderBluetoothDevicesBox(s, selAdapter, innerWidth, devicesBorder)
			blocks = append(blocks, "", devicesBox)
		}
	}

	blocks = append(blocks, renderBluetoothFocusHint(s, focused, detailsOpen, len(adapters)))
	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)

	return contentStyle.Width(width).Height(height).Render(body)
}

func renderBluetoothToggle(a bluetooth.Adapter, busy bool) string {
	icon := "󰂯"
	stateText := "On"
	stateStyle := statusOnlineStyle
	if !a.Powered {
		icon = "󰂲"
		stateText = "Off"
		stateStyle = statusOfflineStyle
	}
	if busy {
		stateText = "…"
		stateStyle = statusBusyStyle
	}
	label := tableHeaderStyle.Render(icon + "  Bluetooth")
	state := stateStyle.Render(stateText)
	hint := placeholderStyle.Render("w to toggle")
	return label + "  " + state + "    " + hint
}

type btAdapterColumn struct {
	header string
	value  func(bluetooth.Adapter) string
	accent func(bluetooth.Adapter) bool
}

func bluetoothAdapterColumns() []btAdapterColumn {
	return []btAdapterColumn{
		{"Name", func(a bluetooth.Adapter) string { return a.Name }, nil},
		{"Alias", func(a bluetooth.Adapter) string { return orDash(a.Alias) }, nil},
		{"Address", func(a bluetooth.Adapter) string { return orDash(a.Address) }, nil},
		{"Powered", func(a bluetooth.Adapter) string { return onOff(a.Powered) }, func(a bluetooth.Adapter) bool { return a.Powered }},
		{"Discoverable", func(a bluetooth.Adapter) string { return yesNo(a.Discoverable) }, nil},
		{"Scanning", func(a bluetooth.Adapter) string { return yesNo(a.Discovering) }, func(a bluetooth.Adapter) bool { return a.Discovering }},
	}
}

func renderBluetoothAdaptersTable(adapters []bluetooth.Adapter, selected int, focused bool) string {
	cols := bluetoothAdapterColumns()
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, a := range adapters {
		for i, c := range cols {
			if w := lipgloss.Width(c.value(a)); w > widths[i] {
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

	for i, a := range adapters {
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
			text := c.value(a)
			var style lipgloss.Style
			switch {
			case isSel:
				style = tableCellSelected
			case c.accent != nil && c.accent(a):
				style = tableCellAccent
			default:
				style = tableCellStyle
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		lines = append(lines, marker+strings.Join(cells, gap))
	}
	return strings.Join(lines, "\n")
}

func renderBluetoothAdapterDetails(a bluetooth.Adapter) string {
	rows := [][2]string{
		{"Name", orDash(a.Name)},
		{"Alias", orDash(a.Alias)},
		{"Address", orDash(a.Address)},
		{"Powered", onOff(a.Powered)},
		{"Discoverable", yesNo(a.Discoverable)},
		{"Discoverable Timeout", formatBluetoothTimeout(a.DiscoverableTimeout)},
		{"Pairable", yesNo(a.Pairable)},
		{"Discovering", yesNo(a.Discovering)},
		{"Devices", fmt.Sprintf("%d", len(a.Devices))},
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

func renderBluetoothDevicesBox(s *core.State, a bluetooth.Adapter, total int, border lipgloss.Color) string {
	title := "Devices"
	if a.Name != "" {
		title = "Devices · " + a.Name
	}

	// Errors fire desktop notifications instead of rendering inline.
	var statusLine string
	switch {
	case s.BluetoothBusy:
		statusLine = statusBusyStyle.Render("working…")
	case a.Discovering:
		statusLine = statusBusyStyle.Render("scanning…")
	}

	sel := -1
	if s.ContentFocused && s.BluetoothDetailsOpen {
		sel = s.BluetoothDevSelected
	}

	if len(a.Devices) == 0 {
		body := placeholderStyle.Render("no devices — press s to start discovery")
		if statusLine != "" {
			body = statusLine + "\n" + body
		}
		return groupBoxSections(title, []string{body}, total, border)
	}

	table := renderBluetoothDevicesTable(a.Devices, sel)
	body := table
	if statusLine != "" {
		body = statusLine + "\n" + table
	}
	return groupBoxSections(title, []string{body}, total, border)
}

// renderBluetoothDeviceInfoBox is the second-level drill: full property
// readout for one device. Replaces the Devices table in the layout so
// the user sees everything BlueZ knows without fighting for vertical
// space. Press esc to back out to the list.
func renderBluetoothDeviceInfoBox(d bluetooth.Device, total int) string {
	title := "Device Info"
	if name := d.DisplayName(); name != "" {
		title = "Device Info · " + name
	}

	rows := [][2]string{
		{"Name", orDash(d.Name)},
		{"Alias", orDash(d.Alias)},
		{"Address", formatBluetoothAddress(d.Address, d.AddressType)},
		{"Type", formatBluetoothIcon(d.Icon)},
		{"Class", formatBluetoothClass(d.Class)},
		{"Modalias", orDash(d.Modalias)},
		{"Paired", yesNo(d.Paired)},
		{"Bonded", yesNo(d.Bonded)},
		{"Trusted", yesNo(d.Trusted)},
		{"Blocked", yesNo(d.Blocked)},
		{"Connected", yesNo(d.Connected)},
		{"LegacyPairing", yesNo(d.LegacyPairing)},
		{"Services Resolved", yesNo(d.ServicesResolved)},
		{"RSSI", formatBluetoothRSSI(d.RSSI)},
		{"Tx Power", formatBluetoothTxPower(d.TxPower)},
		{"Battery", formatBluetoothBattery(d.Battery)},
	}

	labelWidth := 0
	for _, r := range rows {
		if w := lipgloss.Width(r[0]); w > labelWidth {
			labelWidth = w
		}
	}

	propLines := make([]string, 0, len(rows))
	for _, r := range rows {
		label := detailLabelStyle.Width(labelWidth + 2).Render(r[0])
		value := detailValueStyle.Render(orDash(r[1]))
		propLines = append(propLines, label+value)
	}
	properties := strings.Join(propLines, "\n")

	sections := []string{properties}
	if len(d.UUIDs) > 0 {
		sections = append(sections, renderBluetoothUUIDList(d.UUIDs))
	}
	sections = append(sections, placeholderStyle.Render("esc to return to Devices"))

	return groupBoxSections(title, sections, total, colorAccent)
}

// renderBluetoothUUIDList renders the device's advertised service UUIDs
// with a friendly name where dark knows one. Unknown UUIDs fall through
// to their raw hex form so nothing is silently dropped.
func renderBluetoothUUIDList(uuids []string) string {
	lines := []string{tableHeaderStyle.Render("Services")}
	for _, u := range uuids {
		name := bluetooth.LookupUUIDName(u)
		if name == "" {
			lines = append(lines, detailValueStyle.Render(u))
			continue
		}
		label := tableCellAccent.Render(name)
		muted := detailLabelStyle.Render("(" + u + ")")
		lines = append(lines, label+"  "+muted)
	}
	return strings.Join(lines, "\n")
}

// renderBluetoothDevicesTable builds the device list. Columns cover the
// core Tier 1 surface: name, address, type, and the state flags that
// drive the action keys (Paired, Trusted, Connected, RSSI, Battery).
func renderBluetoothDevicesTable(devs []bluetooth.Device, selected int) string {
	type col struct {
		header string
		cell   func(bluetooth.Device) string
	}
	cols := []col{
		{"Name", func(d bluetooth.Device) string { return orDash(d.DisplayName()) }},
		{"Address", func(d bluetooth.Device) string { return orDash(d.Address) }},
		{"Type", func(d bluetooth.Device) string { return formatBluetoothIcon(d.Icon) }},
		{"Paired", func(d bluetooth.Device) string { return yesNo(d.Paired) }},
		{"Trusted", func(d bluetooth.Device) string { return yesNo(d.Trusted) }},
		{"Blocked", func(d bluetooth.Device) string { return yesNo(d.Blocked) }},
		{"RSSI", func(d bluetooth.Device) string { return formatBluetoothRSSI(d.RSSI) }},
		{"Battery", func(d bluetooth.Device) string { return formatBluetoothBattery(d.Battery) }},
	}

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, d := range devs {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(d)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = "  "
	lines := make([]string, 0, len(devs)+1)

	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines = append(lines, "   "+strings.Join(headerCells, gap))

	for i, d := range devs {
		isSel := selected >= 0 && i == selected
		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.cell(d)
			style := tableCellStyle
			switch {
			case isSel:
				style = tableCellSelected
			case d.Connected:
				style = tableCellAccent
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		lines = append(lines, bluetoothRowMarker(isSel, d.Connected)+strings.Join(cells, gap))
	}
	return strings.Join(lines, "\n")
}

// bluetoothRowMarker is the 3-cell prefix: selection marker, status
// glyph (currently-connected indicator), trailing space. Matches the
// wifi row marker shape.
func bluetoothRowMarker(selected, connected bool) string {
	sel := " "
	if selected {
		sel = tableSelectionMarker.Render("▸")
	}
	status := " "
	if connected {
		status = tableSelectionMarker.Render("󰂱")
	}
	return sel + status + " "
}

func renderBluetoothFocusHint(s *core.State, focused, detailsOpen bool, adapterCount int) string {
	if adapterCount == 0 {
		return ""
	}
	var text string
	switch {
	case focused && s.BluetoothDeviceInfoOpen:
		text = "esc back · c connect · d disconnect · t trust · b block · u unpair · w toggle"
	case focused && s.BluetoothFocus == core.BluetoothFocusAdapters:
		text = "tab · j/k select adapter · s scan · y/a disc/pair · r/R rename/reset · T timeout · F filter · w toggle · esc"
	case focused:
		text = "tab · j/k · enter info · c/d · p/x pair/cancel · u unpair · t trust · b block · s scan · y/a · r/R · w · esc"
	default:
		text = "enter · w toggle · s scan · y/a disc/pair · r/R rename/reset · T timeout"
	}
	return statusBarStyle.Render(text)
}

// formatBluetoothIcon maps BlueZ's icon hint strings to a short display
// label. BlueZ uses freedesktop icon names like "audio-headset".
func formatBluetoothIcon(icon string) string {
	switch icon {
	case "":
		return "—"
	case "audio-headset", "audio-headphones":
		return "headset"
	case "audio-card":
		return "audio"
	case "input-keyboard":
		return "keyboard"
	case "input-mouse":
		return "mouse"
	case "input-gaming":
		return "gamepad"
	case "input-tablet":
		return "tablet"
	case "phone":
		return "phone"
	case "computer":
		return "computer"
	case "camera-video", "camera-photo":
		return "camera"
	case "printer":
		return "printer"
	case "network-wireless":
		return "network"
	default:
		return icon
	}
}

func formatBluetoothRSSI(rssi int16) string {
	if rssi == 0 {
		return "—"
	}
	return fmt.Sprintf("%d dBm", rssi)
}

// formatBluetoothTimeout renders a BlueZ timeout value in seconds as a
// human-friendly string. Zero is "never" — BlueZ treats it as "no
// timeout, stay in this state until explicitly toggled".
func formatBluetoothTimeout(seconds uint32) string {
	if seconds == 0 {
		return "never"
	}
	switch {
	case seconds < 60:
		return fmt.Sprintf("%ds", seconds)
	case seconds < 3600:
		return fmt.Sprintf("%dm %ds", seconds/60, seconds%60)
	default:
		h := seconds / 3600
		m := (seconds % 3600) / 60
		return fmt.Sprintf("%dh %dm", h, m)
	}
}

func formatBluetoothTxPower(tx int16) string {
	if tx == 0 {
		return "—"
	}
	return fmt.Sprintf("%d dBm", tx)
}

// formatBluetoothAddress joins a MAC with its address type (public or
// random) when BlueZ reports one. LE devices commonly expose a random
// resolvable address so the type is useful debugging context.
func formatBluetoothAddress(addr, addrType string) string {
	if addr == "" {
		return "—"
	}
	if addrType == "" {
		return addr
	}
	return addr + " (" + addrType + ")"
}

// formatBluetoothClass renders the Class of Device as hex plus the
// decoded major class name.
func formatBluetoothClass(class uint32) string {
	if class == 0 {
		return "—"
	}
	major := bluetooth.MajorClassFromClass(class)
	if major == "" {
		return fmt.Sprintf("0x%06x", class)
	}
	return fmt.Sprintf("0x%06x  (%s)", class, major)
}

func formatBluetoothBattery(pct int8) string {
	if pct < 0 {
		return "—"
	}
	return fmt.Sprintf("%d%%", pct)
}
