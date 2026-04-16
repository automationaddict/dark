package ssh

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// OpenSSHBackend is the real, OpenSSH-toolchain-backed implementation
// of Backend. It shells out to ssh-keygen / ssh-add / ssh-agent /
// ssh-keyscan / sshd for every operation, parses the output, and
// round-trips state through the files under ~/.ssh.
//
// The backend holds no state between calls — every operation reads
// the current filesystem and agent state. Callers must hold the
// Service mutex if they want atomicity across multiple calls.
type OpenSSHBackend struct {
	sshDir string
}

// NewOpenSSHBackend binds the backend to a specific ~/.ssh directory.
// Passing an empty string defaults to the current user's home.
func NewOpenSSHBackend(sshDir string) *OpenSSHBackend {
	if sshDir == "" {
		sshDir = defaultSSHDir()
	}
	return &OpenSSHBackend{sshDir: sshDir}
}

// Name implements Backend.
func (b *OpenSSHBackend) Name() string { return "openssh" }

// Available returns true when ssh-keygen is in PATH. The service
// calls this on construction; callers can recheck at any time.
func (b *OpenSSHBackend) Available() bool {
	_, err := exec.LookPath("ssh-keygen")
	return err == nil
}

// ─── Keys ─────────────────────────────────────────────────────────────

// ListKeys walks the ssh dir looking for *.pub files and pairs each
// one with its private counterpart. Keys with a .pub but no private
// file are reported with an empty Path so the UI can surface them
// as orphaned. Fingerprints come from `ssh-keygen -lf <pub>`.
func (b *OpenSSHBackend) ListKeys(ctx context.Context) ([]Key, error) {
	entries, err := os.ReadDir(b.sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read ssh dir: %w", err)
	}
	var keys []Key
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}
		pubPath := filepath.Join(b.sshDir, e.Name())
		privPath := strings.TrimSuffix(pubPath, ".pub")
		info, _ := e.Info()
		pubBytes, readErr := os.ReadFile(pubPath)
		if readErr != nil {
			continue
		}
		k := Key{
			PublicPath: pubPath,
			PublicKey:  strings.TrimSpace(string(pubBytes)),
		}
		if info != nil {
			k.ModTime = info.ModTime()
		}
		if st, err := os.Stat(privPath); err == nil && !st.IsDir() {
			k.Path = privPath
			k.HasPassphrase = detectPassphrase(privPath)
		}
		if fp, typ, bits, comment, err := fingerprint(ctx, pubPath); err == nil {
			k.Fingerprint = fp
			k.Type = typ
			k.Bits = bits
			k.Comment = comment
		}
		keys = append(keys, k)
	}
	return keys, nil
}

// GenerateKey runs ssh-keygen with the requested parameters. The
// target path is resolved relative to the ssh dir when the caller
// passed a bare filename. Overwrites are rejected — the UI must
// confirm and delete first if the user really wants to replace.
func (b *OpenSSHBackend) GenerateKey(ctx context.Context, opts GenerateKeyOptions) (Key, error) {
	if opts.Type == "" {
		opts.Type = "ed25519"
	}
	if opts.Path == "" {
		opts.Path = "id_" + opts.Type
	}
	target := opts.Path
	if !filepath.IsAbs(target) {
		target = filepath.Join(b.sshDir, filepath.Base(target))
	}
	if _, err := os.Stat(target); err == nil {
		return Key{}, fmt.Errorf("%s already exists — delete it first", target)
	}
	args := []string{"-t", opts.Type, "-f", target, "-N", opts.Passphrase}
	if opts.Comment != "" {
		args = append(args, "-C", opts.Comment)
	}
	if opts.Bits > 0 && (opts.Type == "rsa" || opts.Type == "ecdsa") {
		args = append(args, "-b", strconv.Itoa(opts.Bits))
	}
	cmd := exec.CommandContext(ctx, "ssh-keygen", args...)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return Key{}, fmt.Errorf("ssh-keygen: %s", strings.TrimSpace(errBuf.String()))
	}
	// chmod the private key explicitly — ssh-keygen usually does
	// this but enforcing it keeps the invariant tight.
	_ = os.Chmod(target, 0o600)
	_ = os.Chmod(target+".pub", 0o644)

	// Re-stat and return the fresh key.
	keys, err := b.ListKeys(ctx)
	if err != nil {
		return Key{}, err
	}
	for _, k := range keys {
		if k.Path == target {
			return k, nil
		}
	}
	return Key{}, fmt.Errorf("generated key not visible in listing")
}

