package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/power"
)

type PowerActions struct {
	SetProfile     func(profile string) tea.Cmd
	SetGovernor    func(gov string) tea.Cmd
	SetEPP         func(epp string) tea.Cmd
	SetIdle        func(kind string, sec int) tea.Cmd
	SetIdleRunning func(running bool) tea.Cmd
	SetButton      func(key, value string) tea.Cmd
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

func (m *Model) triggerPowerButtonsDialog() {
	if m.power.SetButton == nil || !m.inPowerContent() {
		return
	}
	btn := m.state.Power.Buttons
	actRef := m.power
	actions := []string{"ignore", "poweroff", "reboot", "halt", "suspend",
		"hibernate", "hybrid-sleep", "suspend-then-hibernate", "lock"}

	m.dialog = NewDialog("System Buttons", []DialogFieldSpec{
		{Key: "power", Label: "Power Button", Kind: DialogFieldSelect, Options: actions, Value: btn.PowerKeyAction},
		{Key: "lid", Label: "Lid Close", Kind: DialogFieldSelect, Options: actions, Value: btn.LidSwitch},
		{Key: "lid_ac", Label: "Lid + AC Power", Kind: DialogFieldSelect, Options: actions, Value: btn.LidSwitchPower},
		{Key: "lid_dock", Label: "Lid + Docked", Kind: DialogFieldSelect, Options: actions, Value: btn.LidSwitchDocked},
	}, func(result DialogResult) tea.Cmd {
		keyMap := map[string]struct{ logindKey, orig string }{
			"power":    {"HandlePowerKey", btn.PowerKeyAction},
			"lid":      {"HandleLidSwitch", btn.LidSwitch},
			"lid_ac":   {"HandleLidSwitchExternalPower", btn.LidSwitchPower},
			"lid_dock": {"HandleLidSwitchDocked", btn.LidSwitchDocked},
		}
		var cmds []tea.Cmd
		for field, info := range keyMap {
			val := strings.TrimSpace(result[field])
			if val == "" || val == info.orig {
				continue
			}
			cmds = append(cmds, actRef.SetButton(info.logindKey, val))
		}
		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	})
}

// inPowerIdleSection is the stricter gate used by per-section idle
// actions: we only react when the user is actually inside the Screen
// & Idle sub-section of the Power page, not just anywhere on Power.
func (m *Model) inPowerIdleSection() bool {
	return m.inPowerContent() && m.state.ActivePowerSection().ID == "idle"
}

// triggerPowerIdleToggle starts or stops the hypridle daemon so the
// Omarchy "stop locking computer when idle" flow is available from
// dark. No-op when not on the Idle sub-section or when the backend
// function isn't wired in.
func (m *Model) triggerPowerIdleToggle() tea.Cmd {
	if !m.inPowerIdleSection() {
		return nil
	}
	if m.power.SetIdleRunning == nil {
		return m.notifyUnavailable("Power")
	}
	target := !m.state.Power.Idle.Running
	return m.power.SetIdleRunning(target)
}

func (m *Model) triggerPowerIdleDialog() {
	if m.power.SetIdle == nil || !m.inPowerContent() {
		return
	}
	idle := m.state.Power.Idle
	actRef := m.power

	m.dialog = NewDialog("Screen & Idle Timers (seconds, 0 to disable)", []DialogFieldSpec{
		{Key: "kbd", Label: "Kbd Backlight Off", Kind: DialogFieldText, Value: fmt.Sprintf("%d", idle.KbdBacklightSec)},
		{Key: "screensaver", Label: "Screensaver", Kind: DialogFieldText, Value: fmt.Sprintf("%d", idle.ScreensaverSec)},
		{Key: "lock", Label: "Screen Lock", Kind: DialogFieldText, Value: fmt.Sprintf("%d", idle.LockSec)},
		{Key: "dpms", Label: "Screen Off (DPMS)", Kind: DialogFieldText, Value: fmt.Sprintf("%d", idle.DPMSOffSec)},
	}, func(result DialogResult) tea.Cmd {
		var cmds []tea.Cmd
		for _, entry := range []struct{ key, kind string }{
			{"kbd", "kbd"},
			{"screensaver", "screensaver"},
			{"lock", "lock"},
			{"dpms", "dpms"},
		} {
			raw := strings.TrimSpace(result[entry.key])
			if raw == "" {
				continue
			}
			sec, err := strconv.Atoi(raw)
			if err != nil || sec < 0 {
				continue
			}
			var orig int
			switch entry.kind {
			case "kbd":
				orig = idle.KbdBacklightSec
			case "screensaver":
				orig = idle.ScreensaverSec
			case "lock":
				orig = idle.LockSec
			case "dpms":
				orig = idle.DPMSOffSec
			}
			if sec != orig {
				cmds = append(cmds, actRef.SetIdle(entry.kind, sec))
			}
		}
		if len(cmds) == 0 {
			return nil
		}
		return tea.Batch(cmds...)
	})
}
