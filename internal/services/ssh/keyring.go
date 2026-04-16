package ssh

import (
	"context"
	"os"
	"strings"
)

// GnomeKeyringBackend is a thin wrapper over the OpenSSH backend
// that's only "available" when gnome-keyring-daemon is running with
// its ssh component enabled. Keys, configs, and files still live in
// ~/.ssh and still get managed by the standard ssh-* tools — what
// gnome-keyring provides is the agent implementation, and that's
// ssh-agent-protocol compatible. So every operation can delegate to
// the embedded OpenSSH backend unchanged; the only thing that
// changes is who holds the `SSH_AUTH_SOCK` on the other end.
//
// The backend exists so F5 and the MCP catalog can see gnome-keyring
// as a named implementation rather than it being invisible behind a
// "current backend = openssh" label. When GnomeKeyringActive returns
// true, Service could swap in this backend via SetBackend (phase 3
// backend picker); today the service always picks OpenSSH and this
// type is here as a seam.
type GnomeKeyringBackend struct {
	*OpenSSHBackend
}

// NewGnomeKeyringBackend constructs the wrapper. Callers should
// check Available() before using it — when gnome-keyring isn't
// running the constructor still succeeds, but operations may try
// to talk to a socket that isn't there.
func NewGnomeKeyringBackend(sshDir string) Backend {
	return &GnomeKeyringBackend{OpenSSHBackend: NewOpenSSHBackend(sshDir)}
}

// Name identifies this backend distinctly from OpenSSH in the F5
// browser and the bus catalog.
func (b *GnomeKeyringBackend) Name() string { return "keyring" }

// Available returns true when the gnome-keyring ssh component is
// the live SSH_AUTH_SOCK provider. Detection is heuristic: the
// gnome-keyring agent socket lives under the user's runtime dir at
// `$XDG_RUNTIME_DIR/keyring/ssh` on standard installs, so if
// SSH_AUTH_SOCK points there we're confident. We also accept the
// legacy `$XDG_RUNTIME_DIR/keyring-*/ssh` pattern some older setups
// still use.
func (b *GnomeKeyringBackend) Available() bool {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock == "" {
		return false
	}
	return strings.Contains(sock, "/keyring/") ||
		strings.Contains(sock, "/keyring-")
}

// AgentStatus overrides the embedded OpenSSH implementation so the
// reported systemd state reflects that dark didn't install the
// unit — gnome-keyring manages its own agent lifecycle via the
// user session. The loaded-keys list still comes from `ssh-add -l`
// which talks to whatever socket SSH_AUTH_SOCK points at.
func (b *GnomeKeyringBackend) AgentStatus(ctx context.Context) (AgentStatus, error) {
	st, err := b.OpenSSHBackend.AgentStatus(ctx)
	if err != nil {
		return st, err
	}
	// gnome-keyring's agent is session-managed, not systemd-managed
	// by dark. Clear the systemd flags so the UI doesn't suggest
	// dark can start/stop it.
	st.SystemdManaged = false
	st.SystemdUnitExists = false
	return st, nil
}

// detectGnomeKeyringActive is a free function used by the service
// to decide whether to log an informational message about gnome-
// keyring being present. Today nothing wires the keyring backend in
// automatically — it's here for the phase-3 backend picker.
func detectGnomeKeyringActive() bool {
	sock := os.Getenv("SSH_AUTH_SOCK")
	return strings.Contains(sock, "/keyring/") ||
		strings.Contains(sock, "/keyring-")
}
