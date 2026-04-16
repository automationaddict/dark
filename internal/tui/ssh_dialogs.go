package tui

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/services/clipboard"
)

// This file owns every dialog and key-handler the F4 SSH tab uses
// to drive the bus command surface. The split keeps ssh_view.go
// focused on rendering and lets each action trigger live next to
// its dialog construction.
//
// Every trigger follows the same shape: build a Dialog with
// NewDialog, set m.dialog, return nil. On submit, the dialog's
// callback returns a tea.Cmd that fires the injected action; the
// resulting SSHActionResultMsg is handled by model.go.

// handleSSHKey dispatches a key press when the F4 SSH tab is
// active and no dialog is open. Focus is on the inner sub-nav when
// SSHContentFocused is false and on the subsection's list when
// true. Returns (handled, tea.Cmd) so the caller can fall through
// on unhandled keys.
func (m *Model) handleSSHKey(key string) (bool, tea.Cmd) {
	if m.state.ActiveTab != core.TabF4 {
		return false, nil
	}
	if m.dialog != nil {
		return false, nil
	}
	// Outer sub-nav focus: enter drops into the detail pane.
	if !m.state.SSHContentFocused {
		switch key {
		case "enter":
			m.state.SSHContentFocused = true
			return true, nil
		}
		return false, nil
	}
	// Content-focused keys. esc backs out to the sub-nav.
	switch key {
	case "esc":
		m.state.SSHContentFocused = false
		return true, nil
	}
	switch m.state.SSHSelection.Subsection {
	case core.SSHSubKeys:
		return m.handleSSHKeysKey(key)
	case core.SSHSubAgent:
		return m.handleSSHAgentKey(key)
	case core.SSHSubClientConfig:
		return m.handleSSHHostsKey(key)
	case core.SSHSubKnownHosts:
		return m.handleSSHKnownHostsKey(key)
	case core.SSHSubAuthorizedKeys:
		return m.handleSSHAuthorizedKey(key)
	}
	return false, nil
}

// ─── Keys ───────────────────────────────────────────────────────

func (m *Model) handleSSHKeysKey(key string) (bool, tea.Cmd) {
	switch key {
	case "g":
		return true, m.openSSHGenerateKeyDialog()
	case "d":
		return true, m.openSSHDeleteKeyDialog()
	case "c":
		return true, m.copySSHPublicKey()
	case "a":
		return true, m.openSSHAgentAddDialog()
	}
	return false, nil
}

func (m *Model) openSSHGenerateKeyDialog() tea.Cmd {
	m.dialog = NewDialog("Generate SSH key", []DialogFieldSpec{
		{Key: "type", Label: "Type", Kind: DialogFieldSelect,
			Options: []string{"ed25519", "rsa", "ecdsa"}, Value: "ed25519"},
		{Key: "name", Label: "Filename (under ~/.ssh)", Kind: DialogFieldText, Value: "id_ed25519"},
		{Key: "comment", Label: "Comment", Kind: DialogFieldText},
		{Key: "passphrase", Label: "Passphrase (optional)", Kind: DialogFieldPassword},
	}, func(r DialogResult) tea.Cmd {
		if m.ssh.GenerateKey == nil {
			return nil
		}
		opts := core.SSHGenerateKeyOptions{
			Type:       r["type"],
			Path:       r["name"],
			Comment:    r["comment"],
			Passphrase: r["passphrase"],
		}
		return m.ssh.GenerateKey(opts)
	})
	return nil
}

func (m *Model) openSSHDeleteKeyDialog() tea.Cmd {
	if len(m.state.SSH.Keys) == 0 {
		return nil
	}
	idx := m.state.SSHKeysIdx
	if idx >= len(m.state.SSH.Keys) {
		idx = 0
	}
	k := m.state.SSH.Keys[idx]
	m.dialog = NewDialog("Delete "+sshKeyBaseName(k)+"?", nil, func(_ DialogResult) tea.Cmd {
		if m.ssh.DeleteKey == nil {
			return nil
		}
		return m.ssh.DeleteKey(k.Path)
	})
	return nil
}

// copySSHPublicKey copies the selected key's public half to the
// system clipboard. Uses the shared clipboard helper; failures
// surface through SSHActionError so the user sees what went wrong.
func (m *Model) copySSHPublicKey() tea.Cmd {
	if len(m.state.SSH.Keys) == 0 {
		return nil
	}
	idx := m.state.SSHKeysIdx
	if idx >= len(m.state.SSH.Keys) {
		idx = 0
	}
	k := m.state.SSH.Keys[idx]
	if k.PublicKey == "" {
		m.state.SSHActionError = "public key is empty"
		return nil
	}
	if err := clipboard.Copy(k.PublicKey); err != nil {
		m.state.SSHActionError = err.Error()
		m.notifyError("SSH", err.Error())
		return nil
	}
	m.state.SSHActionError = ""
	m.notifyInfo("SSH", "Public key copied to clipboard")
	return nil
}

