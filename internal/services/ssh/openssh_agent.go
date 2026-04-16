package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// AgentStatus reads the currently-running ssh-agent from the
// environment. `ssh-add -l` is the source of truth for the key list;
// SSH_AUTH_SOCK resolves the socket and we call `systemctl --user`
// to report whether dark is the one managing the agent.
func (b *OpenSSHBackend) AgentStatus(ctx context.Context) (AgentStatus, error) {
	st := AgentStatus{
		SocketPath:        os.Getenv("SSH_AUTH_SOCK"),
		SystemdUnitExists: agentUnitExists(),
	}
	if st.SocketPath != "" {
		if _, err := os.Stat(st.SocketPath); err == nil {
			st.Running = true
		}
	}
	st.SystemdManaged = st.SystemdUnitExists && systemctlUserActive("ssh-agent.service")
	if st.SystemdManaged {
		st.Running = true
	}
	if !st.Running {
		return st, nil
	}
	// Ask the agent what keys it holds.
	out, err := runCapture(ctx, "ssh-add", "-l")
	if err != nil {
		// `ssh-add -l` returns exit code 1 when the agent has no
		// identities — that's not an error for us, just an empty
		// list. Other exit codes (2 = can't connect) surface.
		if !strings.Contains(out, "no identities") && !strings.Contains(err.Error(), "exit status 1") {
			return st, fmt.Errorf("ssh-add -l: %s", strings.TrimSpace(out))
		}
		return st, nil
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<bits> <fingerprint> <comment> (<type>)"
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		lk := LoadedKey{Fingerprint: parts[1]}
		last := parts[len(parts)-1]
		if strings.HasPrefix(last, "(") && strings.HasSuffix(last, ")") {
			lk.Type = strings.ToLower(strings.Trim(last, "()"))
			parts = parts[:len(parts)-1]
		}
		if len(parts) > 2 {
			lk.Comment = strings.Join(parts[2:], " ")
		}
		st.LoadedKeys = append(st.LoadedKeys, lk)
		st.LoadedFingerprints = append(st.LoadedFingerprints, lk.Fingerprint)
	}
	if pid := os.Getenv("SSH_AGENT_PID"); pid != "" {
		if n, err := strconv.Atoi(pid); err == nil {
			st.Pid = n
		}
	}
	return st, nil
}

// AgentStart brings up dark's managed ssh-agent systemd user unit,
// writing the unit file on first invocation. Idempotent — calling
// when the unit is already running is a no-op.
func (b *OpenSSHBackend) AgentStart(ctx context.Context) error {
	if err := ensureAgentUnit(); err != nil {
		return fmt.Errorf("install agent unit: %w", err)
	}
	if _, err := runCapture(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
		return fmt.Errorf("systemctl daemon-reload: %w", err)
	}
	if _, err := runCapture(ctx, "systemctl", "--user", "start", "ssh-agent.service"); err != nil {
		return fmt.Errorf("systemctl start: %w", err)
	}
	return nil
}

// AgentStop stops the managed agent. Leaves the unit file in place
// so a future AgentStart doesn't have to rewrite it.
func (b *OpenSSHBackend) AgentStop(ctx context.Context) error {
	if _, err := runCapture(ctx, "systemctl", "--user", "stop", "ssh-agent.service"); err != nil {
		return fmt.Errorf("systemctl stop: %w", err)
	}
	return nil
}

// AgentAdd loads a key into the agent. When lifetimeSeconds is
// positive, the agent auto-expires the key after that many seconds
// (ssh-add -t). An empty passphrase passes through as the value ''
// which ssh-add interprets as "no passphrase expected".
//
// Passphrases are piped via SSH_ASKPASS + a helper script so the
// value never lands on the command line. For keys without a
// passphrase we skip the helper entirely.
func (b *OpenSSHBackend) AgentAdd(ctx context.Context, keyPath, passphrase string, lifetimeSeconds int) error {
	if keyPath == "" {
		return fmt.Errorf("missing key path")
	}
	args := []string{}
	if lifetimeSeconds > 0 {
		args = append(args, "-t", strconv.Itoa(lifetimeSeconds))
	}
	args = append(args, keyPath)

	cmd := exec.CommandContext(ctx, "ssh-add", args...)
	cmd.Env = os.Environ()

	// If a passphrase is required, route it through SSH_ASKPASS. We
	// write the passphrase to a FIFO and point SSH_ASKPASS at a tiny
	// helper that cat's the FIFO. Simpler fallback: write a
	// single-use script to a tmp file that echoes the passphrase.
	if passphrase != "" {
		helper, cleanup, err := writeAskpassHelper(passphrase)
		if err != nil {
			return err
		}
		defer cleanup()
		cmd.Env = append(cmd.Env,
			"SSH_ASKPASS="+helper,
			"SSH_ASKPASS_REQUIRE=force",
			"DISPLAY=:0", // ssh-add requires DISPLAY to be set for askpass
		)
		// Detach stdin so ssh-add goes through askpass, not terminal.
		cmd.Stdin = nil
	}

	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-add: %s", strings.TrimSpace(errBuf.String()))
	}
	return nil
}

