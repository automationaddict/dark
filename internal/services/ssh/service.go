package ssh

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Service is the top-level SSH surface darkd uses. It holds a single
// Backend (OpenSSH today) and aggregates a full Snapshot by calling
// every accessor on the backend, so the daemon's publisher just asks
// for Snapshot() and doesn't have to know which implementation is
// active.
//
// Methods are serialized by an internal mutex because the parsers
// and subprocess invocations are not concurrent-safe and there's no
// reason to make them so — command rates are user-driven.
type Service struct {
	mu        sync.Mutex
	backend   Backend
	sshDir    string
	available bool
}

// NewService detects openssh in PATH, resolves the user's ssh dir,
// and returns a Service bound to the OpenSSH backend. Callers that
// want a different backend can swap it with SetBackend after
// construction.
func NewService() *Service {
	sshDir := defaultSSHDir()
	s := &Service{
		sshDir:    sshDir,
		available: detectOpenSSH(),
	}
	s.backend = NewOpenSSHBackend(sshDir)
	return s
}

// SetBackend swaps the active backend. Not currently called — exists
// so phase 2 can wire a backend picker without refactoring the
// constructor.
func (s *Service) SetBackend(b Backend) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.backend = b
}

// BackendName returns the current backend's short identifier.
func (s *Service) BackendName() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend.Name()
}

// Backend returns the raw backend so daemon handlers can dispatch
// individual operations without going through a wrapper method on
// Service for every one.
func (s *Service) Backend() Backend {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.backend
}

// Snapshot aggregates every readable piece of SSH state the backend
// exposes. Errors on individual accessors are collected into
// LastError rather than failing the whole snapshot — a busted
// ~/.ssh/config shouldn't hide the key list or agent status.
func (s *Service) Snapshot() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap := Snapshot{
		InstalledOK: s.available,
		Backend:     s.backend.Name(),
	}
	if !s.available {
		snap.LastError = "openssh-client not installed — run `sudo pacman -S openssh`"
		return snap
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var errs []string
	collect := func(label string, err error) {
		if err != nil {
			errs = append(errs, label+": "+err.Error())
		}
	}

	keys, err := s.backend.ListKeys(ctx)
	collect("keys", err)
	snap.Keys = keys

	certs, err := s.backend.LoadCertificates(ctx)
	collect("certificates", err)
	snap.Certificates = certs

	agent, err := s.backend.AgentStatus(ctx)
	collect("agent", err)
	snap.Agent = agent

	// Cross-reference: mark each key as InAgent when its fingerprint
	// appears in the loaded-keys list.
	loadedSet := map[string]bool{}
	for _, f := range agent.LoadedFingerprints {
		loadedSet[f] = true
	}
	for i := range snap.Keys {
		if loadedSet[snap.Keys[i].Fingerprint] {
			snap.Keys[i].InAgent = true
		}
	}

	cfg, err := s.backend.LoadClientConfig(ctx)
	collect("client_config", err)
	snap.ClientConfig = cfg

	known, err := s.backend.LoadKnownHosts(ctx)
	collect("known_hosts", err)
	snap.KnownHosts = known

	authed, err := s.backend.LoadAuthorizedKeys(ctx)
	collect("authorized_keys", err)
	snap.AuthorizedKeys = authed

	server, err := s.backend.LoadServerConfig(ctx)
	collect("server_config", err)
	snap.ServerConfig = server

	if len(errs) > 0 {
		snap.LastError = strings.Join(errs, "; ")
	}
	return snap
}

// defaultSSHDir returns ~/.ssh, creating it with the correct mode
// (0700) if it doesn't exist. A failure here leaves the path unset
// so callers surface a clear error rather than writing to garbage.
func defaultSSHDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".ssh")
	if st, err := os.Stat(dir); err == nil && st.IsDir() {
		return dir
	}
	// Create with the expected mode. If this fails (read-only FS,
	// permission denied) callers will see parse errors from the
	// individual Load* methods — that's the right place to surface it.
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// detectOpenSSH returns true when ssh-keygen is on PATH. Every
// OpenSSH operation we perform goes through a subprocess, so this
// one check gates the entire backend.
func detectOpenSSH() bool {
	_, err := exec.LookPath("ssh-keygen")
	return err == nil
}
