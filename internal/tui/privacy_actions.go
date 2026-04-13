package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/services/privacy"
)

type PrivacyActions struct {
	SetIdleTimeout    func(field string, seconds int) tea.Cmd
	SetDNSOverTLS     func(value string) tea.Cmd
	SetDNSSEC         func(value string) tea.Cmd
	SetFirewall       func(enable bool) tea.Cmd
	SetSSH            func(enable bool) tea.Cmd
	ClearRecent       func() tea.Cmd
	SetLocation       func(enable bool) tea.Cmd
	SetMACRandom      func(value string) tea.Cmd
	SetIndexer        func(enable bool) tea.Cmd
	SetCoredumpStorage func(value string) tea.Cmd
}

type PrivacyMsg privacy.Snapshot

type PrivacyActionResultMsg struct {
	Snapshot privacy.Snapshot
	Err      string
}

func (m *Model) inPrivacyContent() bool {
	return m.state.ContentFocused &&
		m.state.ActiveTab == core.TabSettings &&
		m.state.ActiveSection().ID == "privacy"
}

func (m *Model) triggerPrivacyScreensaverDialog() {
	if m.privacy.SetIdleTimeout == nil || !m.inPrivacyContent() {
		return
	}
	current := m.state.Privacy.ScreensaverTimeout
	privacyRef := m.privacy
	m.dialog = NewDialog("Screensaver timeout", []DialogFieldSpec{
		{Key: "seconds", Label: "Seconds (0 = disabled)", Value: strconv.Itoa(current)},
	}, func(result DialogResult) tea.Cmd {
		v, err := strconv.Atoi(strings.TrimSpace(result["seconds"]))
		if err != nil || v < 0 || v == current {
			return nil
		}
		return privacyRef.SetIdleTimeout("screensaver", v)
	})
}

func (m *Model) triggerPrivacyLockDialog() {
	if m.privacy.SetIdleTimeout == nil || !m.inPrivacyContent() {
		return
	}
	current := m.state.Privacy.LockTimeout
	privacyRef := m.privacy
	m.dialog = NewDialog("Lock screen timeout", []DialogFieldSpec{
		{Key: "seconds", Label: "Seconds (0 = disabled)", Value: strconv.Itoa(current)},
	}, func(result DialogResult) tea.Cmd {
		v, err := strconv.Atoi(strings.TrimSpace(result["seconds"]))
		if err != nil || v < 0 || v == current {
			return nil
		}
		return privacyRef.SetIdleTimeout("lock", v)
	})
}

func (m *Model) triggerPrivacyScreenOffDialog() {
	if m.privacy.SetIdleTimeout == nil || !m.inPrivacyContent() {
		return
	}
	current := m.state.Privacy.ScreenOffTimeout
	privacyRef := m.privacy
	m.dialog = NewDialog("Screen off timeout", []DialogFieldSpec{
		{Key: "seconds", Label: "Seconds (0 = disabled)", Value: strconv.Itoa(current)},
	}, func(result DialogResult) tea.Cmd {
		v, err := strconv.Atoi(strings.TrimSpace(result["seconds"]))
		if err != nil || v < 0 || v == current {
			return nil
		}
		return privacyRef.SetIdleTimeout("screen_off", v)
	})
}

func (m *Model) triggerPrivacyDNSTLSCycle() tea.Cmd {
	if m.privacy.SetDNSOverTLS == nil || !m.inPrivacyContent() {
		return nil
	}
	current := m.state.Privacy.DNSOverTLS
	next := "no"
	switch current {
	case "no":
		next = "opportunistic"
	case "opportunistic":
		next = "yes"
	case "yes":
		next = "no"
	}
	return m.privacy.SetDNSOverTLS(next)
}

func (m *Model) triggerPrivacyDNSSECCycle() tea.Cmd {
	if m.privacy.SetDNSSEC == nil || !m.inPrivacyContent() {
		return nil
	}
	current := m.state.Privacy.DNSSEC
	next := "no"
	switch current {
	case "no":
		next = "allow-downgrade"
	case "allow-downgrade":
		next = "yes"
	case "yes":
		next = "no"
	}
	return m.privacy.SetDNSSEC(next)
}

func (m *Model) triggerPrivacyFirewallToggle() tea.Cmd {
	if m.privacy.SetFirewall == nil || !m.inPrivacyContent() {
		return nil
	}
	if !m.state.Privacy.FirewallInstalled {
		m.notifyError("Privacy", "ufw is not installed")
		return nil
	}
	return m.privacy.SetFirewall(!m.state.Privacy.FirewallActive)
}

func (m *Model) triggerPrivacySSHToggle() tea.Cmd {
	if m.privacy.SetSSH == nil || !m.inPrivacyContent() {
		return nil
	}
	if !m.state.Privacy.SSHInstalled {
		m.notifyError("Privacy", "openssh is not installed")
		return nil
	}
	return m.privacy.SetSSH(!m.state.Privacy.SSHActive)
}

func (m *Model) triggerPrivacyClearRecent() {
	if m.privacy.ClearRecent == nil || !m.inPrivacyContent() {
		return
	}
	count := m.state.Privacy.RecentFileCount
	if count == 0 {
		return
	}
	privacyRef := m.privacy
	m.dialog = NewDialog(fmt.Sprintf("Clear %d recent files?", count), nil,
		func(_ DialogResult) tea.Cmd {
			return privacyRef.ClearRecent()
		})
}

func (m *Model) triggerPrivacyLocationToggle() tea.Cmd {
	if m.privacy.SetLocation == nil || !m.inPrivacyContent() {
		return nil
	}
	if !m.state.Privacy.LocationInstalled {
		m.notifyError("Privacy", "geoclue is not installed")
		return nil
	}
	return m.privacy.SetLocation(!m.state.Privacy.LocationActive)
}

func (m *Model) triggerPrivacyMACCycle() tea.Cmd {
	if m.privacy.SetMACRandom == nil || !m.inPrivacyContent() {
		return nil
	}
	current := m.state.Privacy.MACRandomization
	next := "disabled"
	switch current {
	case "disabled":
		next = "once"
	case "once":
		next = "network"
	case "network":
		next = "disabled"
	}
	return m.privacy.SetMACRandom(next)
}

func (m *Model) triggerPrivacyIndexerToggle() tea.Cmd {
	if m.privacy.SetIndexer == nil || !m.inPrivacyContent() {
		return nil
	}
	if !m.state.Privacy.IndexerInstalled {
		m.notifyError("Privacy", "localsearch/tracker is not installed")
		return nil
	}
	return m.privacy.SetIndexer(!m.state.Privacy.IndexerActive)
}

func (m *Model) triggerPrivacyCoredumpCycle() tea.Cmd {
	if m.privacy.SetCoredumpStorage == nil || !m.inPrivacyContent() {
		return nil
	}
	current := m.state.Privacy.CoredumpStorage
	next := "external"
	switch current {
	case "external":
		next = "journal"
	case "journal":
		next = "none"
	case "none":
		next = "external"
	}
	return m.privacy.SetCoredumpStorage(next)
}
