package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/keybind"
)

func renderKeybindings(s *core.State, width, height int) string {
	if !s.KeybindingsLoaded {
		return renderContentPane(width, height,
			placeholderStyle.Render("Loading keybindings…"))
	}

	secs := core.KeybindSections()
	entries := make([]sidebarEntry, len(secs))
	for i, sec := range secs {
		entries[i] = sidebarEntry{Icon: sec.Icon, Label: sec.Label, Enabled: true}
	}
	sidebarFocused := s.ContentFocused && !s.KeybindTableFocused
	sidebar := renderInnerSidebarFocused(s, entries, s.KeybindFilter, height, sidebarFocused)
	contentWidth := width - lipgloss.Width(sidebar)

	content := renderKeybindTable(s, contentWidth, height)
	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
}

func renderKeybindTable(s *core.State, width, height int) string {
	bindings := s.FilteredKeybindings()

	innerWidth := width - 6
	if innerWidth < 60 {
		innerWidth = 60
	}

	sec := core.KeybindSections()[s.KeybindFilter]
	title := contentTitle.Render(sec.Label)

	if len(bindings) == 0 {
		body := lipgloss.JoinVertical(lipgloss.Left,
			title, "",
			placeholderStyle.Render("No keybindings found."))
		return renderContentPane(width, height, body)
	}

	numW := 5
	modsW := 16
	keyW := 22
	sourceW := 8
	descW := innerWidth - numW - modsW - keyW - sourceW - 6
	if descW < 16 {
		descW = 16
	}

	selectedCell := lipgloss.NewStyle().
		Foreground(colorBg).
		Background(colorAccent)

	// Chrome lines: padding(1) + title(1) + blank(1) +
	// table header(3) + table bottom border(1) + blank(1) + hint(1) +
	// blank(1) + summary(1) + padding(1) = 12
	chromeLines := 12
	maxVisible := height - chromeLines
	if maxVisible < 3 {
		maxVisible = 3
	}

	startIdx := 0
	if len(bindings) > maxVisible {
		startIdx = s.KeybindIdx - maxVisible/2
		if startIdx < 0 {
			startIdx = 0
		}
		if startIdx+maxVisible > len(bindings) {
			startIdx = len(bindings) - maxVisible
		}
	}
	endIdx := startIdx + maxVisible
	if endIdx > len(bindings) {
		endIdx = len(bindings)
	}

	var data [][]string
	for i := startIdx; i < endIdx; i++ {
		b := bindings[i]
		source := string(b.Source)
		if b.Source == keybind.SourceDefault {
			source = "default"
		}
		mods := b.Mods
		if mods == "" {
			mods = "—"
		}
		desc := b.Desc
		if desc == "" {
			desc = b.Dispatcher
		}
		data = append(data, []string{
			fmt.Sprintf("%d", i+1),
			mods,
			friendlyKeyName(b.Key),
			desc,
			source,
		})
	}

	visibleIdx := s.KeybindIdx - startIdx

	table := renderTable(
		[]string{"#", "Modifiers", "Key", "Description", "Source"},
		[]int{numW, modsW, keyW, descW, sourceW},
		data,
		visibleIdx, s.KeybindTableFocused, selectedCell,
	)

	hint := lipgloss.NewStyle().Foreground(colorDim).Render(
		"enter edit · a add · d delete")

	summaryText := fmt.Sprintf("%d bindings", len(bindings))
	if len(bindings) > maxVisible {
		summaryText += fmt.Sprintf("  ↕ %d-%d/%d",
			startIdx+1, endIdx, len(bindings))
	}
	summary := lipgloss.NewStyle().Foreground(colorDim).Render(summaryText)

	body := lipgloss.JoinVertical(lipgloss.Left,
		title, "", table, "", hint, "", summary)

	return renderContentPane(width, height, strings.TrimRight(body, "\n"))
}

