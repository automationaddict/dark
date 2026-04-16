package core

import "time"

// This file mirrors the internal/services/ssh snapshot types into
// core so the TUI package never imports the service directly. Every
// F1/F2/F3 service follows this same pattern — the translation
// happens in cmd/dark/*_actions.go at bus-reply decoding time.

// SSHSelectionKind identifies the top-level row in F4's outer
// sidebar. Today SSH is the only entry; future power-user features
// extend this enum.
type SSHSelectionKind int

const (
	SSHSelKindSSH SSHSelectionKind = iota
)

// SSHSubsection identifies the currently-active sub-nav tab inside
// the SSH content pane. The six values correspond to the six
// sub-sections described in the plan: keys, agent, client config,
// known hosts, authorized keys, and server config.
type SSHSubsection int

const (
	SSHSubKeys SSHSubsection = iota
	SSHSubAgent
	SSHSubClientConfig
	SSHSubKnownHosts
	SSHSubAuthorizedKeys
	SSHSubServerConfig
)

// SSHSelection is the combined outer + sub-section selection state.
// Persisted across tab switches so returning to F4 lands on the same
// view the user last used.
type SSHSelection struct {
	Outer      SSHSelectionKind
	Subsection SSHSubsection
}

// SSHSnapshot mirrors services/ssh.Snapshot. Only the fields the TUI
// actually renders are copied across — anything the UI doesn't need
// (raw sshd_config lines, for instance) stays in the service type.
type SSHSnapshot struct {
	InstalledOK    bool
	Backend        string
	Keys           []SSHKey
	Certificates   []SSHCertificate
	Agent          SSHAgentStatus
	ClientConfig   SSHClientConfig
	KnownHosts     []SSHKnownHost
	AuthorizedKeys []SSHAuthorizedKey
	ServerConfig   SSHServerConfig
	LastError      string
}

// SSHCertificate is the TUI-side view of a parsed -cert.pub file.
type SSHCertificate struct {
	CertPath       string
	Type           string // "user" or "host"
	KeyID          string
	Serial         string
	ValidAfter     time.Time
	ValidBefore    time.Time
	Principals     []string
	CAFingerprint  string
	KeyFingerprint string
	Expired        bool
}

// SSHKey is the TUI-side view of one SSH key pair. Private key
// contents are never copied — Path is only used when dispatching
// mutations back through the bus.
type SSHKey struct {
	Path          string
	PublicPath    string
	Type          string
	Bits          int
	Comment       string
	Fingerprint   string
	PublicKey     string
	HasPassphrase bool
	InAgent       bool
	ModTime       time.Time
}

// SSHAgentStatus summarizes the currently-running ssh-agent state.
type SSHAgentStatus struct {
	Running            bool
	SystemdManaged     bool
	SystemdUnitExists  bool
	Forwarded          bool
	SocketPath         string
	Pid                int
	LoadedKeys         []SSHLoadedKey
}

// SSHLoadedKey is one entry from ssh-add -l.
type SSHLoadedKey struct {
	Fingerprint string
	Comment     string
	Type        string
}

// SSHClientConfig mirrors the parsed ~/.ssh/config.
type SSHClientConfig struct {
	Path  string
	Hosts []SSHHostEntry
}

// SSHHostEntry is one Host block from ~/.ssh/config.
type SSHHostEntry struct {
	Pattern               string
	HostName              string
	User                  string
	Port                  int
	IdentityFile          string
	ForwardAgent          bool
	ProxyJump             string
	StrictHostKeyChecking string
	Extras                map[string]string
}

// SSHKnownHost is one known_hosts entry.
type SSHKnownHost struct {
	Hostname    string
	KeyType     string
	Fingerprint string
	Comment     string
}

// SSHAuthorizedKey is one authorized_keys entry.
type SSHAuthorizedKey struct {
	Options     string
	KeyType     string
	Fingerprint string
	Comment     string
}

