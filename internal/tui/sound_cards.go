package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/audio"
)

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
		text = "esc back · +/- vol · </> bal · m mute · p profile · o port · Z suspend · D default"
	case focused && (s.AudioFocus == core.AudioFocusPlayApps || s.AudioFocus == core.AudioFocusRecordApps):
		text = "tab · j/k · +/- vol · m mute · M move · K kill · esc"
	case focused:
		text = "tab · j/k · enter info · +/- vol · </> bal · m · p · o · Z · D · esc"
	default:
		text = "enter · then tab/+-/m/p/o/Z/M/K/D"
	}
	return statusBarStyle.Render(text)
}
