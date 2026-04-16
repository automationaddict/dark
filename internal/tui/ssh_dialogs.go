package tui

import (
	"strconv"
	"strings"

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

// boolPtr / strPtr / intPtr / stringsPtr are convenience
// constructors for the pointer-semantics fields on
// core.SSHServerConfigEdit.
func boolPtr(v bool) *bool         { return &v }
func strPtr(v string) *string      { return &v }
func intPtr(v int) *int            { return &v }
func stringsPtr(v []string) *[]string { return &v }

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
	case core.SSHSubServerConfig:
		return m.handleSSHServerConfigKey(key)
	}
	return false, nil
}

func (m *Model) handleSSHServerConfigKey(key string) (bool, tea.Cmd) {
	switch key {
	case "e":
		return true, m.openSSHServerConfigDialog()
	case "R":
		return true, m.openSSHRestoreDialog("server_config", "/etc/ssh/sshd_config")
	}
	return false, nil
}

// openSSHServerConfigDialog shows a form pre-filled with the seven
// editable sshd_config directives. On submit it builds an edit
// payload that includes ONLY the fields whose value changed, so a
// no-op submit still rewrites the file but only touches directives
// the user actually modified. pkexec will prompt the user for auth
// and sshd -t validates before installing.
func (m *Model) openSSHServerConfigDialog() tea.Cmd {
	sc := m.state.SSH.ServerConfig
	if !sc.Readable {
		m.state.SSHActionError = "cannot read current sshd_config"
		return nil
	}
	currentPort := ""
	if sc.Port > 0 {
		currentPort = strconv.Itoa(sc.Port)
	}
	permitOptions := []string{"yes", "no", "prohibit-password", "forced-commands-only"}
	currentPermit := sc.PermitRootLogin
	if currentPermit == "" {
		currentPermit = "prohibit-password"
	}
	m.dialog = NewDialog("Edit sshd_config", []DialogFieldSpec{
		{Key: "port", Label: "Port", Kind: DialogFieldText, Value: currentPort},
		{Key: "permit_root_login", Label: "PermitRootLogin", Kind: DialogFieldSelect,
			Options: permitOptions, Value: currentPermit},
		{Key: "password_auth", Label: "PasswordAuthentication", Kind: DialogFieldSelect,
			Options: []string{"yes", "no"}, Value: boolLabel(sc.PasswordAuthentication, "yes", "no")},
		{Key: "pubkey_auth", Label: "PubkeyAuthentication", Kind: DialogFieldSelect,
			Options: []string{"yes", "no"}, Value: boolLabel(sc.PubkeyAuthentication, "yes", "no")},
		{Key: "x11", Label: "X11Forwarding", Kind: DialogFieldSelect,
			Options: []string{"yes", "no"}, Value: boolLabel(sc.X11Forwarding, "yes", "no")},
		{Key: "allow_users", Label: "AllowUsers (space-separated)", Kind: DialogFieldText,
			Value: strings.Join(sc.AllowUsers, " ")},
		{Key: "allow_groups", Label: "AllowGroups (space-separated)", Kind: DialogFieldText,
			Value: strings.Join(sc.AllowGroups, " ")},
	}, func(r DialogResult) tea.Cmd {
		if m.ssh.SaveServerConfig == nil {
			return nil
		}
		edit := core.SSHServerConfigEdit{}
		// Diff against the current snapshot so unchanged fields
		// stay as nil pointers and the splice logic skips them.
		if r["port"] != currentPort {
			if p, err := strconv.Atoi(r["port"]); err == nil && p > 0 {
				edit.Port = intPtr(p)
			}
		}
		if r["permit_root_login"] != sc.PermitRootLogin && r["permit_root_login"] != "" {
			edit.PermitRootLogin = strPtr(r["permit_root_login"])
		}
		newPasswordAuth := r["password_auth"] == "yes"
		if newPasswordAuth != sc.PasswordAuthentication {
			edit.PasswordAuthentication = boolPtr(newPasswordAuth)
		}
		newPubkeyAuth := r["pubkey_auth"] == "yes"
		if newPubkeyAuth != sc.PubkeyAuthentication {
			edit.PubkeyAuthentication = boolPtr(newPubkeyAuth)
		}
		newX11 := r["x11"] == "yes"
		if newX11 != sc.X11Forwarding {
			edit.X11Forwarding = boolPtr(newX11)
		}
		newAllowUsers := splitWords(r["allow_users"])
		if !stringSliceEqual(newAllowUsers, sc.AllowUsers) {
			edit.AllowUsers = stringsPtr(newAllowUsers)
		}
		newAllowGroups := splitWords(r["allow_groups"])
		if !stringSliceEqual(newAllowGroups, sc.AllowGroups) {
			edit.AllowGroups = stringsPtr(newAllowGroups)
		}
		// Nothing changed — no-op.
		if edit.Port == nil && edit.PermitRootLogin == nil &&
			edit.PasswordAuthentication == nil && edit.PubkeyAuthentication == nil &&
			edit.X11Forwarding == nil && edit.AllowUsers == nil && edit.AllowGroups == nil {
			return nil
		}
		return m.ssh.SaveServerConfig(edit)
	})
	return nil
}

// splitWords normalizes a whitespace-separated string into a slice,
// trimming empty tokens so accidental double-spaces don't produce
// garbage entries in the output config.
func splitWords(s string) []string {
	var out []string
	for _, w := range strings.Fields(s) {
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

// stringSliceEqual compares two string slices for set-independent
// equality. sshd_config preserves order so a strict element-wise
// compare is what we want here.
func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
	case "p":
		return true, m.openSSHChangePassphraseDialog()
	}
	return false, nil
}

// openSSHChangePassphraseDialog wraps the ssh-keygen -p flow in a
// two-field form. An empty new passphrase means "remove the
// passphrase" which is how ssh-keygen itself treats `-N ''`.
func (m *Model) openSSHChangePassphraseDialog() tea.Cmd {
	if len(m.state.SSH.Keys) == 0 {
		return nil
	}
	idx := m.state.SSHKeysIdx
	if idx >= len(m.state.SSH.Keys) {
		idx = 0
	}
	k := m.state.SSH.Keys[idx]
	if k.Path == "" {
		m.state.SSHActionError = "orphan public key — nothing to re-encrypt"
		return nil
	}
	m.dialog = NewDialog("Change passphrase for "+sshKeyBaseName(k), []DialogFieldSpec{
		{Key: "old", Label: "Current passphrase (empty if none)", Kind: DialogFieldPassword},
		{Key: "new", Label: "New passphrase (empty to remove)", Kind: DialogFieldPassword},
	}, func(r DialogResult) tea.Cmd {
		if m.ssh.ChangePassphrase == nil {
			return nil
		}
		return m.ssh.ChangePassphrase(k.Path, r["old"], r["new"])
	})
	return nil
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
	case "R":
		return true, m.openSSHRestoreDialog("client_config", "~/.ssh/config")
	}
	return false, nil
}

// openSSHRestoreDialog shows a confirmation dialog for rolling a
// file back to its `.bak` sibling. The target string is the same
// one the bus subject takes so the TUI doesn't have to know about
// service package types.
func (m *Model) openSSHRestoreDialog(target, label string) tea.Cmd {
	m.dialog = NewDialog("Restore "+label+" from .bak?", nil, func(_ DialogResult) tea.Cmd {
		if m.ssh.RestoreBackup == nil {
			return nil
		}
		return m.ssh.RestoreBackup(target)
	})
	return nil
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
	case "R":
		return true, m.openSSHRestoreDialog("authorized_keys", "~/.ssh/authorized_keys")
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