// ─── Agent ──────────────────────────────────────────────────────

func (m *Model) handleSSHAgentKey(key string) (bool, tea.Cmd) {
	switch key {
	case "s":
		if m.ssh.AgentStart != nil {
			return true, m.ssh.AgentStart()
		}
	case "x":
		if m.ssh.AgentStop != nil {
			return true, m.ssh.AgentStop()
		}
	case "a":
		return true, m.openSSHAgentAddDialog()
	case "d":
		return true, m.openSSHAgentRemoveDialog()
	case "D":
		return true, m.openSSHAgentRemoveAllDialog()
	}
	return false, nil
}

func (m *Model) openSSHAgentAddDialog() tea.Cmd {
	if len(m.state.SSH.Keys) == 0 {
		m.state.SSHActionError = "no keys in ~/.ssh to add"
		return nil
	}
	paths := make([]string, 0, len(m.state.SSH.Keys))
	for _, k := range m.state.SSH.Keys {
		if k.Path != "" {
			paths = append(paths, k.Path)
		}
	}
	if len(paths) == 0 {
		m.state.SSHActionError = "no private keys found"
		return nil
	}
	m.dialog = NewDialog("Add key to agent", []DialogFieldSpec{
		{Key: "path", Label: "Key", Kind: DialogFieldSelect, Options: paths, Value: paths[0]},
		{Key: "passphrase", Label: "Passphrase (if required)", Kind: DialogFieldPassword},
		{Key: "ttl", Label: "TTL seconds (0 = permanent)", Kind: DialogFieldText, Value: "0"},
	}, func(r DialogResult) tea.Cmd {
		if m.ssh.AgentAdd == nil {
			return nil
		}
		ttl, _ := strconv.Atoi(r["ttl"])
		return m.ssh.AgentAdd(r["path"], r["passphrase"], ttl)
	})
	return nil
}

func (m *Model) openSSHAgentRemoveDialog() tea.Cmd {
	loaded := m.state.SSH.Agent.LoadedKeys
	if len(loaded) == 0 {
		m.state.SSHActionError = "no keys loaded in agent"
		return nil
	}
	idx := m.state.SSHAgentIdx
	if idx >= len(loaded) {
		idx = 0
	}
	lk := loaded[idx]
	label := lk.Fingerprint
	if lk.Comment != "" {
		label = lk.Comment + " (" + lk.Fingerprint + ")"
	}
	m.dialog = NewDialog("Remove "+label+" from agent?", nil, func(_ DialogResult) tea.Cmd {
		if m.ssh.AgentRemove == nil {
			return nil
		}
		return m.ssh.AgentRemove(lk.Fingerprint)
	})
	return nil
}

func (m *Model) openSSHAgentRemoveAllDialog() tea.Cmd {
	m.dialog = NewDialog("Remove all keys from agent?", nil, func(_ DialogResult) tea.Cmd {
		if m.ssh.AgentRemoveAll == nil {
			return nil
		}
		return m.ssh.AgentRemoveAll()
	})
	return nil
}

// ─── Client config hosts ────────────────────────────────────────

func (m *Model) handleSSHHostsKey(key string) (bool, tea.Cmd) {
	switch key {
	case "n":
		return true, m.openSSHHostDialog(core.SSHHostEntry{}, "New host")
	case "e":
		if len(m.state.SSH.ClientConfig.Hosts) == 0 {
			return true, nil
		}
		idx := m.state.SSHHostsIdx
		if idx >= len(m.state.SSH.ClientConfig.Hosts) {
			idx = 0
		}
		h := m.state.SSH.ClientConfig.Hosts[idx]
		return true, m.openSSHHostDialog(h, "Edit "+h.Pattern)
	case "d":
		return true, m.openSSHDeleteHostDialog()
	}
	return false, nil
}

func (m *Model) openSSHHostDialog(initial core.SSHHostEntry, title string) tea.Cmd {
	portStr := ""
	if initial.Port > 0 {
		portStr = strconv.Itoa(initial.Port)
	}
	m.dialog = NewDialog(title, []DialogFieldSpec{
		{Key: "pattern", Label: "Host pattern", Kind: DialogFieldText, Value: initial.Pattern},
		{Key: "hostname", Label: "HostName", Kind: DialogFieldText, Value: initial.HostName},
		{Key: "user", Label: "User", Kind: DialogFieldText, Value: initial.User},
		{Key: "port", Label: "Port", Kind: DialogFieldText, Value: portStr},
		{Key: "identity_file", Label: "IdentityFile", Kind: DialogFieldText, Value: initial.IdentityFile},
		{Key: "proxy_jump", Label: "ProxyJump", Kind: DialogFieldText, Value: initial.ProxyJump},
	}, func(r DialogResult) tea.Cmd {
		if m.ssh.SaveHost == nil {
			return nil
		}
		entry := core.SSHHostEntry{
			Pattern:      r["pattern"],
			HostName:     r["hostname"],
			User:         r["user"],
			IdentityFile: r["identity_file"],
			ProxyJump:    r["proxy_jump"],
			Extras:       initial.Extras, // preserve unknown directives
		}
		if p, err := strconv.Atoi(r["port"]); err == nil {
			entry.Port = p
		}
		if entry.Pattern == "" {
			m.state.SSHActionError = "host pattern is required"
			return nil
		}
		return m.ssh.SaveHost(entry)
	})
	return nil
}

