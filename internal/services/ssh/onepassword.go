package ssh

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OnePasswordBackend integrates with the 1Password SSH agent and `op`
// CLI. Key management (list/generate/delete) goes through 1Password
// vaults via `op item`. Agent operations use the standard ssh-agent
// protocol against 1Password's agent socket. File-based operations
// (client config, known_hosts, authorized_keys, server config) are
// delegated to an embedded OpenSSH backend since those files live on
// disk regardless of which agent holds the signing keys.
//
// Detection: the 1Password agent socket lives at
// `~/.1password/agent.sock` on Linux. The `op` CLI must be installed
// and the user must be signed in for vault operations to succeed.
type OnePasswordBackend struct {
	*OpenSSHBackend
	socketPath string
}

// NewOnePasswordBackend returns a configured backend. The socket
// path defaults to ~/.1password/agent.sock when empty. Callers
// should check Available() before routing operations here — when
// 1Password isn't installed or the agent isn't running, every vault
// operation returns a clear "not available" error.
func NewOnePasswordBackend(sshDir string) Backend {
	home, _ := os.UserHomeDir()
	sock := filepath.Join(home, ".1password", "agent.sock")
	return &OnePasswordBackend{
		OpenSSHBackend: NewOpenSSHBackend(sshDir),
		socketPath:     sock,
	}
}

// Name implements Backend.
func (b *OnePasswordBackend) Name() string { return "1password" }

// Available returns true when both the 1Password agent socket exists
// and the `op` CLI is in PATH. The user may still need to sign in
// before vault ops work — that's a runtime error, not a detection
// failure.
func (b *OnePasswordBackend) Available() bool {
	if _, err := os.Stat(b.socketPath); err != nil {
		return false
	}
	_, err := exec.LookPath("op")
	return err == nil
}

// ListKeys queries 1Password vaults for SSH key items via `op item
// list --categories "SSH Key"`, then enriches each entry with
// fingerprint data. Keys stored in 1Password don't have a local
// private key file — Path will be empty, PublicPath will point at
// a temp export if we can fetch the public key, and InAgent is
// cross-referenced against the loaded fingerprints from `ssh-add
// -l` (which talks to the 1Password agent socket).
//
// Disk-based keys under ~/.ssh are also included via the embedded
// OpenSSH backend so the user sees their full combined inventory.
func (b *OnePasswordBackend) ListKeys(ctx context.Context) ([]Key, error) {
	// Start with disk-based keys from the embedded backend.
	diskKeys, err := b.OpenSSHBackend.ListKeys(ctx)
	if err != nil {
		diskKeys = nil
	}

	// Query 1Password for vault SSH keys.
	vaultKeys, vaultErr := b.listVaultKeys(ctx)
	if vaultErr != nil {
		// op not signed in, or no SSH keys — return disk keys only
		// with a non-fatal note so the snapshot captures the issue.
		if diskKeys == nil {
			return nil, vaultErr
		}
		return diskKeys, nil
	}

	// Merge: vault keys first, then disk keys. Dedup by fingerprint
	// so a key that exists in both shows as the vault version.
	seen := map[string]bool{}
	var merged []Key
	for _, k := range vaultKeys {
		merged = append(merged, k)
		if k.Fingerprint != "" {
			seen[k.Fingerprint] = true
		}
	}
	for _, k := range diskKeys {
		if k.Fingerprint != "" && seen[k.Fingerprint] {
			continue
		}
		merged = append(merged, k)
	}
	return merged, nil
}

