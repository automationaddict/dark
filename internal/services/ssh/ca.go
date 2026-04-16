// SSH Certificate Authority support. Covers the signing workflow
// (sign a user or host key with a CA), parsing -cert.pub files,
// and the sshd_config directives that trust a CA (TrustedUserCAKeys,
// RevokedKeys). The CA private key itself lives under ~/.ssh like
// any other key — dark's safety invariant (never display or log
// private key contents) still applies.
//
// This file adds types, parsing, and the signing subprocess wrapper.
// The Backend interface gains SignKey and LoadCertificates; sshd
// directive parsing is extended in openssh_config.go. The F4 Keys
// subsection renders certificates alongside regular keys.
package ssh

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Certificate represents a parsed SSH certificate (-cert.pub file).
// OpenSSH embeds certificate data into the public key blob; we get
// the human-readable fields from `ssh-keygen -Lf <cert>`.
type Certificate struct {
	CertPath    string    `json:"cert_path"`     // path to the -cert.pub file
	Type        string    `json:"type"`          // "user" or "host"
	KeyID       string    `json:"key_id"`        // the identity string from signing
	Serial      string    `json:"serial"`        // serial number
	ValidAfter  time.Time `json:"valid_after"`   // not-before
	ValidBefore time.Time `json:"valid_before"`  // expiry (zero = forever)
	Principals  []string  `json:"principals"`    // authorized principals
	CriticalOpt []string  `json:"critical_opts"` // force-command, source-address, etc.
	CAFingerprint string  `json:"ca_fingerprint"` // signing CA's fingerprint
	KeyFingerprint string `json:"key_fingerprint"` // the key this cert is for
	Expired     bool      `json:"expired"`
}

// SignKeyOptions drives `ssh-keygen -s <ca_key> -I <key_id> ...`.
type SignKeyOptions struct {
	CAKeyPath   string   `json:"ca_key_path"`  // path to the CA private key
	KeyPath     string   `json:"key_path"`     // public key to sign (the .pub)
	KeyID       string   `json:"key_id"`       // -I <identity>
	Serial      int64    `json:"serial,omitempty"`      // -z <serial>
	CertType    string   `json:"cert_type"`    // "user" or "host"
	Principals  []string `json:"principals,omitempty"`  // -n <principals>
	ValidityStr string   `json:"validity,omitempty"`    // -V <validity> e.g. "+52w"
	Passphrase  string   `json:"passphrase,omitempty"`  // CA key passphrase
}

// SignKey runs `ssh-keygen -s <ca> -I <id> [-h] [-n ...] [-V ...]
// <key.pub>` and returns the resulting certificate. The -cert.pub
// file is created by ssh-keygen next to the input public key.
func (b *OpenSSHBackend) SignKey(ctx context.Context, opts SignKeyOptions) (Certificate, error) {
	if opts.CAKeyPath == "" {
		return Certificate{}, fmt.Errorf("missing CA key path")
	}
	if opts.KeyPath == "" {
		return Certificate{}, fmt.Errorf("missing key path to sign")
	}
	if opts.KeyID == "" {
		return Certificate{}, fmt.Errorf("missing key ID")
	}

	args := []string{"-s", opts.CAKeyPath, "-I", opts.KeyID}
	if opts.CertType == "host" {
		args = append(args, "-h")
	}
	if len(opts.Principals) > 0 {
		args = append(args, "-n", strings.Join(opts.Principals, ","))
	}
	if opts.ValidityStr != "" {
		args = append(args, "-V", opts.ValidityStr)
	}
	if opts.Serial > 0 {
		args = append(args, "-z", fmt.Sprintf("%d", opts.Serial))
	}
	args = append(args, opts.KeyPath)

	cmd := exec.CommandContext(ctx, "ssh-keygen", args...)
	if opts.Passphrase != "" {
		helper, cleanup, err := writeAskpassHelper(opts.Passphrase)
		if err != nil {
			return Certificate{}, err
		}
		defer cleanup()
		cmd.Env = append(os.Environ(),
			"SSH_ASKPASS="+helper,
			"SSH_ASKPASS_REQUIRE=force",
			"DISPLAY=:0",
		)
		cmd.Stdin = nil
	}
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return Certificate{}, fmt.Errorf("ssh-keygen sign: %s", strings.TrimSpace(errBuf.String()))
	}

	// ssh-keygen creates <keypath>-cert.pub next to the input.
	certPath := strings.TrimSuffix(opts.KeyPath, ".pub") + "-cert.pub"
	certs, err := parseCertFile(ctx, certPath)
	if err != nil || len(certs) == 0 {
		return Certificate{}, fmt.Errorf("signed but failed to parse result: %v", err)
	}
	return certs[0], nil
}

