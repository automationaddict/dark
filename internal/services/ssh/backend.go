package ssh

import "context"

// Backend is the pluggable strategy for SSH key and agent management.
// v1 ships a single concrete implementation (OpenSSH via the standard
// ssh-* command-line tools). Stubs for OnePassword and GnomeKeyring
// live alongside so the seam is visible — they return ErrUnsupported
// from every method until phase 2 wires them up.
type Backend interface {
	Name() string
	Available() bool

	// Keys
	ListKeys(ctx context.Context) ([]Key, error)
	GenerateKey(ctx context.Context, opts GenerateKeyOptions) (Key, error)
	DeleteKey(ctx context.Context, path string) error
	ChangePassphrase(ctx context.Context, path, oldPass, newPass string) error

	// Agent
	AgentStatus(ctx context.Context) (AgentStatus, error)
	AgentStart(ctx context.Context) error
	AgentStop(ctx context.Context) error
	AgentAdd(ctx context.Context, keyPath, passphrase string, lifetimeSeconds int) error
	AgentRemove(ctx context.Context, fingerprint string) error
	AgentRemoveAll(ctx context.Context) error

	// Client config
	LoadClientConfig(ctx context.Context) (ClientConfig, error)
	SaveHostEntry(ctx context.Context, entry HostEntry) error
	DeleteHostEntry(ctx context.Context, pattern string) error

	// Known hosts
	LoadKnownHosts(ctx context.Context) ([]KnownHost, error)
	ScanHost(ctx context.Context, hostname string) ([]KnownHost, error)
	RemoveKnownHost(ctx context.Context, hostname string) error

	// Authorized keys
	LoadAuthorizedKeys(ctx context.Context) ([]AuthorizedKey, error)
	AddAuthorizedKey(ctx context.Context, line string) error
	RemoveAuthorizedKey(ctx context.Context, fingerprint string) error

	// Server config — read with LoadServerConfig, edit a subset of
	// directives with SaveServerConfig. The save path runs sshd -t
	// for validation before installing the new file and keeps a
	// .bak of the previous contents (both via dark-helper under
	// pkexec). A failed validation returns the sshd -t output
	// verbatim so the user sees which line was rejected.
	LoadServerConfig(ctx context.Context) (ServerConfig, error)
	SaveServerConfig(ctx context.Context, edit ServerConfigEdit) error

	// RestoreBackup rolls one of the rewritable files back to its
	// `.bak` sibling. Writes go through the same pkexec helper
	// path as SaveServerConfig when the target is root-owned.
	RestoreBackup(ctx context.Context, target RestoreTarget) error

	// SSH CA operations — sign keys with a CA and list certificates.
	SignKey(ctx context.Context, opts SignKeyOptions) (Certificate, error)
	LoadCertificates(ctx context.Context) ([]Certificate, error)
}

// ErrUnsupported is returned by stub backends for every method until
// phase 2 fills them in. The daemon handlers surface this unchanged
// so the TUI can show a "not implemented" message instead of a
// cryptic error string.
type errUnsupported struct{ backend, op string }

func (e errUnsupported) Error() string {
	return e.backend + ": " + e.op + " not implemented in v1 — see phase 2 plan"
}

// unsupportedBackend is embedded into the stub backends below so they
// can return the same errUnsupported for every method without
// duplicating error construction.
type unsupportedBackend struct{ name string }

func (b unsupportedBackend) Name() string  { return b.name }
func (b unsupportedBackend) Available() bool { return false }

