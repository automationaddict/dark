package ssh

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// LoadClientConfig parses ~/.ssh/config into a slice of HostEntry.
// Line numbers are preserved so SaveHostEntry can splice updated
// blocks back in without rewriting unrelated content. A missing
// config file returns an empty ClientConfig, not an error, so fresh
// installs don't look broken.
func (b *OpenSSHBackend) LoadClientConfig(ctx context.Context) (ClientConfig, error) {
	path := filepath.Join(b.sshDir, "config")
	cfg := ClientConfig{Path: path}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var current *HostEntry
	lineNo := 0
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		lineNo++
		raw := scan.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			if current != nil {
				current.LineEnd = lineNo
			}
			continue
		}
		key, value := splitDirective(line)
		if strings.EqualFold(key, "Host") {
			if current != nil {
				cfg.Hosts = append(cfg.Hosts, *current)
			}
			current = &HostEntry{
				Pattern:   value,
				Extras:    map[string]string{},
				LineStart: lineNo,
				LineEnd:   lineNo,
			}
			continue
		}
		if current == nil {
			// Directive before any Host block — ignore. Dark's UI
			// only edits Host blocks; global defaults stay where
			// they are.
			continue
		}
		current.LineEnd = lineNo
		applyDirective(current, key, value)
	}
	if current != nil {
		cfg.Hosts = append(cfg.Hosts, *current)
	}
	return cfg, nil
}

// SaveHostEntry writes a host entry back to ~/.ssh/config. When an
// entry with the same Pattern already exists, the old block is
// replaced in place; otherwise the new block is appended. Writes
// are atomic: temp file + rename, with a .bak next to the original.
func (b *OpenSSHBackend) SaveHostEntry(ctx context.Context, entry HostEntry) error {
	path := filepath.Join(b.sshDir, "config")
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lines := []string{}
	if len(existing) > 0 {
		lines = strings.Split(strings.TrimRight(string(existing), "\n"), "\n")
	}
	block := renderHostBlock(entry)

	// Locate an existing block by Pattern. This reparses the file
	// rather than trusting LineStart/LineEnd from the caller, because
	// the caller's state may be stale after a concurrent edit.
	cfg, _ := b.LoadClientConfig(ctx)
	replaced := false
	for _, h := range cfg.Hosts {
		if h.Pattern != entry.Pattern {
			continue
		}
		// Splice [LineStart-1, LineEnd) out and drop the new block
		// at the same position. Line numbers from the parser are
		// 1-indexed and LineEnd is inclusive.
		before := lines[:h.LineStart-1]
		after := []string{}
		if h.LineEnd < len(lines) {
			after = lines[h.LineEnd:]
		}
		newLines := append([]string{}, before...)
		newLines = append(newLines, strings.Split(block, "\n")...)
		newLines = append(newLines, after...)
		lines = newLines
		replaced = true
		break
	}
	if !replaced {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, strings.Split(block, "\n")...)
	}
	content := strings.Join(lines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return atomicWrite(path, []byte(content), 0o600)
}

// DeleteHostEntry removes the Host block matching pattern from
// ~/.ssh/config. The atomic-write pattern keeps a .bak for recovery.
func (b *OpenSSHBackend) DeleteHostEntry(ctx context.Context, pattern string) error {
	path := filepath.Join(b.sshDir, "config")
	cfg, err := b.LoadClientConfig(ctx)
	if err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimRight(string(existing), "\n"), "\n")
	var target *HostEntry
	for i := range cfg.Hosts {
		if cfg.Hosts[i].Pattern == pattern {
			target = &cfg.Hosts[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("host %q not found in %s", pattern, path)
	}
	before := lines[:target.LineStart-1]
	after := []string{}
	if target.LineEnd < len(lines) {
		after = lines[target.LineEnd:]
	}
	out := append([]string{}, before...)
	out = append(out, after...)
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") && content != "" {
		content += "\n"
	}
	return atomicWrite(path, []byte(content), 0o600)
}

// ─── Server config (read-only in v1) ──────────────────────────────

