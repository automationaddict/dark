package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/input"
)

type InputActions struct {
	SetRepeatRate   func(rate int) tea.Cmd
	SetRepeatDelay  func(delay int) tea.Cmd
	SetSensitivity  func(sens float64) tea.Cmd
	SetNaturalScroll func(enabled bool) tea.Cmd
	SetScrollFactor func(factor float64) tea.Cmd
	SetKBLayout     func(layout string) tea.Cmd
}

type InputMsg input.Snapshot

type InputActionResultMsg struct {
	Snapshot input.Snapshot
	Err      string
}

func (m *Model) inInputContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "input"
}

func (m *Model) triggerInputRepeatRateDelta(delta int) tea.Cmd {
	if m.input.SetRepeatRate == nil || !m.inInputContent() {
		return nil
	}
	rate := m.state.InputDevices.Config.RepeatRate + delta
	if rate < 1 {
		rate = 1
	}
	if rate > 100 {
		rate = 100
	}
	return m.input.SetRepeatRate(rate)
}

func (m *Model) triggerInputRepeatDelayDelta(delta int) tea.Cmd {
	if m.input.SetRepeatDelay == nil || !m.inInputContent() {
		return nil
	}
	delay := m.state.InputDevices.Config.RepeatDelay + delta
	if delay < 100 {
		delay = 100
	}
	if delay > 2000 {
		delay = 2000
	}
	return m.input.SetRepeatDelay(delay)
}

func (m *Model) triggerInputSensitivityDelta(delta float64) tea.Cmd {
	if m.input.SetSensitivity == nil || !m.inInputContent() {
		return nil
	}
	sens := m.state.InputDevices.Config.Sensitivity + delta
	if sens < -1.0 {
		sens = -1.0
	}
	if sens > 1.0 {
		sens = 1.0
	}
	return m.input.SetSensitivity(sens)
}

func (m *Model) triggerInputNaturalScrollToggle() tea.Cmd {
	if m.input.SetNaturalScroll == nil || !m.inInputContent() {
		return nil
	}
	return m.input.SetNaturalScroll(!m.state.InputDevices.Config.NaturalScroll)
}

func (m *Model) triggerInputScrollFactorDelta(delta float64) tea.Cmd {
	if m.input.SetScrollFactor == nil || !m.inInputContent() {
		return nil
	}
	factor := m.state.InputDevices.Config.ScrollFactor + delta
	if factor < 0.1 {
		factor = 0.1
	}
	if factor > 5.0 {
		factor = 5.0
	}
	return m.input.SetScrollFactor(factor)
}

func (m *Model) triggerInputKBLayoutDialog() {
	if m.input.SetKBLayout == nil || !m.inInputContent() {
		return
	}
	current := m.state.InputDevices.Config.KBLayout
	inputRef := m.input
	m.dialog = NewDialog("Keyboard Layout", []DialogFieldSpec{
		{Key: "layout", Label: "Layout (e.g. us, de, fr)", Value: current},
	}, func(result DialogResult) tea.Cmd {
		layout := result["layout"]
		if layout == "" || layout == current {
			return nil
		}
		return inputRef.SetKBLayout(layout)
	})
}
