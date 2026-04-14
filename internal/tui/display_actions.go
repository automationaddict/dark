package tui

import (
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
	SetGPUMode       func(mode string) tea.Cmd
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

func (m *Model) inDisplayDetails() bool {
	return m.inDisplayContent() && m.state.DisplayContentFocused
}

func (m *Model) triggerDisplayDpmsToggle() tea.Cmd {
	if !m.inDisplayDetails() || m.state.DisplayBusy {
		return nil
	}
	if m.display.SetDpms == nil {
		return m.notifyUnavailable("Displays")
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return nil
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetDpms(mon.Name, !mon.DpmsStatus)
}

func (m *Model) triggerDisplayCycleTransform() tea.Cmd {
	if !m.inDisplayDetails() || m.state.DisplayBusy {
		return nil
	}
	if m.display.SetTransform == nil {
		return m.notifyUnavailable("Displays")
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
	if m.display.SetScale == nil || !m.inDisplayDetails() || m.state.DisplayBusy {
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
	if m.display.SetVrr == nil || !m.inDisplayDetails() || m.state.DisplayBusy {
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
		if m.display.Identify != nil {
			m.state.DisplayBusy = true
			m.state.DisplayActionError = ""
			return m, m.display.Identify()
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

func (m *Model) triggerGPUModeToggle() tea.Cmd {
	if m.display.SetGPUMode == nil || !m.inDisplayContent() {
		return nil
	}
	gpu := m.state.Display.GPU
	if !gpu.HybridSupported {
		return nil
	}
	var newMode, prompt string
	if gpu.Mode == "Integrated" {
		newMode = "hybrid"
		prompt = "Enable dedicated GPU and reboot?"
	} else {
		newMode = "integrated"
		prompt = "Use only integrated GPU and reboot?"
	}
	displayRef := m.display
	m.dialog = NewDialog(prompt, nil, func(_ DialogResult) tea.Cmd {
		return displayRef.SetGPUMode(newMode)
	})
	return nil
}