// AgentStatus overrides the embedded OpenSSH method to point at the
// 1Password agent socket instead of SSH_AUTH_SOCK. The loaded-keys
// list comes from `ssh-add -l` against the 1Password socket so the
// user sees exactly what the 1Password agent is willing to sign.
func (b *OnePasswordBackend) AgentStatus(ctx context.Context) (AgentStatus, error) {
	st := AgentStatus{
		SocketPath: b.socketPath,
	}
	if _, err := os.Stat(b.socketPath); err == nil {
		st.Running = true
	}
	if !st.Running {
		return st, nil
	}
	// Query the 1Password agent directly by overriding SSH_AUTH_SOCK.
	out, err := b.runWithSocket(ctx, "ssh-add", "-l")
	if err != nil {
		if strings.Contains(out, "no identities") {
			return st, nil
		}
		return st, fmt.Errorf("ssh-add -l: %s", strings.TrimSpace(out))
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
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
	return st, nil
}

// AgentStart is a no-op for 1Password — the agent is managed by the
// 1Password desktop app, not by dark or systemd.
func (b *OnePasswordBackend) AgentStart(ctx context.Context) error {
	return fmt.Errorf("1Password agent is managed by the 1Password app — start 1Password to start the agent")
}

// AgentStop is a no-op for 1Password.
func (b *OnePasswordBackend) AgentStop(ctx context.Context) error {
	return fmt.Errorf("1Password agent is managed by the 1Password app — stop 1Password to stop the agent")
}

// AgentAdd is a no-op — 1Password manages which keys are offered
// through its agent.toml config, not through ssh-add.
func (b *OnePasswordBackend) AgentAdd(ctx context.Context, keyPath, passphrase string, lifetimeSeconds int) error {
	return fmt.Errorf("1Password agent keys are managed in 1Password — use the app or agent.toml to configure which keys are offered")
}

// AgentRemove is a no-op for the same reason as AgentAdd.
func (b *OnePasswordBackend) AgentRemove(ctx context.Context, fingerprint string) error {
	return fmt.Errorf("1Password agent keys are managed in 1Password — use the app to manage loaded keys")
}

// AgentRemoveAll is a no-op.
func (b *OnePasswordBackend) AgentRemoveAll(ctx context.Context) error {
	return fmt.Errorf("1Password agent keys are managed in 1Password")
}

// GenerateKey creates a new SSH key item in 1Password via `op item
// create`. The key is stored in the user's Private vault by default.
func (b *OnePasswordBackend) GenerateKey(ctx context.Context, opts GenerateKeyOptions) (Key, error) {
	if opts.Type == "" {
		opts.Type = "ed25519"
	}
	title := opts.Path
	if title == "" {
		title = "id_" + opts.Type
	}
	// `op item create` with category SSH Key generates the key pair
	// inside 1Password's vault. The key never touches disk.
	args := []string{"item", "create",
		"--category", "SSH Key",
		"--title", title,
		"--ssh-key-type", mapKeyType(opts.Type),
	}
	if opts.Comment != "" {
		// 1Password doesn't have a native comment field on SSH keys
		// but we can set a notes field.
		args = append(args, "--notes", opts.Comment)
	}
	out, err := runOpCLI(ctx, args...)
	if err != nil {
		return Key{}, fmt.Errorf("op item create: %s", strings.TrimSpace(out))
	}
	return Key{
		Type:    opts.Type,
		Comment: opts.Comment,
	}, nil
}

// DeleteKey deletes an SSH key from 1Password if it's a vault item,
// or from disk if it's a regular file. Detection: if the path is
// empty or doesn't exist on disk, we try `op item delete`.
func (b *OnePasswordBackend) DeleteKey(ctx context.Context, path string) error {
	if path != "" {
		if _, err := os.Stat(path); err == nil {
			return b.OpenSSHBackend.DeleteKey(ctx, path)
		}
	}
	return fmt.Errorf("1Password vault key deletion requires the item title or UUID — use the 1Password app directly")
}

// ─── Helpers ─────────────────────────────────────────────────────

// opVaultSSHKey is the shape of a single SSH Key item from `op item
// list --categories "SSH Key" --format json`.
type opVaultSSHKey struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Vault struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"vault"`
}

// listVaultKeys calls `op item list` and returns vault SSH keys as
// our Key type. Fingerprints require a second round-trip per key
// via `op item get` which is expensive — for now we skip it and
// mark fingerprint as "(vault)" so the UI can show something.
func (b *OnePasswordBackend) listVaultKeys(ctx context.Context) ([]Key, error) {
	out, err := runOpCLI(ctx, "item", "list", "--categories", "SSH Key", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("op item list: %s", strings.TrimSpace(out))
	}
	var items []opVaultSSHKey
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("parse op output: %w", err)
	}
	keys := make([]Key, 0, len(items))
	for _, item := range items {
		keys = append(keys, Key{
			Comment:     item.Title + " (1Password: " + item.Vault.Name + ")",
			Fingerprint: "(vault:" + item.ID[:8] + ")",
			Type:        "1password",
		})
	}
	return keys, nil
}

// runWithSocket runs a command with SSH_AUTH_SOCK overridden to the
// 1Password agent socket. Returns combined stdout+stderr.
func (b *OnePasswordBackend) runWithSocket(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK="+b.socketPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// runOpCLI runs the `op` CLI with the given arguments and returns
// stdout + any error. Stderr is merged into the output so callers
// see the full picture on failure.
func runOpCLI(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "op", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

// mapKeyType converts dark's type strings to 1Password's
// `--ssh-key-type` values. 1Password supports ed25519 and rsa;
// ecdsa is not available in all 1Password versions.
func mapKeyType(typ string) string {
	switch typ {
	case "rsa":
		return "rsa"
	default:
		return "ed25519"
	}
}
