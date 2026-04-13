package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) triggerDisplayToggleEnabled() tea.Cmd {
	if m.display.ToggleEnabled == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return nil
	}

	// Warn if disabling the only active monitor.
	if !mon.Disabled {
		active := 0
		for _, other := range m.state.Display.Monitors {
			if !other.Disabled {
				active++
			}
		}
		if active <= 1 {
			name := mon.Name
			displayRef := m.display
			state := m.state
			m.dialog = NewDialog("Disable "+name+"?", []DialogFieldSpec{
				{Key: "confirm", Label: "This is your only active display. You may not be able to get it back without restarting. Type YES to confirm"},
			}, func(result DialogResult) tea.Cmd {
				if result["confirm"] != "YES" {
					return nil
				}
				state.DisplayBusy = true
				state.DisplayActionError = ""
				return displayRef.ToggleEnabled(name)
			})
			return nil
		}
	}

	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.ToggleEnabled(mon.Name)
}

// triggerDisplayModeDialog pops a scrollable list of available modes
// for the selected monitor. The current mode is pre-selected.
func (m *Model) triggerDisplayModeDialog() {
	if m.display.SetResolution == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return
	}
	if len(mon.AvailableModes) == 0 {
		m.notifyError("Displays", "no available modes for this monitor")
		return
	}

	current := fmt.Sprintf("%dx%d@%.2fHz", mon.Width, mon.Height, mon.RefreshRate)
	displayRef := m.display
	state := m.state
	name := mon.Name

	m.dialog = NewDialog("Set mode for "+mon.Name,
		[]DialogFieldSpec{
			{Key: "mode", Label: "Resolution @ Refresh Rate", Kind: DialogFieldSelect, Options: mon.AvailableModes, Value: current},
		},
		func(result DialogResult) tea.Cmd {
			mode := strings.TrimSpace(result["mode"])
			if mode == "" || mode == current {
				return nil
			}
			w, h, rate, err := parseMode(mode)
			if err != nil {
				state.DisplayActionError = "invalid mode: " + err.Error()
				return nil
			}
			state.DisplayBusy = true
			state.DisplayActionError = ""
			return displayRef.SetResolution(name, w, h, rate)
		},
	)
}

// triggerDisplayPositionDialog pops a dialog to set monitor position.
func (m *Model) triggerDisplayPositionDialog() {
	if m.display.SetPosition == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return
	}

	displayRef := m.display
	state := m.state
	name := mon.Name

	m.dialog = NewDialog("Position for "+mon.Name,
		[]DialogFieldSpec{
			{Key: "x", Label: "X", Kind: DialogFieldText, Value: strconv.Itoa(mon.X)},
			{Key: "y", Label: "Y", Kind: DialogFieldText, Value: strconv.Itoa(mon.Y)},
		},
		func(result DialogResult) tea.Cmd {
			x, errX := strconv.Atoi(strings.TrimSpace(result["x"]))
			y, errY := strconv.Atoi(strings.TrimSpace(result["y"]))
			if errX != nil || errY != nil {
				state.DisplayActionError = "position must be integers"
				return nil
			}
			state.DisplayBusy = true
			state.DisplayActionError = ""
			return displayRef.SetPosition(name, x, y)
		},
	)
}

// triggerDisplayMirrorDialog pops a dialog to set which monitor to
// mirror (or clear mirroring).
func (m *Model) triggerDisplayMirrorDialog() {
	if m.display.SetMirror == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return
	}
	if len(m.state.Display.Monitors) < 2 {
		m.notifyError("Displays", "need at least two monitors to mirror")
		return
	}

	var others []string
	for _, other := range m.state.Display.Monitors {
		if other.Name != mon.Name {
			others = append(others, other.Name)
		}
	}

	displayRef := m.display
	state := m.state
	name := mon.Name

	m.dialog = NewDialog("Mirror "+mon.Name+" to",
		[]DialogFieldSpec{
			{Key: "target", Label: "Target (" + strings.Join(others, ", ") + ") or empty to clear", Kind: DialogFieldText},
		},
		func(result DialogResult) tea.Cmd {
			target := strings.TrimSpace(result["target"])
			if target == "" {
				return nil
			}
			state.DisplayBusy = true
			state.DisplayActionError = ""
			return displayRef.SetMirror(name, target)
		},
	)
}

// parseMode parses "WIDTHxHEIGHT@RATEHz" into its components.
func parseMode(s string) (int, int, float64, error) {
	s = strings.TrimSuffix(s, "Hz")

	atIdx := strings.Index(s, "@")
	if atIdx < 0 {
		return 0, 0, 0, fmt.Errorf("expected WxH@RATEHz format")
	}

	res := s[:atIdx]
	rateStr := s[atIdx+1:]

	xIdx := strings.Index(res, "x")
	if xIdx < 0 {
		return 0, 0, 0, fmt.Errorf("expected WxH@RATEHz format")
	}

	w, err := strconv.Atoi(res[:xIdx])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad width: %w", err)
	}
	h, err := strconv.Atoi(res[xIdx+1:])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad height: %w", err)
	}
	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("bad rate: %w", err)
	}
	return w, h, rate, nil
}
