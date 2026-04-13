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
		return renderContentPane(width, height,
			placeholderStyle.Render("loading audio devices…"),
		)
	}
	if len(s.Audio.Sinks) == 0 && len(s.Audio.Sources) == 0 {
		title := contentTitle.Render("Sound")
		body := placeholderStyle.Render("No audio devices detected.")
		return renderContentPane(width, height,
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
	return renderContentPane(width, height,body)
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
	if balSlider := renderAudioBalanceSlider(d, innerWidth); balSlider != "" {
		sections = append(sections, balSlider)
	}

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

