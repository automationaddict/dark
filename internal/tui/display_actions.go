package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/display"
)

// DisplayActions is the set of asynchronous commands the TUI can
// dispatch at darkd to drive the display service.
type DisplayActions struct {
	SetResolution    func(name string, width, height int, refreshRate float64) tea.Cmd
	SetScale         func(name string, scale float64) tea.Cmd
	SetTransform     func(name string, transform int) tea.Cmd
	SetPosition      func(name string, x, y int) tea.Cmd
	SetDpms          func(name string, on bool) tea.Cmd
	SetVrr           func(name string, mode int) tea.Cmd
	SetMirror        func(name, mirrorOf string) tea.Cmd
	ToggleEnabled    func(name string) tea.Cmd
	Identify         func() tea.Cmd
	SetBrightness    func(pct int) tea.Cmd
	SetKbdBrightness func(pct int) tea.Cmd
	SetNightLight    func(enable bool, tempK int, gamma int) tea.Cmd
	SetGamma         func(pct int) tea.Cmd
	SaveProfile      func(name string) tea.Cmd
	ApplyProfile     func(name string) tea.Cmd
	DeleteProfile    func(name string) tea.Cmd
}

// DisplayMsg is dispatched whenever darkd publishes a display snapshot.
type DisplayMsg display.Snapshot

// DisplayActionResultMsg is dispatched when a display action completes.
type DisplayActionResultMsg struct {
	Snapshot display.Snapshot
	Err      string
}

func (m *Model) inDisplayContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "display"
}

func (m *Model) triggerDisplayDpmsToggle() tea.Cmd {
	if m.display.SetDpms == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return nil
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetDpms(mon.Name, !mon.DpmsStatus)
}

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

func (m *Model) triggerDisplayCycleTransform() tea.Cmd {
	if m.display.SetTransform == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return nil
	}
	next := (mon.Transform + 1) % 4
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetTransform(mon.Name, next)
}

func (m *Model) triggerDisplayScaleUp() tea.Cmd {
	return m.triggerDisplayScaleDelta(0.25)
}

func (m *Model) triggerDisplayScaleDown() tea.Cmd {
	return m.triggerDisplayScaleDelta(-0.25)
}

func (m *Model) triggerDisplayScaleDelta(delta float64) tea.Cmd {
	if m.display.SetScale == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return nil
	}
	newScale := mon.Scale + delta
	if newScale < 0.25 {
		newScale = 0.25
	}
	if newScale > 4.0 {
		newScale = 4.0
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetScale(mon.Name, newScale)
}

func (m *Model) triggerDisplayVrrToggle() tea.Cmd {
	if m.display.SetVrr == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return nil
	}
	mode := 0
	if !mon.Vrr {
		mode = 1
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetVrr(mon.Name, mode)
}

// handleDisplayLayoutKey routes keys while the arrangement view is open.
// Arrow keys nudge the selected monitor's position; j/k cycle through
// monitors; esc/q close layout mode; i triggers identify.
func (m Model) handleDisplayLayoutKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state.CloseDisplayLayout()
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "q":
		m.state.CloseDisplayLayout()
		return m, nil
	case "j":
		m.state.MoveDisplaySelection(1)
	case "k":
		m.state.MoveDisplaySelection(-1)
	case "left", "h":
		return m.nudgeDisplayPosition(-1, 0)
	case "right", "l":
		return m.nudgeDisplayPosition(1, 0)
	case "up":
		return m.nudgeDisplayPosition(0, -1)
	case "down":
		return m.nudgeDisplayPosition(0, 1)
	case "i":
		if cmd := m.triggerDisplayIdentify(); cmd != nil {
			return m, cmd
		}
	case "?":
		m.state.OpenHelp()
	}
	return m, nil
}

func (m Model) nudgeDisplayPosition(dx, dy int) (tea.Model, tea.Cmd) {
	if m.display.SetPosition == nil || m.state.DisplayBusy {
		return m, nil
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return m, nil
	}
	newX, newY := snapPosition(m.state.Display.Monitors, m.state.DisplayMonitorIdx, dx, dy)
	if newX == mon.X && newY == mon.Y {
		return m, nil
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m, m.display.SetPosition(mon.Name, newX, newY)
}

func (m *Model) triggerDisplayIdentify() tea.Cmd {
	if m.display.Identify == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.Identify()
}

func (m *Model) triggerDisplayBrightnessDelta(delta int) tea.Cmd {
	if m.display.SetBrightness == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	if !m.state.Display.HasBacklight {
		return nil
	}
	pct := m.state.Display.Brightness + delta
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetBrightness(pct)
}

func (m *Model) triggerDisplayKbdBrightnessDelta(delta int) tea.Cmd {
	if m.display.SetKbdBrightness == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	if !m.state.Display.HasKbdLight {
		return nil
	}
	pct := m.state.Display.KbdBrightness + delta
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetKbdBrightness(pct)
}

func (m *Model) triggerDisplayNightLightToggle() tea.Cmd {
	if m.display.SetNightLight == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	enable := !m.state.NightLightActive
	temp := m.state.NightLightTemp
	if temp == 0 {
		temp = 4500
	}
	gamma := m.state.NightLightGamma
	if gamma == 0 {
		gamma = 100
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetNightLight(enable, temp, gamma)
}

func (m *Model) triggerDisplayNightLightTempDialog() {
	if m.display.SetNightLight == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return
	}
	current := m.state.NightLightTemp
	if current == 0 {
		current = 4500
	}

	displayRef := m.display
	state := m.state
	gamma := m.state.NightLightGamma
	if gamma == 0 {
		gamma = 100
	}

	m.dialog = NewDialog("Night Light Temperature",
		[]DialogFieldSpec{
			{Key: "temp", Label: "Temperature (K)", Kind: DialogFieldText, Value: strconv.Itoa(current)},
		},
		func(result DialogResult) tea.Cmd {
			t, err := strconv.Atoi(strings.TrimSpace(result["temp"]))
			if err != nil || t < 1000 || t > 10000 {
				state.DisplayActionError = "temperature must be 1000-10000K"
				return nil
			}
			state.DisplayBusy = true
			state.DisplayActionError = ""
			return displayRef.SetNightLight(true, t, gamma)
		},
	)
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