// DeleteKey removes the private and public files for the given key
// path. Callers are expected to have confirmed with the user first.
func (b *OpenSSHBackend) DeleteKey(ctx context.Context, path string) error {
	if path == "" {
		return fmt.Errorf("missing key path")
	}
	abs := path
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(b.sshDir, filepath.Base(path))
	}
	if !strings.HasPrefix(abs, b.sshDir) {
		return fmt.Errorf("refusing to delete outside %s", b.sshDir)
	}
	// Best-effort: both files may not exist (orphan pub-only key).
	_ = os.Remove(abs)
	_ = os.Remove(abs + ".pub")
	return nil
}

// ChangePassphrase runs `ssh-keygen -p` with the old and new
// passphrases. An empty new passphrase removes the passphrase.
func (b *OpenSSHBackend) ChangePassphrase(ctx context.Context, path, oldPass, newPass string) error {
	if path == "" {
		return fmt.Errorf("missing key path")
	}
	cmd := exec.CommandContext(ctx, "ssh-keygen", "-p", "-P", oldPass, "-N", newPass, "-f", path)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ssh-keygen -p: %s", strings.TrimSpace(errBuf.String()))
	}
	return nil
}

// ─── Fingerprint helper ───────────────────────────────────────────────

// fingerprint parses `ssh-keygen -lf <pub>` which outputs a single
// line in the shape `<bits> <fingerprint> <comment> (<type>)`.
// Missing comment leaves that field empty; unrecognized types leave
// Type empty. All other fields are required — failure returns an
// error so the caller can skip the key.
func fingerprint(ctx context.Context, pubPath string) (fp, typ string, bits int, comment string, err error) {
	cmd := exec.CommandContext(ctx, "ssh-keygen", "-lf", pubPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if runErr := cmd.Run(); runErr != nil {
		return "", "", 0, "", runErr
	}
	line := strings.TrimSpace(out.String())
	// Example: "256 SHA256:xxxxx user@host (ED25519)"
	scan := bufio.NewScanner(strings.NewReader(line))
	scan.Split(bufio.ScanWords)
	var words []string
	for scan.Scan() {
		words = append(words, scan.Text())
	}
	if len(words) < 3 {
		return "", "", 0, "", fmt.Errorf("unexpected fingerprint output: %q", line)
	}
	bits, _ = strconv.Atoi(words[0])
	fp = words[1]
	// Type lives at the end wrapped in parens.
	last := words[len(words)-1]
	if strings.HasPrefix(last, "(") && strings.HasSuffix(last, ")") {
		typ = strings.ToLower(strings.Trim(last, "()"))
		words = words[:len(words)-1]
	}
	if len(words) > 2 {
		comment = strings.Join(words[2:], " ")
	}
	return fp, typ, bits, comment, nil
}

// detectPassphrase tries to load the private key with an empty
// passphrase. A success means the key is unencrypted; a failure
// (any error) is interpreted as "passphrase present". The call is
// silent — ssh-keygen prints to stderr, we swallow it.
func detectPassphrase(path string) bool {
	cmd := exec.Command("ssh-keygen", "-y", "-P", "", "-f", path)
	cmd.Stdout = bytes.NewBuffer(nil)
	cmd.Stderr = bytes.NewBuffer(nil)
	return cmd.Run() != nil
}