var xf86KeyNames = map[string]string{
	"XF86AudioRaiseVolume":  "Vol Up",
	"XF86AudioLowerVolume":  "Vol Down",
	"XF86AudioMute":         "Mute",
	"XF86AudioMicMute":      "Mic Mute",
	"XF86AudioPlay":         "Play/Pause",
	"XF86AudioPause":        "Pause",
	"XF86AudioStop":         "Stop",
	"XF86AudioNext":         "Next Track",
	"XF86AudioPrev":         "Prev Track",
	"XF86AudioMedia":        "Media",
	"XF86AudioRecord":       "Record",
	"XF86AudioRewind":       "Rewind",
	"XF86AudioForward":      "Fast Forward",
	"XF86MonBrightnessUp":   "Brightness Up",
	"XF86MonBrightnessDown": "Brightness Down",
	"XF86KbdBrightnessUp":   "Kbd Bright Up",
	"XF86KbdBrightnessDown": "Kbd Bright Down",
	"XF86KbdLightOnOff":     "Kbd Light Toggle",
	"XF86Display":           "Display",
	"XF86WLAN":              "WiFi",
	"XF86Bluetooth":         "Bluetooth",
	"XF86TouchpadToggle":    "Touchpad Toggle",
	"XF86TouchpadOn":        "Touchpad On",
	"XF86TouchpadOff":       "Touchpad Off",
	"XF86PowerOff":          "Power Off",
	"XF86PowerDown":         "Power Down",
	"XF86Sleep":             "Sleep",
	"XF86Suspend":           "Suspend",
	"XF86Hibernate":         "Hibernate",
	"XF86ScreenSaver":       "Screen Saver",
	"XF86Launch1":           "Launch 1",
	"XF86Launch2":           "Launch 2",
	"XF86Launch3":           "Launch 3",
	"XF86Launch4":           "Launch 4",
	"XF86Launch5":           "Launch 5",
	"XF86Calculator":        "Calculator",
	"XF86Explorer":          "File Manager",
	"XF86Mail":              "Mail",
	"XF86HomePage":          "Home Page",
	"XF86Search":            "Search",
	"XF86Eject":             "Eject",
	"XF86Tools":             "Tools",
	"XF86Favorites":         "Favorites",
	"XF86Back":              "Back",
	"XF86Forward":           "Forward",
	"XF86Reload":            "Reload",
	"XF86Camera":            "Camera",
	"XF86Phone":             "Phone",
	"XF86Copy":              "Copy",
	"XF86Cut":               "Cut",
	"XF86Paste":             "Paste",
	"XF86RFKill":            "RF Kill",
}

// Linux evdev keycodes to friendly names.
var evdevKeyNames = map[string]string{
	"code:1":   "Esc",
	"code:2":   "1",
	"code:3":   "2",
	"code:4":   "3",
	"code:5":   "4",
	"code:6":   "5",
	"code:7":   "6",
	"code:8":   "7",
	"code:9":   "8",
	"code:10":  "1",
	"code:11":  "2",
	"code:12":  "3",
	"code:13":  "4",
	"code:14":  "5",
	"code:15":  "6",
	"code:16":  "7",
	"code:17":  "8",
	"code:18":  "9",
	"code:19":  "0",
	"code:20":  "Minus",
	"code:21":  "Equal",
	"code:22":  "Backspace",
	"code:23":  "Tab",
	"code:59":  "F1",
	"code:60":  "F2",
	"code:61":  "F3",
	"code:62":  "F4",
	"code:63":  "F5",
	"code:64":  "F6",
	"code:65":  "F7",
	"code:66":  "F8",
	"code:67":  "F9",
	"code:68":  "F10",
	"code:87":  "F11",
	"code:88":  "F12",
	"code:96":  "Enter",
	"code:102": "Home",
	"code:103": "Up",
	"code:104": "PgUp",
	"code:105": "Left",
	"code:106": "Right",
	"code:107": "End",
	"code:108": "Down",
	"code:109": "PgDn",
	"code:110": "Insert",
	"code:111": "Delete",
	"code:113": "Mute",
	"code:114": "Vol Down",
	"code:115": "Vol Up",
	"code:116": "Power",
	"code:119": "Pause",
	"code:122": "Vol Down",
	"code:123": "Vol Up",
	"code:142": "Sleep",
	"code:150": "Sleep",
	"code:152": "Screen Off",
	"code:161": "Eject",
	"code:163": "Next Track",
	"code:164": "Play/Pause",
	"code:165": "Prev Track",
	"code:166": "Stop",
	"code:190": "Mic Mute",
	"code:224": "Brightness Down",
	"code:225": "Brightness Up",
	"code:237": "Kbd Bright Down",
	"code:238": "Kbd Bright Up",
}

func friendlyKeyName(key string) string {
	if name, ok := xf86KeyNames[key]; ok {
		return name
	}
	if name, ok := evdevKeyNames[key]; ok {
		return name
	}
	return key
}
