// Package ssh is dark's SSH management service. It wraps the OpenSSH
// toolchain (ssh-keygen, ssh-add, ssh-agent, ssh-keyscan, sshd) and
// exposes a Backend-typed surface so alternative implementations
// (1Password CLI, gnome-keyring, future agents) can slot in without
// touching the daemon's command handlers or the TUI.
//
// The package owns parsing and writing of the files under ~/.ssh and
// the read-only parse of /etc/ssh/sshd_config. Every write is atomic
// (temp file + rename) and every destructive operation assumes the
// caller already confirmed intent — the TUI handles user prompts.
package ssh

import "time"

// Snapshot is the full SSH state dark publishes on `dark.ssh.snapshot`
// and replies with on `dark.cmd.ssh.snapshot`. Every field is a plain
// value type so the TUI can mirror the shape into core without taking
// a build-time dependency on this package.
type Snapshot struct {
	InstalledOK    bool           `json:"installed_ok"`
	Backend        string         `json:"backend"`
	Keys           []Key          `json:"keys,omitempty"`
	Agent          AgentStatus    `json:"agent"`
	ClientConfig   ClientConfig   `json:"client_config"`
	KnownHosts     []KnownHost    `json:"known_hosts,omitempty"`
	AuthorizedKeys []AuthorizedKey `json:"authorized_keys,omitempty"`
	ServerConfig   ServerConfig   `json:"server_config"`
	LastError      string         `json:"last_error,omitempty"`
}

// Key describes one SSH key pair living under ~/.ssh. Dark never
// stores or displays the private key contents — Path is the file
// path only, used to invoke ssh-keygen and ssh-add subprocesses.
type Key struct {
	Path          string    `json:"path"`           // ~/.ssh/id_ed25519
	PublicPath    string    `json:"public_path"`    // ~/.ssh/id_ed25519.pub
	Type          string    `json:"type"`           // "ed25519" | "rsa" | "ecdsa" | "dsa"
	Bits          int       `json:"bits,omitempty"` // rsa/ecdsa only
	Comment       string    `json:"comment,omitempty"`
	Fingerprint   string    `json:"fingerprint"` // SHA256:... format
	HasPassphrase bool      `json:"has_passphrase"`
	PublicKey     string    `json:"public_key,omitempty"` // the .pub contents (safe to expose)
	ModTime       time.Time `json:"mod_time"`
	InAgent       bool      `json:"in_agent"` // cross-referenced with agent state
}

// AgentStatus summarizes the currently-running ssh-agent — whether
// dark is managing it via systemd, its socket, and the keys it holds.
type AgentStatus struct {
	Running            bool        `json:"running"`
	SystemdManaged     bool        `json:"systemd_managed"`
	SystemdUnitExists  bool        `json:"systemd_unit_exists"`
	SocketPath         string      `json:"socket_path,omitempty"`
	Pid                int         `json:"pid,omitempty"`
	LoadedKeys         []LoadedKey `json:"loaded_keys,omitempty"`
	LoadedFingerprints []string    `json:"loaded_fingerprints,omitempty"`
}

// LoadedKey is one entry from `ssh-add -l`. Only the fingerprint is
// guaranteed; comment and type may be empty for keys loaded outside
// of dark's control.
type LoadedKey struct {
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment,omitempty"`
	Type        string `json:"type,omitempty"`
}

// ClientConfig mirrors ~/.ssh/config as a list of Host blocks plus
// the raw path so the TUI can show where edits will land.
type ClientConfig struct {
	Path  string      `json:"path"`
	Hosts []HostEntry `json:"hosts,omitempty"`
}

// HostEntry is one Host block from ~/.ssh/config. LineStart/LineEnd
// let SaveHostEntry splice an updated block back into place without
// rewriting the whole file. Extras preserves every directive the
// typed fields don't cover so round-trips don't lose data.
type HostEntry struct {
	Pattern              string            `json:"pattern"` // the Host line value
	HostName             string            `json:"host_name,omitempty"`
	User                 string            `json:"user,omitempty"`
	Port                 int               `json:"port,omitempty"`
	IdentityFile         string            `json:"identity_file,omitempty"`
	ForwardAgent         bool              `json:"forward_agent,omitempty"`
	ProxyJump            string            `json:"proxy_jump,omitempty"`
	StrictHostKeyChecking string           `json:"strict_host_key_checking,omitempty"`
	Extras               map[string]string `json:"extras,omitempty"`
	LineStart            int               `json:"line_start"`
	LineEnd              int               `json:"line_end"`
}

// KnownHost is one entry from ~/.ssh/known_hosts.
type KnownHost struct {
	Hostname    string `json:"hostname"`
	KeyType     string `json:"key_type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment,omitempty"`
	LineNumber  int    `json:"line_number"`
}

// AuthorizedKey is one entry from ~/.ssh/authorized_keys — a key the
// user has pre-authorized for incoming connections to this machine.
type AuthorizedKey struct {
	Options     string `json:"options,omitempty"` // command=..., no-pty, etc.
	KeyType     string `json:"key_type"`
	Fingerprint string `json:"fingerprint"`
	Comment     string `json:"comment,omitempty"`
	LineNumber  int    `json:"line_number"`
}

// ServerConfig is a parsed view of /etc/ssh/sshd_config. Readable is
// false when the file can't be opened (missing / permission denied).
// RawLines preserves the file verbatim so phase-2 edits can round-trip
// cleanly.
type ServerConfig struct {
	Path                   string   `json:"path"`
	Readable               bool     `json:"readable"`
	Port                   int      `json:"port,omitempty"`
	PermitRootLogin        string   `json:"permit_root_login,omitempty"`
	PasswordAuthentication bool     `json:"password_authentication"`
	PubkeyAuthentication   bool     `json:"pubkey_authentication"`
	X11Forwarding          bool     `json:"x11_forwarding,omitempty"`
	AllowUsers             []string `json:"allow_users,omitempty"`
	AllowGroups            []string `json:"allow_groups,omitempty"`
	RawLines               []string `json:"-"` // not serialized — too big for the bus
	ParseError             string   `json:"parse_error,omitempty"`
}

// GenerateKeyOptions carries the parameters for ssh-keygen. Empty
// Passphrase means no passphrase (ssh-keygen -N ''). Bits is ignored
// for ed25519 (which has a fixed size) and defaults to 3072 for rsa,
// 256 for ecdsa when zero.
type GenerateKeyOptions struct {
	Type       string `json:"type"`
	Bits       int    `json:"bits,omitempty"`
	Path       string `json:"path"`
	Comment    string `json:"comment,omitempty"`
	Passphrase string `json:"passphrase,omitempty"`
}
