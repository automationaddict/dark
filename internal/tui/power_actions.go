package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/power"
)

type PowerActions struct {
	SetProfile  func(profile string) tea.Cmd
	SetGovernor func(gov string) tea.Cmd
	SetEPP      func(epp string) tea.Cmd
}

type PowerMsg power.Snapshot

type PowerActionResultMsg struct {
	Snapshot power.Snapshot
	Err      string
}

func (m *Model) inPowerContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "power"
}

func (m *Model) triggerPowerProfileCycle() tea.Cmd {
	if m.power.SetProfile == nil || !m.inPowerContent() {
		return nil
	}
	profiles := m.state.Power.Profiles
	if len(profiles) == 0 {
		return nil
	}
	current := m.state.Power.Profile
	next := profiles[0]
	for i, p := range profiles {
		if p == current && i+1 < len(profiles) {
			next = profiles[i+1]
			break
		}
	}
	return m.power.SetProfile(next)
}

func (m *Model) triggerPowerGovernorCycle() tea.Cmd {
	if m.power.SetGovernor == nil || !m.inPowerContent() {
		return nil
	}
	govs := m.state.Power.Governors
	if len(govs) == 0 {
		return nil
	}
	current := m.state.Power.Governor
	next := govs[0]
	for i, g := range govs {
		if g == current && i+1 < len(govs) {
			next = govs[i+1]
			break
		}
	}
	return m.power.SetGovernor(next)
}

func (m *Model) triggerPowerEPPCycle() tea.Cmd {
	if m.power.SetEPP == nil || !m.inPowerContent() {
		return nil
	}
	epps := m.state.Power.EPPs
	if len(epps) == 0 {
		return nil
	}
	current := m.state.Power.EPP
	next := epps[0]
	for i, e := range epps {
		if e == current && i+1 < len(epps) {
			next = epps[i+1]
			break
		}
	}
	return m.power.SetEPP(next)
}