func (m *Model) openSSHDeleteHostDialog() tea.Cmd {
	if len(m.state.SSH.ClientConfig.Hosts) == 0 {
		return nil
	}
	idx := m.state.SSHHostsIdx
	if idx >= len(m.state.SSH.ClientConfig.Hosts) {
		idx = 0
	}
	h := m.state.SSH.ClientConfig.Hosts[idx]
	m.dialog = NewDialog("Delete host "+h.Pattern+"?", nil, func(_ DialogResult) tea.Cmd {
		if m.ssh.DeleteHost == nil {
			return nil
		}
		return m.ssh.DeleteHost(h.Pattern)
	})
	return nil
}

// ─── Known hosts ────────────────────────────────────────────────

func (m *Model) handleSSHKnownHostsKey(key string) (bool, tea.Cmd) {
	switch key {
	case "s":
		return true, m.openSSHScanDialog()
	case "d":
		return true, m.openSSHRemoveKnownHostDialog()
	}
	return false, nil
}

func (m *Model) openSSHScanDialog() tea.Cmd {
	m.dialog = NewDialog("Scan host keys", []DialogFieldSpec{
		{Key: "hostname", Label: "Hostname or IP", Kind: DialogFieldText},
	}, func(r DialogResult) tea.Cmd {
		if m.ssh.ScanHost == nil {
			return nil
		}
		if r["hostname"] == "" {
			m.state.SSHActionError = "hostname required"
			return nil
		}
		return m.ssh.ScanHost(r["hostname"])
	})
	return nil
}

func (m *Model) openSSHRemoveKnownHostDialog() tea.Cmd {
	if len(m.state.SSH.KnownHosts) == 0 {
		return nil
	}
	idx := m.state.SSHKnownHostsIdx
	if idx >= len(m.state.SSH.KnownHosts) {
		idx = 0
	}
	kh := m.state.SSH.KnownHosts[idx]
	m.dialog = NewDialog("Remove known host "+kh.Hostname+"?", nil, func(_ DialogResult) tea.Cmd {
		if m.ssh.RemoveKnownHost == nil {
			return nil
		}
		return m.ssh.RemoveKnownHost(kh.Hostname)
	})
	return nil
}

// ─── Authorized keys ────────────────────────────────────────────

func (m *Model) handleSSHAuthorizedKey(key string) (bool, tea.Cmd) {
	switch key {
	case "n":
		return true, m.openSSHAddAuthorizedKeyDialog()
	case "d":
		return true, m.openSSHRemoveAuthorizedKeyDialog()
	}
	return false, nil
}

func (m *Model) openSSHAddAuthorizedKeyDialog() tea.Cmd {
	m.dialog = NewDialog("Add authorized key", []DialogFieldSpec{
		{Key: "line", Label: "Paste full key line (ssh-ed25519 AAA... comment)", Kind: DialogFieldText},
	}, func(r DialogResult) tea.Cmd {
		if m.ssh.AddAuthorizedKey == nil {
			return nil
		}
		if r["line"] == "" {
			m.state.SSHActionError = "key line required"
			return nil
		}
		return m.ssh.AddAuthorizedKey(r["line"])
	})
	return nil
}

func (m *Model) openSSHRemoveAuthorizedKeyDialog() tea.Cmd {
	if len(m.state.SSH.AuthorizedKeys) == 0 {
		return nil
	}
	idx := m.state.SSHAuthorizedIdx
	if idx >= len(m.state.SSH.AuthorizedKeys) {
		idx = 0
	}
	ak := m.state.SSH.AuthorizedKeys[idx]
	label := ak.Fingerprint
	if ak.Comment != "" {
		label = ak.Comment
	}
	m.dialog = NewDialog("Remove authorized key "+label+"?", nil, func(_ DialogResult) tea.Cmd {
		if m.ssh.RemoveAuthorizedKey == nil {
			return nil
		}
		return m.ssh.RemoveAuthorizedKey(ak.Fingerprint)
	})
	return nil
}
