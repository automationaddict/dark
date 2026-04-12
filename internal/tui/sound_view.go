package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/audio"
)

func renderSound(s *core.State, width, height int) string {
	if !s.AudioLoaded {
		return contentStyle.Width(width).Height(height).Render(
			placeholderStyle.Render("loading audio devices…"),
		)
	}
	if len(s.Audio.Sinks) == 0 && len(s.Audio.Sources) == 0 {
		title := contentTitle.Render("Sound")
		body := placeholderStyle.Render("No audio devices detected.")
		return contentStyle.Width(width).Height(height).Render(
			lipgloss.JoinVertical(lipgloss.Left, title, body),
		)
	}

	innerWidth := width - 6
	if innerWidth < 46 {
		innerWidth = 46
	}

	focused := s.ContentFocused

	sinksBorder := colorBorder
	sourcesBorder := colorBorder
	if focused {
		switch s.AudioFocus {
		case core.AudioFocusSources:
			sourcesBorder = colorAccent
		default:
			sinksBorder = colorAccent
		}
	}

	var blocks []string

	if s.AudioDeviceInfoOpen {
		dev, isSink, ok := s.SelectedAudioDevice()
		if ok {
			blocks = append(blocks, renderAudioDeviceInfoBox(s, dev, isSink, innerWidth))
		} else {
			s.AudioDeviceInfoOpen = false
		}
	}

	if !s.AudioDeviceInfoOpen {
		sinksBox := renderAudioDeviceBox(
			s, "Output Devices", s.Audio.Sinks, s.AudioSinkIdx,
			focused && s.AudioFocus == core.AudioFocusSinks, true,
			innerWidth, sinksBorder,
		)
		sourcesBox := renderAudioDeviceBox(
			s, "Input Devices", s.Audio.Sources, s.AudioSourceIdx,
			focused && s.AudioFocus == core.AudioFocusSources, false,
			innerWidth, sourcesBorder,
		)

		blocks = append(blocks, sinksBox, "", sourcesBox)

		if len(s.Audio.SinkInputs) > 0 {
			playBorder := colorBorder
			if focused && s.AudioFocus == core.AudioFocusPlayApps {
				playBorder = colorAccent
			}
			playBox := renderAudioStreamBox(
				s, "Playing Applications", s.Audio.SinkInputs, s.AudioPlayAppIdx,
				focused && s.AudioFocus == core.AudioFocusPlayApps, true,
				innerWidth, playBorder,
			)
			blocks = append(blocks, "", playBox)
		}

		if len(s.Audio.SourceOutputs) > 0 {
			recBorder := colorBorder
			if focused && s.AudioFocus == core.AudioFocusRecordApps {
				recBorder = colorAccent
			}
			recBox := renderAudioStreamBox(
				s, "Recording Applications", s.Audio.SourceOutputs, s.AudioRecordAppIdx,
				focused && s.AudioFocus == core.AudioFocusRecordApps, false,
				innerWidth, recBorder,
			)
			blocks = append(blocks, "", recBox)
		}

		// Card box only renders when focus is on a device sub-table —
		// it would be confusing or empty when an apps row is selected.
		if s.AudioFocus == core.AudioFocusSinks || s.AudioFocus == core.AudioFocusSources {
			if cardBox, ok := renderAudioCardBox(s, innerWidth); ok {
				blocks = append(blocks, "", cardBox)
			}
		}
	}

	// Errors fire desktop notifications instead of rendering inline.
	if s.AudioBusy {
		blocks = append(blocks, "", statusBusyStyle.Render("working…"))
	}

	blocks = append(blocks, renderAudioFocusHint(s, focused))

	body := lipgloss.JoinVertical(lipgloss.Left, blocks...)
	return contentStyle.Width(width).Height(height).Render(body)
}

