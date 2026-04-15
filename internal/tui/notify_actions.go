package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/notifycfg"
)

type NotifyConfigActions struct {
	ToggleDND    func() tea.Cmd
	DismissAll   func() tea.Cmd
	SetAnchor    func(anchor string) tea.Cmd
	SetTimeout   func(ms int) tea.Cmd
	SetWidth     func(px int) tea.Cmd
	SetLayer     func(layer string) tea.Cmd
	SetSound     func(soundPath string) tea.Cmd
	AddAppRule   func(appName string, hide bool) tea.Cmd
	RemoveRule   func(criteria string) tea.Cmd
}

type NotifyCfgMsg notifycfg.Snapshot

type NotifyCfgActionResultMsg struct {
	Snapshot notifycfg.Snapshot
	Err      string
}

func (m *Model) inNotifyContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "notifications"
}

func (m *Model) triggerNotifyDNDToggle() tea.Cmd {
	if m.notifyCfg.ToggleDND == nil || !m.inNotifyContent() {
		return nil
	}
	return m.notifyCfg.ToggleDND()
}

func (m *Model) triggerNotifyDismissAll() tea.Cmd {
	if m.notifyCfg.DismissAll == nil || !m.inNotifyContent() {
		return nil
	}
	return m.notifyCfg.DismissAll()
}

func (m *Model) triggerNotifyAnchorCycle() tea.Cmd {
	if m.notifyCfg.SetAnchor == nil || !m.inNotifyContent() {
		return nil
	}
	positions := []string{"top-right", "top-left", "top-center", "bottom-right", "bottom-left", "bottom-center", "center"}
	current := m.state.Notify.Anchor
	next := positions[0]
	for i, p := range positions {
		if p == current && i+1 < len(positions) {
			next = positions[i+1]
			break
		}
	}
	return m.notifyCfg.SetAnchor(next)
}

func (m *Model) triggerNotifyTimeoutDelta(delta int) tea.Cmd {
	if m.notifyCfg.SetTimeout == nil || !m.inNotifyContent() {
		return nil
	}
	ms := m.state.Notify.TimeoutMS + delta
	if ms < 1000 {
		ms = 1000
	}
	if ms > 30000 {
		ms = 30000
	}
	return m.notifyCfg.SetTimeout(ms)
}

func (m *Model) triggerNotifyWidthDelta(delta int) tea.Cmd {
	if m.notifyCfg.SetWidth == nil || !m.inNotifyContent() {
		return nil
	}
	px := m.state.Notify.Width + delta
	if px < 100 {
		px = 100
	}
	if px > 800 {
		px = 800
	}
	return m.notifyCfg.SetWidth(px)
}

func (m *Model) triggerNotifyLayerToggle() tea.Cmd {
	if m.notifyCfg.SetLayer == nil || !m.inNotifyContent() {
		return nil
	}
	current := m.state.Notify.Layer
	next := "overlay"
	if current == "overlay" {
		next = "top"
	}
	return m.notifyCfg.SetLayer(next)
}

func (m *Model) triggerNotifySoundDialog() {
	if m.notifyCfg.SetSound == nil || !m.inNotifyContent() {
		return
	}
	sounds := m.state.Notify.Sounds
	if len(sounds) == 0 {
		m.notifyError("Notifications", "no system sounds found")
		return
	}
	current := "message"
	notifyCfgRef := m.notifyCfg
	m.dialog = NewDialog("Set notification sound", []DialogFieldSpec{
		{Key: "sound", Label: "Sound", Kind: DialogFieldSelect, Options: sounds, Value: current,
			OnChange: func(value string) {
				previewSound(value)
			},
		},
	}, func(result DialogResult) tea.Cmd {
		stopPreview()
		name := result["sound"]
		if name == "" {
			return nil
		}
		path := "/usr/share/sounds/freedesktop/stereo/" + name + ".oga"
		return notifyCfgRef.SetSound(path)
	})
}

func (m *Model) triggerNotifySoundDisable() tea.Cmd {
	if m.notifyCfg.SetSound == nil || !m.inNotifyContent() {
		return nil
	}
	return m.notifyCfg.SetSound("")
}

func (m *Model) triggerNotifyAddRuleDialog() {
	if m.notifyCfg.AddAppRule == nil || !m.inNotifyContent() {
		return
	}
	notifyCfgRef := m.notifyCfg
	m.dialog = NewDialog("Add notification rule", []DialogFieldSpec{
		{Key: "app", Label: "App name"},
		{Key: "action", Label: "Action (hide/show)", Value: "hide"},
	}, func(result DialogResult) tea.Cmd {
		app := result["app"]
		if app == "" {
			return nil
		}
		hide := result["action"] != "show"
		return notifyCfgRef.AddAppRule(app, hide)
	})
}

func (m *Model) triggerNotifyRemoveRuleDialog() {
	if m.notifyCfg.RemoveRule == nil || !m.inNotifyContent() {
		return
	}
	rules := m.state.Notify.Rules
	if len(rules) == 0 {
		m.notifyError("Notifications", "no rules to remove")
		return
	}
	notifyCfgRef := m.notifyCfg
	m.dialog = NewDialog("Remove notification rule", []DialogFieldSpec{
		{Key: "criteria", Label: "Rule criteria (from list above)"},
	}, func(result DialogResult) tea.Cmd {
		criteria := result["criteria"]
		if criteria == "" {
			return nil
		}
		return notifyCfgRef.RemoveRule(criteria)
	})
}