// AgentRemove unloads a specific key from the agent. ssh-add -d
// takes a file path; we look up which key file the fingerprint
// belongs to by running ListKeys.
func (b *OpenSSHBackend) AgentRemove(ctx context.Context, fingerprint string) error {
	if fingerprint == "" {
		return fmt.Errorf("missing fingerprint")
	}
	keys, err := b.ListKeys(ctx)
	if err != nil {
		return err
	}
	for _, k := range keys {
		if k.Fingerprint == fingerprint && k.Path != "" {
			if _, err := runCapture(ctx, "ssh-add", "-d", k.Path); err != nil {
				return fmt.Errorf("ssh-add -d: %w", err)
			}
			return nil
		}
	}
	return fmt.Errorf("fingerprint %s not found in %s", fingerprint, b.sshDir)
}

// AgentRemoveAll clears every identity from the agent with
// `ssh-add -D`.
func (b *OpenSSHBackend) AgentRemoveAll(ctx context.Context) error {
	if _, err := runCapture(ctx, "ssh-add", "-D"); err != nil {
		return fmt.Errorf("ssh-add -D: %w", err)
	}
	return nil
}

// runCapture runs a command with the given context and returns its
// combined stdout/stderr as a string plus any run error. Shared by
// every subprocess in the backend.
func runCapture(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// systemctlUserActive shells out to `systemctl --user is-active` and
// returns true on exit code 0. Any other result (including missing
// systemctl) is false.
func systemctlUserActive(unit string) bool {
	out, err := runCapture(context.Background(), "systemctl", "--user", "is-active", unit)
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "active"
}

// writeAskpassHelper creates a temp shell script that echoes the
// passphrase and returns its path plus a cleanup function. The
// script is marked executable and lives under XDG_RUNTIME_DIR so
// it's torn down on logout even if cleanup is skipped.
func writeAskpassHelper(passphrase string) (string, func(), error) {
	dir := os.Getenv("XDG_RUNTIME_DIR")
	if dir == "" {
		dir = os.TempDir()
	}
	f, err := os.CreateTemp(dir, "dark-askpass-*.sh")
	if err != nil {
		return "", nil, err
	}
	// Write a minimal shell script. The passphrase is single-quoted;
	// escape any single quote in the value by closing + opening the
	// string around it.
	escaped := strings.ReplaceAll(passphrase, "'", `'"'"'`)
	script := "#!/bin/sh\nprintf '%s\\n' '" + escaped + "'\n"
	if _, err := f.WriteString(script); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", nil, err
	}
	f.Close()
	if err := os.Chmod(f.Name(), 0o700); err != nil {
		os.Remove(f.Name())
		return "", nil, err
	}
	cleanup := func() { _ = os.Remove(f.Name()) }
	return f.Name(), cleanup, nil
}

// ensureAgentUnit writes ~/.config/systemd/user/ssh-agent.service on
// first call. Idempotent — returns nil if the file already exists
// with plausible content.
func ensureAgentUnit() error {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configDir = filepath.Join(home, ".config")
	}
	unitDir := filepath.Join(configDir, "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return err
	}
	unitPath := filepath.Join(unitDir, "ssh-agent.service")
	if _, err := os.Stat(unitPath); err == nil {
		return nil
	}
	unit := `[Unit]
Description=SSH key agent (managed by dark)

[Service]
Type=simple
Environment=SSH_AUTH_SOCK=%t/ssh-agent.socket
ExecStart=/usr/bin/ssh-agent -D -a $SSH_AUTH_SOCK
ExecStop=/usr/bin/ssh-add -D
Restart=on-failure

[Install]
WantedBy=default.target
`
	return os.WriteFile(unitPath, []byte(unit), 0o644)
}

// agentUnitExists checks whether dark's managed ssh-agent.service
// file has been written yet. Used to distinguish "not installed" from
// "installed but stopped" in the UI.
func agentUnitExists() bool {
	configDir := os.Getenv("XDG_CONFIG_HOME")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return false
		}
		configDir = filepath.Join(home, ".config")
	}
	path := filepath.Join(configDir, "systemd", "user", "ssh-agent.service")
	_, err := os.Stat(path)
	return err == nil
}
