package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/audio"
)

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

// renderAudioBalanceSlider shows a center-anchored balance indicator.
// The bar is split in half; the filled portion extends from center
// toward left or right depending on the balance value.
func renderAudioBalanceSlider(d audio.Device, width int) string {
	if d.Channels < 2 {
		return ""
	}

	const indent = 3
	const labelWidth = 5
	barWidth := width - indent - labelWidth - 2
	if barWidth < 10 {
		barWidth = 10
	}

	bal := d.Balance
	if bal < -100 {
		bal = -100
	}
	if bal > 100 {
		bal = 100
	}

	half := barWidth / 2
	center := half

	label := placeholderStyle.Render(fmt.Sprintf("%+4d ", bal))

	bar := make([]rune, barWidth)
	for i := range bar {
		bar[i] = '┄'
	}

	if bal < 0 {
		filled := -bal * half / 100
		for i := center - filled; i < center; i++ {
			if i >= 0 {
				bar[i] = '─'
			}
		}
	} else if bal > 0 {
		filled := bal * half / 100
		for i := center; i < center+filled && i < barWidth; i++ {
			bar[i] = '─'
		}
	}

	var rendered strings.Builder
	for i, r := range bar {
		ch := string(r)
		if i == center {
			rendered.WriteString(lipgloss.NewStyle().Foreground(colorText).Render("│"))
		} else if r == '─' {
			rendered.WriteString(audioBarFilledStyle.Render(ch))
		} else {
			rendered.WriteString(audioBarEmptyStyle.Render(ch))
		}
	}

	dim := lipgloss.NewStyle().Foreground(colorDim)
	return strings.Repeat(" ", indent) + label + dim.Render("L") + rendered.String() + dim.Render("R")
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