// SSHServerConfig mirrors the parsed /etc/ssh/sshd_config. Read-only
// in v1 — the TUI renders the phase-2 banner next to this pane.
type SSHServerConfig struct {
	Path                   string
	Readable               bool
	Port                   int
	PermitRootLogin        string
	PasswordAuthentication bool
	PubkeyAuthentication   bool
	X11Forwarding          bool
	AllowUsers             []string
	AllowGroups            []string
	ParseError             string
}

// SSHGenerateKeyOptions mirrors services/ssh.GenerateKeyOptions for
// the client→daemon request payload.
type SSHGenerateKeyOptions struct {
	Type       string `json:"type,omitempty"`
	Bits       int    `json:"bits,omitempty"`
	Path       string `json:"path,omitempty"`
	Comment    string `json:"comment,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}

// SSHServerConfigEdit mirrors services/ssh.ServerConfigEdit for the
// client→daemon request payload. Pointer fields preserve the
// "omit = leave untouched" semantics the backend splice relies on.
type SSHServerConfigEdit struct {
	Port                   *int      `json:"port,omitempty"`
	PermitRootLogin        *string   `json:"permit_root_login,omitempty"`
	PasswordAuthentication *bool     `json:"password_authentication,omitempty"`
	PubkeyAuthentication   *bool     `json:"pubkey_authentication,omitempty"`
	X11Forwarding          *bool     `json:"x11_forwarding,omitempty"`
	AllowUsers             *[]string `json:"allow_users,omitempty"`
	AllowGroups            *[]string `json:"allow_groups,omitempty"`
}

// SetSSH replaces the cached SSH snapshot and resets any transient
// action-error state so the user sees a clean slate after a
// successful refresh.
func (s *State) SetSSH(snap SSHSnapshot) {
	s.SSH = snap
	s.SSHLoaded = true
	s.SSHActionError = ""

	// Clamp the inner-list selections in case the underlying lists
	// shrank since the last snapshot.
	if s.SSHKeysIdx >= len(snap.Keys) {
		s.SSHKeysIdx = 0
	}
	if s.SSHAgentIdx >= len(snap.Agent.LoadedKeys) {
		s.SSHAgentIdx = 0
	}
	if s.SSHHostsIdx >= len(snap.ClientConfig.Hosts) {
		s.SSHHostsIdx = 0
	}
	if s.SSHKnownHostsIdx >= len(snap.KnownHosts) {
		s.SSHKnownHostsIdx = 0
	}
	if s.SSHAuthorizedIdx >= len(snap.AuthorizedKeys) {
		s.SSHAuthorizedIdx = 0
	}
}

// MoveSSHSubsection walks the inner sub-nav up (-1) or down (+1),
// clamping at the ends. The subsections render as a vertical list
// in the content pane's left column — same behavior as every other
// sidebar in dark.
func (s *State) MoveSSHSubsection(delta int) {
	const total = 6
	next := int(s.SSHSelection.Subsection) + delta
	if next < 0 {
		next = 0
	}
	if next >= total {
		next = total - 1
	}
	s.SSHSelection.Subsection = SSHSubsection(next)
}

// MoveSSHInner walks the currently-active list (keys / agent /
// hosts / known_hosts / authorized_keys) by delta, clamping at the
// list's bounds.
func (s *State) MoveSSHInner(delta int) {
	switch s.SSHSelection.Subsection {
	case SSHSubKeys:
		s.SSHKeysIdx = clampSSH(s.SSHKeysIdx+delta, 0, len(s.SSH.Keys)-1)
	case SSHSubAgent:
		s.SSHAgentIdx = clampSSH(s.SSHAgentIdx+delta, 0, len(s.SSH.Agent.LoadedKeys)-1)
	case SSHSubClientConfig:
		s.SSHHostsIdx = clampSSH(s.SSHHostsIdx+delta, 0, len(s.SSH.ClientConfig.Hosts)-1)
	case SSHSubKnownHosts:
		s.SSHKnownHostsIdx = clampSSH(s.SSHKnownHostsIdx+delta, 0, len(s.SSH.KnownHosts)-1)
	case SSHSubAuthorizedKeys:
		s.SSHAuthorizedIdx = clampSSH(s.SSHAuthorizedIdx+delta, 0, len(s.SSH.AuthorizedKeys)-1)
	}
}

func clampSSH(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
