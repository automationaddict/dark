package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/appearance"
)

type AppearanceActions struct {
	SetTheme    func(name string) tea.Cmd
	SetGapsIn   func(val int) tea.Cmd
	SetGapsOut  func(val int) tea.Cmd
	SetBorder   func(val int) tea.Cmd
	SetRounding func(val int) tea.Cmd
	SetBlur     func(enabled bool) tea.Cmd
	SetBlurSize func(val int) tea.Cmd
	SetBlurPass func(val int) tea.Cmd
	SetAnim     func(enabled bool) tea.Cmd
}

type AppearanceMsg appearance.Snapshot

type AppearanceActionResultMsg struct {
	Snapshot appearance.Snapshot
	Err      string
}

func (m *Model) inAppearanceContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "appearance"
}

func (m *Model) triggerAppearanceThemeDialog() {
	if m.appearance.SetTheme == nil || !m.inAppearanceContent() {
		return
	}
	current := m.state.Appearance.Theme
	themes := m.state.Appearance.Themes
	if len(themes) == 0 {
		return
	}
	actRef := m.appearance
	m.dialog = NewDialog("Set Theme", []DialogFieldSpec{
		{Key: "theme", Label: "Theme", Kind: DialogFieldSelect, Options: themes, Value: current},
	}, func(result DialogResult) tea.Cmd {
		name := result["theme"]
		if name == "" || name == current {
			return nil
		}
		return actRef.SetTheme(name)
	})
}

func (m *Model) triggerAppearanceGapsIn(delta int) tea.Cmd {
	if m.appearance.SetGapsIn == nil || !m.inAppearanceContent() {
		return nil
	}
	val := m.state.Appearance.General.GapsIn + delta
	if val < 0 {
		val = 0
	}
	if val > 50 {
		val = 50
	}
	return m.appearance.SetGapsIn(val)
}

func (m *Model) triggerAppearanceGapsOut(delta int) tea.Cmd {
	if m.appearance.SetGapsOut == nil || !m.inAppearanceContent() {
		return nil
	}
	val := m.state.Appearance.General.GapsOut + delta
	if val < 0 {
		val = 0
	}
	if val > 60 {
		val = 60
	}
	return m.appearance.SetGapsOut(val)
}

func (m *Model) triggerAppearanceBorderCycle(delta int) tea.Cmd {
	if m.appearance.SetBorder == nil || !m.inAppearanceContent() {
		return nil
	}
	val := m.state.Appearance.General.BorderSize + delta
	if val < 0 {
		val = 0
	}
	if val > 10 {
		val = 10
	}
	return m.appearance.SetBorder(val)
}

func (m *Model) triggerAppearanceRounding(delta int) tea.Cmd {
	if m.appearance.SetRounding == nil || !m.inAppearanceContent() {
		return nil
	}
	val := m.state.Appearance.Decoration.Rounding + delta
	if val < 0 {
		val = 0
	}
	if val > 30 {
		val = 30
	}
	return m.appearance.SetRounding(val)
}

func (m *Model) triggerAppearanceBlurToggle() tea.Cmd {
	if m.appearance.SetBlur == nil || !m.inAppearanceContent() {
		return nil
	}
	return m.appearance.SetBlur(!m.state.Appearance.Blur.Enabled)
}

func (m *Model) triggerAppearanceBlurSize(delta int) tea.Cmd {
	if m.appearance.SetBlurSize == nil || !m.inAppearanceContent() {
		return nil
	}
	val := m.state.Appearance.Blur.Size + delta
	if val < 1 {
		val = 1
	}
	if val > 20 {
		val = 20
	}
	return m.appearance.SetBlurSize(val)
}

func (m *Model) triggerAppearanceBlurPasses(delta int) tea.Cmd {
	if m.appearance.SetBlurPass == nil || !m.inAppearanceContent() {
		return nil
	}
	val := m.state.Appearance.Blur.Passes + delta
	if val < 1 {
		val = 1
	}
	if val > 10 {
		val = 10
	}
	return m.appearance.SetBlurPass(val)
}

func (m *Model) triggerAppearanceAnimToggle() tea.Cmd {
	if m.appearance.SetAnim == nil || !m.inAppearanceContent() {
		return nil
	}
	return m.appearance.SetAnim(!m.state.Appearance.Animations.Enabled)
}