// LoadServerConfig parses /etc/ssh/sshd_config into a typed view.
// Many fields stay at their zero value when the directive isn't set
// explicitly — OpenSSH falls back to compiled-in defaults that we
// don't try to emulate here. The TUI's phase-2 editor will grow a
// defaults layer; v1 just reports what's literally in the file.
func (b *OpenSSHBackend) LoadServerConfig(ctx context.Context) (ServerConfig, error) {
	path := "/etc/ssh/sshd_config"
	sc := ServerConfig{Path: path}
	data, err := os.ReadFile(path)
	if err != nil {
		sc.ParseError = err.Error()
		return sc, nil
	}
	sc.Readable = true
	sc.RawLines = strings.Split(string(data), "\n")
	for _, raw := range sc.RawLines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value := splitDirective(line)
		switch strings.ToLower(key) {
		case "port":
			sc.Port, _ = strconv.Atoi(value)
		case "permitrootlogin":
			sc.PermitRootLogin = value
		case "passwordauthentication":
			sc.PasswordAuthentication = strings.EqualFold(value, "yes")
		case "pubkeyauthentication":
			sc.PubkeyAuthentication = strings.EqualFold(value, "yes")
		case "x11forwarding":
			sc.X11Forwarding = strings.EqualFold(value, "yes")
		case "allowusers":
			sc.AllowUsers = strings.Fields(value)
		case "allowgroups":
			sc.AllowGroups = strings.Fields(value)
		}
	}
	return sc, nil
}

// ─── helpers ─────────────────────────────────────────────────────

// splitDirective splits an ssh_config / sshd_config line into
// (key, value). Keys are single words; everything after the first
// whitespace run is the value. An optional `=` between key and
// value is tolerated.
func splitDirective(line string) (string, string) {
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return line, ""
	}
	key := line[:i]
	value := strings.TrimLeft(line[i+1:], " \t=")
	return key, strings.TrimSpace(value)
}

// applyDirective fills in the typed HostEntry fields from one key/
// value pair, shunting anything unrecognized into Extras so the
// round-trip is lossless.
func applyDirective(h *HostEntry, key, value string) {
	switch strings.ToLower(key) {
	case "hostname":
		h.HostName = value
	case "user":
		h.User = value
	case "port":
		h.Port, _ = strconv.Atoi(value)
	case "identityfile":
		h.IdentityFile = value
	case "forwardagent":
		h.ForwardAgent = strings.EqualFold(value, "yes")
	case "proxyjump":
		h.ProxyJump = value
	case "stricthostkeychecking":
		h.StrictHostKeyChecking = value
	default:
		h.Extras[key] = value
	}
}

// renderHostBlock turns a HostEntry back into its canonical
// ssh_config representation. Typed fields come first, then Extras
// in alphabetical order for determinism.
func renderHostBlock(h HostEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Host %s\n", h.Pattern)
	if h.HostName != "" {
		fmt.Fprintf(&b, "    HostName %s\n", h.HostName)
	}
	if h.User != "" {
		fmt.Fprintf(&b, "    User %s\n", h.User)
	}
	if h.Port > 0 {
		fmt.Fprintf(&b, "    Port %d\n", h.Port)
	}
	if h.IdentityFile != "" {
		fmt.Fprintf(&b, "    IdentityFile %s\n", h.IdentityFile)
	}
	if h.ForwardAgent {
		b.WriteString("    ForwardAgent yes\n")
	}
	if h.ProxyJump != "" {
		fmt.Fprintf(&b, "    ProxyJump %s\n", h.ProxyJump)
	}
	if h.StrictHostKeyChecking != "" {
		fmt.Fprintf(&b, "    StrictHostKeyChecking %s\n", h.StrictHostKeyChecking)
	}
	// Deterministic order for Extras so diffs stay small.
	extraKeys := make([]string, 0, len(h.Extras))
	for k := range h.Extras {
		extraKeys = append(extraKeys, k)
	}
	sortStrings(extraKeys)
	for _, k := range extraKeys {
		fmt.Fprintf(&b, "    %s %s\n", k, h.Extras[k])
	}
	return strings.TrimRight(b.String(), "\n")
}

// atomicWrite writes data to path via a temp file + rename,
// preserving a .bak of the previous content on first write. Mode
// is applied to the new file explicitly so umask doesn't widen it.
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	if err := os.Chmod(tmpPath, mode); err != nil {
		os.Remove(tmpPath)
		return err
	}
	// Keep a .bak of the previous contents if any.
	if orig, err := os.ReadFile(path); err == nil {
		_ = os.WriteFile(path+".bak", orig, mode)
	}
	return os.Rename(tmpPath, path)
}

// sortStrings is a tiny helper to avoid importing sort into every
// file that needs deterministic output.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
