package ssh

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LoadKnownHosts parses ~/.ssh/known_hosts into typed entries. Each
// line is a single entry in OpenSSH's known_hosts format:
//
//	<hostname(s)> <key-type> <base64-key> [comment]
//
// Hashed host entries (starting with |1|) are preserved as-is —
// their hostname is shown as "(hashed)" so the user knows there's
// something there but can't recover the original from the file.
func (b *OpenSSHBackend) LoadKnownHosts(ctx context.Context) ([]KnownHost, error) {
	path := filepath.Join(b.sshDir, "known_hosts")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []KnownHost
	scan := bufio.NewScanner(f)
	lineNo := 0
	for scan.Scan() {
		lineNo++
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		hostname := parts[0]
		if strings.HasPrefix(hostname, "|1|") {
			hostname = "(hashed)"
		}
		entry := KnownHost{
			Hostname:    hostname,
			KeyType:     parts[1],
			Fingerprint: sha256KeyFingerprint(parts[2]),
			LineNumber:  lineNo,
		}
		if len(parts) > 3 {
			entry.Comment = strings.Join(parts[3:], " ")
		}
		out = append(out, entry)
	}
	return out, scan.Err()
}

// ScanHost runs `ssh-keyscan -T 5 <hostname>` and returns the parsed
// entries. The output is a superset of known_hosts format so we can
// reuse the same parser — write the stdout to a temp file and feed
// it back through LoadKnownHosts is overkill; instead we parse
// inline here.
//
// This is auto-called by SaveHostEntry per the plan, but it's also
// exposed as a standalone command so users can scan without adding.
func (b *OpenSSHBackend) ScanHost(ctx context.Context, hostname string) ([]KnownHost, error) {
	if hostname == "" {
		return nil, fmt.Errorf("missing hostname")
	}
	out, err := runCapture(ctx, "ssh-keyscan", "-T", "5", hostname)
	if err != nil {
		return nil, fmt.Errorf("ssh-keyscan: %s", strings.TrimSpace(out))
	}
	var result []KnownHost
	for i, raw := range strings.Split(out, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		result = append(result, KnownHost{
			Hostname:    parts[0],
			KeyType:     parts[1],
			Fingerprint: sha256KeyFingerprint(parts[2]),
			LineNumber:  i + 1,
		})
	}
	return result, nil
}

// RemoveKnownHost wraps `ssh-keygen -R <hostname>`, the canonical
// way to strip an entry. That handles hashed hosts correctly while
// a line-based remove would miss them.
func (b *OpenSSHBackend) RemoveKnownHost(ctx context.Context, hostname string) error {
	if hostname == "" {
		return fmt.Errorf("missing hostname")
	}
	if _, err := runCapture(ctx, "ssh-keygen", "-R", hostname); err != nil {
		return fmt.Errorf("ssh-keygen -R: %w", err)
	}
	return nil
}

// ─── Authorized keys ────────────────────────────────────────────

// LoadAuthorizedKeys parses ~/.ssh/authorized_keys. The format is
// similar to known_hosts but with optional "options" prefixing the
// key (command="...", no-pty, etc). We keep the options string
// intact because re-rendering it would lose information.
func (b *OpenSSHBackend) LoadAuthorizedKeys(ctx context.Context) ([]AuthorizedKey, error) {
	path := filepath.Join(b.sshDir, "authorized_keys")
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var out []AuthorizedKey
	scan := bufio.NewScanner(f)
	lineNo := 0
	for scan.Scan() {
		lineNo++
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		entry := parseAuthorizedKeyLine(line, lineNo)
		if entry.KeyType != "" {
			out = append(out, entry)
		}
	}
	return out, scan.Err()
}

// AddAuthorizedKey appends a raw key line to ~/.ssh/authorized_keys,
// creating the file with 0600 mode if it doesn't exist.
func (b *OpenSSHBackend) AddAuthorizedKey(ctx context.Context, line string) error {
	line = strings.TrimSpace(line)
	if line == "" {
		return fmt.Errorf("missing key")
	}
	path := filepath.Join(b.sshDir, "authorized_keys")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	content := string(existing)
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += line + "\n"
	return atomicWrite(path, []byte(content), 0o600)
}

// RemoveAuthorizedKey strips every line from authorized_keys whose
// fingerprint matches. Hash collisions are not possible in practice
// with SHA256, so matching by fingerprint is safe even though
// multiple keys with the same public-key material would share one.
func (b *OpenSSHBackend) RemoveAuthorizedKey(ctx context.Context, fingerprint string) error {
	if fingerprint == "" {
		return fmt.Errorf("missing fingerprint")
	}
	path := filepath.Join(b.sshDir, "authorized_keys")
	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(existing), "\n"), "\n")
	out := make([]string, 0, len(lines))
	for i, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			out = append(out, raw)
			continue
		}
		entry := parseAuthorizedKeyLine(line, i+1)
		if entry.Fingerprint == fingerprint {
			continue
		}
		out = append(out, raw)
	}
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") && content != "" {
		content += "\n"
	}
	return atomicWrite(path, []byte(content), 0o600)
}

// parseAuthorizedKeyLine handles both the "options <key>" and bare
// "<type> <base64> [comment]" shapes. Options can contain quoted
// strings with spaces so we find the key type by scanning for the
// first field that looks like ssh-*, ecdsa-*, or sk-*.
func parseAuthorizedKeyLine(line string, lineNo int) AuthorizedKey {
	out := AuthorizedKey{LineNumber: lineNo}
	fields := splitAuthorizedFields(line)
	for i, f := range fields {
		if isKeyType(f) {
			if i > 0 {
				out.Options = strings.Join(fields[:i], " ")
			}
			out.KeyType = f
			if i+1 < len(fields) {
				out.Fingerprint = sha256KeyFingerprint(fields[i+1])
			}
			if i+2 < len(fields) {
				out.Comment = strings.Join(fields[i+2:], " ")
			}
			return out
		}
	}
	return out
}

// splitAuthorizedFields splits a line on whitespace but keeps
// quoted segments together so `command="ls -la"` becomes one field.
func splitAuthorizedFields(line string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for _, r := range line {
		switch {
		case r == '"':
			inQuote = !inQuote
			cur.WriteRune(r)
		case (r == ' ' || r == '\t') && !inQuote:
			if cur.Len() > 0 {
				out = append(out, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		out = append(out, cur.String())
	}
	return out
}

// isKeyType returns true for tokens that look like an ssh key type.
func isKeyType(s string) bool {
	return strings.HasPrefix(s, "ssh-") ||
		strings.HasPrefix(s, "ecdsa-") ||
		strings.HasPrefix(s, "sk-")
}

// sha256KeyFingerprint computes the SHA256:<base64> fingerprint of
// a base64-encoded public key blob. Matches what `ssh-keygen -lf`
// produces so the UI can cross-reference known_hosts entries with
// ssh-agent's loaded fingerprints directly.
func sha256KeyFingerprint(b64 string) string {
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	encoded := base64.StdEncoding.EncodeToString(sum[:])
	encoded = strings.TrimRight(encoded, "=")
	return "SHA256:" + encoded
}