func (b unsupportedBackend) ListKeys(ctx context.Context) ([]Key, error) {
	return nil, errUnsupported{b.name, "ListKeys"}
}
func (b unsupportedBackend) GenerateKey(ctx context.Context, opts GenerateKeyOptions) (Key, error) {
	return Key{}, errUnsupported{b.name, "GenerateKey"}
}
func (b unsupportedBackend) DeleteKey(ctx context.Context, path string) error {
	return errUnsupported{b.name, "DeleteKey"}
}
func (b unsupportedBackend) ChangePassphrase(ctx context.Context, path, oldPass, newPass string) error {
	return errUnsupported{b.name, "ChangePassphrase"}
}
func (b unsupportedBackend) AgentStatus(ctx context.Context) (AgentStatus, error) {
	return AgentStatus{}, errUnsupported{b.name, "AgentStatus"}
}
func (b unsupportedBackend) AgentStart(ctx context.Context) error {
	return errUnsupported{b.name, "AgentStart"}
}
func (b unsupportedBackend) AgentStop(ctx context.Context) error {
	return errUnsupported{b.name, "AgentStop"}
}
func (b unsupportedBackend) AgentAdd(ctx context.Context, keyPath, passphrase string, lifetimeSeconds int) error {
	return errUnsupported{b.name, "AgentAdd"}
}
func (b unsupportedBackend) AgentRemove(ctx context.Context, fingerprint string) error {
	return errUnsupported{b.name, "AgentRemove"}
}
func (b unsupportedBackend) AgentRemoveAll(ctx context.Context) error {
	return errUnsupported{b.name, "AgentRemoveAll"}
}
func (b unsupportedBackend) LoadClientConfig(ctx context.Context) (ClientConfig, error) {
	return ClientConfig{}, errUnsupported{b.name, "LoadClientConfig"}
}
func (b unsupportedBackend) SaveHostEntry(ctx context.Context, entry HostEntry) error {
	return errUnsupported{b.name, "SaveHostEntry"}
}
func (b unsupportedBackend) DeleteHostEntry(ctx context.Context, pattern string) error {
	return errUnsupported{b.name, "DeleteHostEntry"}
}
func (b unsupportedBackend) LoadKnownHosts(ctx context.Context) ([]KnownHost, error) {
	return nil, errUnsupported{b.name, "LoadKnownHosts"}
}
func (b unsupportedBackend) ScanHost(ctx context.Context, hostname string) ([]KnownHost, error) {
	return nil, errUnsupported{b.name, "ScanHost"}
}
func (b unsupportedBackend) RemoveKnownHost(ctx context.Context, hostname string) error {
	return errUnsupported{b.name, "RemoveKnownHost"}
}
func (b unsupportedBackend) LoadAuthorizedKeys(ctx context.Context) ([]AuthorizedKey, error) {
	return nil, errUnsupported{b.name, "LoadAuthorizedKeys"}
}
func (b unsupportedBackend) AddAuthorizedKey(ctx context.Context, line string) error {
	return errUnsupported{b.name, "AddAuthorizedKey"}
}
func (b unsupportedBackend) RemoveAuthorizedKey(ctx context.Context, fingerprint string) error {
	return errUnsupported{b.name, "RemoveAuthorizedKey"}
}
func (b unsupportedBackend) LoadServerConfig(ctx context.Context) (ServerConfig, error) {
	return ServerConfig{}, errUnsupported{b.name, "LoadServerConfig"}
}
func (b unsupportedBackend) SaveServerConfig(ctx context.Context, edit ServerConfigEdit) error {
	return errUnsupported{b.name, "SaveServerConfig"}
}
func (b unsupportedBackend) RestoreBackup(ctx context.Context, target RestoreTarget) error {
	return errUnsupported{b.name, "RestoreBackup"}
}
func (b unsupportedBackend) SignKey(ctx context.Context, opts SignKeyOptions) (Certificate, error) {
	return Certificate{}, errUnsupported{b.name, "SignKey"}
}
func (b unsupportedBackend) LoadCertificates(ctx context.Context) ([]Certificate, error) {
	return nil, errUnsupported{b.name, "LoadCertificates"}
}

// OnePasswordBackend lives in onepassword.go as a real implementation.
// GnomeKeyringBackend lives in keyring.go.