// renderAudioDeviceInfoBox is the second-level drill: full property
// readout for one device, plus a wide slider, the backing card's
// profile list, and the active port list. Replaces the Output/Input
// device boxes when open. Press esc to back out.
func renderAudioDeviceInfoBox(s *core.State, d audio.Device, isSink bool, total int) string {
	innerWidth := total - 4
	if innerWidth < 24 {
		innerWidth = 24
	}

	kind := "Input"
	if isSink {
		kind = "Output"
	}
	title := kind + " · " + audioDisplayName(d)

	rows := [][2]string{
		{"Description", orDash(d.Description)},
		{"Internal Name", orDash(d.Name)},
		{"Index", fmt.Sprintf("%d", d.Index)},
		{"State", orDash(d.State)},
		{"Mute", yesNo(d.Mute)},
		{"Default", yesNo(d.IsDefault)},
		{"Channels", fmt.Sprintf("%d", d.Channels)},
		{"Active Port", orDash(activePortLabel(d))},
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

	sections := []string{strings.Join(propLines, "\n")}
	sections = append(sections, renderAudioVolumeSlider(s, d, isSink, innerWidth))

	if card, ok := s.Audio.CardByIndex(d.CardIndex); ok {
		sections = append(sections, renderAudioCardHeader(card))
		if len(card.Profiles) > 0 {
			sections = append(sections, renderAudioProfileList(card))
		}
	}
	if len(d.Ports) > 0 {
		sections = append(sections, renderAudioPortList(d))
	}
	sections = append(sections, placeholderStyle.Render(
		"+/- volume · m mute · p profile · o port · D set default · esc back"))

	return groupBoxSections(title, []string{strings.Join(sections, "\n\n")}, total, colorAccent)
}

// renderAudioDeviceBox renders one of the two device sub-tables
// (sinks or sources). Each device occupies two lines: a table-style
// header row (marker, name, state, mute) the user can highlight and
// drill into, plus a horizontal volume slider with an inline VU meter
// indented below it. isSink controls which level map the meter reads
// from when fetching peak values for each row.
func renderAudioDeviceBox(state *core.State, title string, devs []audio.Device, selected int, focused, isSink bool, total int, border lipgloss.Color) string {
	if len(devs) == 0 {
		body := placeholderStyle.Render("no devices")
		return groupBoxSections(title, []string{body}, total, border)
	}

	innerWidth := total - 4
	if innerWidth < 24 {
		innerWidth = 24
	}

	type col struct {
		header string
		cell   func(audio.Device) string
	}
	cols := []col{
		{"Name", func(d audio.Device) string { return orDash(audioDisplayName(d)) }},
		{"State", func(d audio.Device) string { return orDash(d.State) }},
		{"Mute", func(d audio.Device) string { return muteGlyph(d.Mute) }},
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
	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines := []string{"   " + strings.Join(headerCells, gap)}

	for i, d := range devs {
		isSel := focused && i == selected
		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.cell(d)
			var style lipgloss.Style
			switch {
			case isSel:
				style = tableCellSelected
			case d.IsDefault:
				style = tableCellAccent
			case d.Mute:
				style = placeholderStyle
			default:
				style = tableCellStyle
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		row := audioRowMarker(isSel, d.IsDefault) + strings.Join(cells, gap)
		lines = append(lines, row)
		lines = append(lines, renderAudioVolumeSlider(state, d, isSink, innerWidth))
	}
	return groupBoxSections(title, []string{strings.Join(lines, "\n")}, total, border)
}

// renderAudioStreamBox renders one of the per-app stream sub-tables
// (sink inputs or source outputs). Each stream is two lines: a header
// row with the marker, mute glyph, app/media name, and the device
// it's currently routed to; followed by an indented volume slider.
// The slider's inline VU meter reads from the level of the device the
// stream is routed through, since per-app peak detection isn't part
// of the protocol.
func renderAudioStreamBox(state *core.State, title string, streams []audio.Stream, selected int, focused, isPlay bool, total int, border lipgloss.Color) string {
	if len(streams) == 0 {
		body := placeholderStyle.Render("no active streams")
		return groupBoxSections(title, []string{body}, total, border)
	}

	innerWidth := total - 4
	if innerWidth < 24 {
		innerWidth = 24
	}

	type col struct {
		header string
		cell   func(audio.Stream) string
	}
	cols := []col{
		{"Application", func(s audio.Stream) string { return orDash(s.DisplayName()) }},
		{"Routed To", func(s audio.Stream) string { return orDash(s.DeviceName) }},
		{"Mute", func(s audio.Stream) string { return muteGlyph(s.Mute) }},
	}

	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = lipgloss.Width(c.header)
	}
	for _, st := range streams {
		for i, c := range cols {
			if w := lipgloss.Width(c.cell(st)); w > widths[i] {
				widths[i] = w
			}
		}
	}

	const gap = "  "
	headerCells := make([]string, 0, len(cols))
	for i, c := range cols {
		headerCells = append(headerCells, tableHeaderStyle.Width(widths[i]).Render(c.header))
	}
	lines := []string{"   " + strings.Join(headerCells, gap)}

	for i, st := range streams {
		isSel := focused && i == selected
		cells := make([]string, 0, len(cols))
		for j, c := range cols {
			text := c.cell(st)
			var style lipgloss.Style
			switch {
			case isSel:
				style = tableCellSelected
			case st.Mute:
				style = placeholderStyle
			default:
				style = tableCellStyle
			}
			cells = append(cells, style.Width(widths[j]).Render(text))
		}
		row := streamRowMarker(isSel) + strings.Join(cells, gap)
		lines = append(lines, row)
		lines = append(lines, renderAudioStreamSlider(state, st, isPlay, innerWidth))
	}
	return groupBoxSections(title, []string{strings.Join(lines, "\n")}, total, border)
}

// streamRowMarker matches audioRowMarker's two-glyph layout but
// without the default-device glyph (streams have no default concept).
func streamRowMarker(selected bool) string {
	sel := " "
	if selected {
		sel = tableSelectionMarker.Render("▸")
	}
	return sel + "  "
}

// renderAudioStreamSlider mirrors renderAudioVolumeSlider for streams.
// The inline meter pulls from the routed device's level since the
// PulseAudio protocol doesn't expose per-stream peak detection.
func renderAudioStreamSlider(s *core.State, st audio.Stream, isPlay bool, width int) string {
	const indent = 3
	const labelWidth = 5
	const meterWidth = 16
	const meterGap = 2
	barWidth := width - indent - labelWidth - meterGap - meterWidth
	if barWidth < 10 {
		barWidth = 10
	}

	pct := st.Volume
	if pct < 0 {
		pct = 0
	}
	filled := pct * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}

	filledStyle := audioBarFilledStyle
	if st.Mute {
		filledStyle = audioBarMutedStyle
	}

	label := placeholderStyle.Render(fmt.Sprintf("%3d%% ", pct))
	filledPart := filledStyle.Render(strings.Repeat("─", filled))
	emptyPart := audioBarEmptyStyle.Render(strings.Repeat("┄", barWidth-filled))

	var levels [2]float32
	if isPlay {
		levels = s.SinkLevel(st.DeviceIndex)
	} else {
		levels = s.SourceLevel(st.DeviceIndex)
	}
	if st.Mute {
		levels = [2]float32{}
	}
	meter := renderAudioStereoMeter(levels[0], levels[1], meterWidth)

	return strings.Repeat(" ", indent) + label + filledPart + emptyPart + strings.Repeat(" ", meterGap) + meter
}

// audioRowMarker is the 3-cell prefix for a device row: selection
// cursor, default-device glyph, trailing space.
func audioRowMarker(selected, isDefault bool) string {
	sel := " "
	if selected {
		sel = tableSelectionMarker.Render("▸")
	}
	status := " "
	if isDefault {
		status = tableSelectionMarker.Render("★")
	}
	return sel + status + " "
}

// renderAudioVolumeSlider returns the slider line that sits beneath
// each device row: indented past the marker columns, then "NNN% ", a
// horizontal volume bar, and a center-anchored stereo VU meter on the
// right.
//
// Volume bar: filled portion is `─` (light horizontal) in accent
// color, dim when muted. Unfilled portion is `┄` (light triple dash)
// in the muted-border color.
//
// VU meter: a 16-cell stereo indicator with the silent point at the
// center. Left channel grows leftward from center, right channel
// grows rightward. Color zones run green (inner) → gold → red at the
// extremes so clipping is visible at a glance.
func renderAudioVolumeSlider(s *core.State, d audio.Device, isSink bool, width int) string {
	const indent = 3
	const labelWidth = 5
	const meterWidth = 16
	const meterGap = 2
	barWidth := width - indent - labelWidth - meterGap - meterWidth
	if barWidth < 10 {
		barWidth = 10
	}

	pct := d.Volume
	if pct < 0 {
		pct = 0
	}
	filled := pct * barWidth / 100
	if filled > barWidth {
		filled = barWidth
	}

	filledStyle := audioBarFilledStyle
	if d.Mute {
		filledStyle = audioBarMutedStyle
	}

	label := placeholderStyle.Render(fmt.Sprintf("%3d%% ", pct))
	filledPart := filledStyle.Render(strings.Repeat("─", filled))
	emptyPart := audioBarEmptyStyle.Render(strings.Repeat("┄", barWidth-filled))

	var levels [2]float32
	if isSink {
		levels = s.SinkLevel(d.Index)
	} else {
		levels = s.SourceLevel(d.Index)
	}
	if d.Mute {
		levels = [2]float32{}
	}
	meter := renderAudioStereoMeter(levels[0], levels[1], meterWidth)

	return strings.Repeat(" ", indent) + label + filledPart + emptyPart + strings.Repeat(" ", meterGap) + meter
}

// renderAudioStereoMeter draws a center-anchored stereo VU meter.
// Width must be even; half the cells go to the left channel (drawn
// from outer edge inward toward center) and half to the right (drawn
// from center outward toward the right edge). Silence puts every cell
// in the dim dotted state. As each channel's peak rises, cells light
// up starting at the center and progressing outward, hitting the
// gold/red zones at the very edges so clipping is unmistakable.
//
// Mono devices read identical L and R values from the levels map
// (PulseAudio upmixes them server-side), which means a mono mic
// produces a perfectly symmetric meter — accurate and visually clean.
func renderAudioStereoMeter(left, right float32, width int) string {
	half := width / 2
	return renderAudioHalfMeter(left, half, true) + renderAudioHalfMeter(right, half, false)
}

// renderAudioHalfMeter renders one channel of a stereo meter. When
// leftSide is true the cells run outer-edge → center (so the lit
// region appears anchored to the center, growing outward leftward).
// When false the cells run center → outer-edge.
func renderAudioHalfMeter(level float32, halfWidth int, leftSide bool) string {
	if level < 0 {
		level = 0
	}
	if level > 1 {
		level = 1
	}
	lit := int(level * float32(halfWidth))
	if lit > halfWidth {
		lit = halfWidth
	}

	var b strings.Builder
	for i := 0; i < halfWidth; i++ {
		// distFromCenter: 0 = innermost (lit at low levels),
		// halfWidth-1 = outermost (lit only when channel is loud).
		var distFromCenter int
		if leftSide {
			distFromCenter = halfWidth - 1 - i
		} else {
			distFromCenter = i
		}

		if distFromCenter < lit {
			switch {
			case distFromCenter >= halfWidth-1:
				b.WriteString(audioMeterHotStyle.Render("┃"))
			case distFromCenter >= halfWidth-2:
				b.WriteString(audioMeterWarmStyle.Render("┃"))
			default:
				b.WriteString(audioMeterFilledStyle.Render("┃"))
			}
		} else {
			b.WriteString(audioMeterDimStyle.Render("┊"))
		}
	}
	return b.String()
}

// activePortLabel returns a friendly description of the device's
// active port, falling back to the raw port name. Returns "" when
// the device has no port (virtual sinks).
func activePortLabel(d audio.Device) string {
	if d.ActivePort == "" {
		return ""
	}
	for _, p := range d.Ports {
		if p.Name == d.ActivePort {
			if p.Description != "" {
				return p.Description
			}
			return p.Name
		}
	}
	return d.ActivePort
}

// renderAudioCardBox renders the Card group box reflecting whichever
// device is currently selected. Returns false when there's nothing
// useful to show — virtual sinks (null sinks, loopbacks) aren't
// card-backed and would just produce an empty box.
func renderAudioCardBox(s *core.State, total int) (string, bool) {
	dev, _, ok := s.SelectedAudioDevice()
	if !ok {
		return "", false
	}
	card, hasCard := s.Audio.CardByIndex(dev.CardIndex)
	if !hasCard && len(dev.Ports) == 0 {
		return "", false
	}

	var sections []string

	if hasCard {
		sections = append(sections, renderAudioCardHeader(card))
		sections = append(sections, renderAudioProfileList(card))
	}
	if len(dev.Ports) > 0 {
		sections = append(sections, renderAudioPortList(dev))
	}
	sections = append(sections, placeholderStyle.Render("p cycle profile · o cycle port"))

	title := "Card"
	if card.Description != "" {
		title = "Card · " + card.Description
	} else if card.Name != "" {
		title = "Card · " + card.Name
	}
	return groupBoxSections(title, []string{strings.Join(sections, "\n\n")}, total, colorBorder), true
}

func renderAudioCardHeader(card audio.Card) string {
	rows := [][2]string{
		{"Name", orDash(card.Name)},
		{"Driver", orDash(card.Driver)},
		{"Active Profile", orDash(card.ActiveProfile)},
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

func renderAudioProfileList(card audio.Card) string {
	if len(card.Profiles) == 0 {
		return placeholderStyle.Render("(no profiles)")
	}
	lines := []string{tableHeaderStyle.Render("Profiles")}
	for _, p := range card.Profiles {
		marker := "  "
		if p.Name == card.ActiveProfile {
			marker = tableSelectionMarker.Render("★ ")
		}
		label := orDash(p.Description)
		if label == "—" {
			label = p.Name
		}
		raw := detailLabelStyle.Render("(" + p.Name + ")")
		var line string
		switch {
		case p.Available == 1:
			line = marker + placeholderStyle.Render(label) + "  " + placeholderStyle.Render("(unavailable)")
		case p.Name == card.ActiveProfile:
			line = marker + tableCellAccent.Render(label) + "  " + raw
		default:
			line = marker + detailValueStyle.Render(label) + "  " + raw
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func renderAudioPortList(dev audio.Device) string {
	lines := []string{tableHeaderStyle.Render("Ports")}
	for _, p := range dev.Ports {
		marker := "  "
		if p.Name == dev.ActivePort {
			marker = tableSelectionMarker.Render("★ ")
		}
		label := orDash(p.Description)
		if label == "—" {
			label = p.Name
		}
		raw := detailLabelStyle.Render("(" + p.Name + ")")
		var line string
		switch {
		case p.Available == 1:
			line = marker + placeholderStyle.Render(label) + "  " + placeholderStyle.Render("(unplugged)")
		case p.Name == dev.ActivePort:
			line = marker + tableCellAccent.Render(label) + "  " + raw
		default:
			line = marker + detailValueStyle.Render(label) + "  " + raw
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// audioDisplayName prefers the human-readable Description over the raw
// Name (which is usually an internal identifier like `alsa_output.
// pci-0000_00_1f.3.analog-stereo`).
func audioDisplayName(d audio.Device) string {
	if d.Description != "" {
		return d.Description
	}
	return d.Name
}

func muteGlyph(muted bool) string {
	if muted {
		return "󰝟"
	}
	return "󰕾"
}

func renderAudioFocusHint(s *core.State, focused bool) string {
	var text string
	switch {
	case s.AudioDeviceInfoOpen:
		text = "esc back · +/- vol · m mute · p profile · o port · Z suspend · D default"
	case focused && (s.AudioFocus == core.AudioFocusPlayApps || s.AudioFocus == core.AudioFocusRecordApps):
		text = "tab · j/k · +/- vol · m mute · M move · K kill · esc"
	case focused:
		text = "tab · j/k · enter info · +/- · m · p profile · o port · Z suspend · D default · esc"
	default:
		text = "enter · then tab/+-/m/p/o/Z/M/K/D"
	}
	return statusBarStyle.Render(text)
}
