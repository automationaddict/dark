package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/services/display"
)

var scalePresets = []string{
	"0.50", "0.75", "1.00", "1.25", "1.50", "1.75", "2.00", "2.50", "3.00", "4.00",
}

func (m *Model) triggerDisplayScaleDialog() {
	if m.display.SetScale == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return
	}
	mon, ok := m.state.SelectedMonitor()
	if !ok {
		return
	}

	current := fmt.Sprintf("%.2f", mon.Scale)
	bestIdx := display.ClosestScaleIndex(scalePresets, mon.Scale)

	displayRef := m.display
	state := m.state
	name := mon.Name

	m.dialog = NewDialog("Scale for "+mon.Name,
		[]DialogFieldSpec{
			{Key: "scale", Label: "Scale", Kind: DialogFieldSelect, Options: scalePresets, Value: scalePresets[bestIdx]},
		},
		func(result DialogResult) tea.Cmd {
			chosen := strings.TrimSpace(result["scale"])
			if chosen == "" || chosen == current {
				return nil
			}
			s, err := strconv.ParseFloat(chosen, 64)
			if err != nil {
				state.DisplayActionError = "invalid scale"
				return nil
			}
			state.DisplayBusy = true
			state.DisplayActionError = ""
			return displayRef.SetScale(name, s)
		},
	)
}

func (m *Model) triggerDisplayGammaDelta(delta int) tea.Cmd {
	if m.display.SetGamma == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return nil
	}
	gamma := m.state.NightLightGamma
	if gamma == 0 {
		gamma = 100
	}
	gamma += delta
	if gamma < 0 {
		gamma = 0
	}
	if gamma > 200 {
		gamma = 200
	}
	m.state.DisplayBusy = true
	m.state.DisplayActionError = ""
	return m.display.SetGamma(gamma)
}

func (m *Model) triggerDisplaySaveProfile() {
	if m.display.SaveProfile == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return
	}

	displayRef := m.display
	state := m.state

	m.dialog = NewDialog("Save Display Profile",
		[]DialogFieldSpec{
			{Key: "name", Label: "Profile Name", Kind: DialogFieldText},
		},
		func(result DialogResult) tea.Cmd {
			name := strings.TrimSpace(result["name"])
			if name == "" {
				return nil
			}
			state.DisplayBusy = true
			state.DisplayActionError = ""
			return displayRef.SaveProfile(name)
		},
	)
}

func (m *Model) triggerDisplayApplyProfile() {
	if m.display.ApplyProfile == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return
	}
	profiles := m.state.Display.Profiles
	if len(profiles) == 0 {
		m.state.DisplayActionError = "no saved profiles"
		return
	}

	displayRef := m.display
	state := m.state

	m.dialog = NewDialog("Apply Display Profile",
		[]DialogFieldSpec{
			{Key: "name", Label: "Profile", Kind: DialogFieldSelect, Options: profiles},
		},
		func(result DialogResult) tea.Cmd {
			name := strings.TrimSpace(result["name"])
			if name == "" {
				return nil
			}
			state.DisplayBusy = true
			state.DisplayActionError = ""
			return displayRef.ApplyProfile(name)
		},
	)
}

func (m *Model) triggerDisplayDeleteProfile() {
	if m.display.DeleteProfile == nil || !m.inDisplayContent() || m.state.DisplayBusy {
		return
	}
	profiles := m.state.Display.Profiles
	if len(profiles) == 0 {
		m.state.DisplayActionError = "no saved profiles"
		return
	}

	displayRef := m.display
	state := m.state

	m.dialog = NewDialog("Delete Display Profile",
		[]DialogFieldSpec{
			{Key: "name", Label: "Profile", Kind: DialogFieldSelect, Options: profiles},
		},
		func(result DialogResult) tea.Cmd {
			name := strings.TrimSpace(result["name"])
			if name == "" {
				return nil
			}
			state.DisplayBusy = true
			state.DisplayActionError = ""
			return displayRef.DeleteProfile(name)
		},
	)
}