// LoadCertificates finds all *-cert.pub files under ~/.ssh and
// parses each one. Returns only the successfully-parsed entries;
// corrupt cert files are silently skipped.
func (b *OpenSSHBackend) LoadCertificates(ctx context.Context) ([]Certificate, error) {
	entries, err := os.ReadDir(b.sshDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var certs []Certificate
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), "-cert.pub") {
			continue
		}
		path := filepath.Join(b.sshDir, e.Name())
		parsed, err := parseCertFile(ctx, path)
		if err != nil {
			continue
		}
		certs = append(certs, parsed...)
	}
	return certs, nil
}

// parseCertFile runs `ssh-keygen -Lf <path>` and parses the human-
// readable output into a Certificate. The output shape is:
//
//	<path>:
//	        Type: ssh-ed25519-cert-v01@openssh.com user certificate
//	        Public key: ED25519-CERT SHA256:xxxxx
//	        Signing CA: ED25519 SHA256:yyyyy
//	        Key ID: "some-id"
//	        Serial: 0
//	        Valid: from 2026-01-01T00:00:00 to 2027-01-01T00:00:00
//	        Principals:
//	                root
//	                admin
//	        Critical Options: (none)
//	        Extensions:
//	                permit-pty
func parseCertFile(ctx context.Context, path string) ([]Certificate, error) {
	cmd := exec.CommandContext(ctx, "ssh-keygen", "-Lf", path)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	c := Certificate{CertPath: path}
	inPrincipals := false
	inCritical := false

	scan := bufio.NewScanner(&out)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "Principals:") {
			inPrincipals = true
			inCritical = false
			continue
		}
		if strings.HasPrefix(line, "Critical Options:") {
			inCritical = true
			inPrincipals = false
			continue
		}
		if strings.HasPrefix(line, "Extensions:") || strings.HasPrefix(line, "Type:") && inPrincipals {
			inPrincipals = false
			inCritical = false
		}
		if inPrincipals && !strings.Contains(line, ":") {
			c.Principals = append(c.Principals, strings.TrimSpace(line))
			continue
		}
		if inCritical && !strings.Contains(line, ":") && line != "(none)" {
			c.CriticalOpt = append(c.CriticalOpt, strings.TrimSpace(line))
			continue
		}

		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "Type":
			if strings.Contains(val, "user") {
				c.Type = "user"
			} else if strings.Contains(val, "host") {
				c.Type = "host"
			}
		case "Public key":
			if idx := strings.Index(val, "SHA256:"); idx >= 0 {
				c.KeyFingerprint = val[idx:]
			}
		case "Signing CA":
			if idx := strings.Index(val, "SHA256:"); idx >= 0 {
				c.CAFingerprint = val[idx:]
			}
		case "Key ID":
			c.KeyID = strings.Trim(val, "\"")
		case "Serial":
			c.Serial = val
		case "Valid":
			c.ValidAfter, c.ValidBefore = parseValidRange(val)
			if !c.ValidBefore.IsZero() && c.ValidBefore.Before(time.Now()) {
				c.Expired = true
			}
		}
	}
	return []Certificate{c}, nil
}

// parseValidRange parses "from 2026-01-01T00:00:00 to 2027-01-01T00:00:00"
// into two time.Time values. "forever" in either position yields zero.
func parseValidRange(s string) (time.Time, time.Time) {
	s = strings.TrimSpace(s)
	var after, before time.Time
	parts := strings.SplitN(s, " to ", 2)
	if len(parts) >= 1 {
		from := strings.TrimPrefix(parts[0], "from ")
		after, _ = time.Parse("2006-01-02T15:04:05", from)
	}
	if len(parts) >= 2 && parts[1] != "forever" {
		before, _ = time.Parse("2006-01-02T15:04:05", parts[1])
	}
	return after, before
}
