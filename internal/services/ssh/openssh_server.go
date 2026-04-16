package ssh

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ServerConfigEdit is the typed payload for a sshd_config mutation.
// Callers (UI dialogs, bus handlers, Lua scripts) send one of these
// to apply any subset of the editable directives. Nil pointer means
// "leave this directive untouched"; a non-nil pointer to zero means
// "remove this directive entirely". The splice algorithm in
// applyServerConfigEdit honors both semantics.
type ServerConfigEdit struct {
	Port                   *int      `json:"port,omitempty"`
	PermitRootLogin        *string   `json:"permit_root_login,omitempty"`
	PasswordAuthentication *bool     `json:"password_authentication,omitempty"`
	PubkeyAuthentication   *bool     `json:"pubkey_authentication,omitempty"`
	X11Forwarding          *bool     `json:"x11_forwarding,omitempty"`
	AllowUsers             *[]string `json:"allow_users,omitempty"`
	AllowGroups            *[]string `json:"allow_groups,omitempty"`
}

// SaveServerConfig applies a ServerConfigEdit to /etc/ssh/sshd_config
// via the dark-helper privilege-escalation path. The sequence is:
//
//  1. Read the current file line-by-line (already parsed into
//     ServerConfig.RawLines by LoadServerConfig).
//  2. Splice each set directive into an existing line of the same
//     name, or append a new line at the end when the directive
//     isn't present yet.
//  3. Pipe the reconstructed content through
//     `pkexec dark-helper sshd-config-write` which runs `sshd -t`
//     for validation, keeps a .bak, and atomically installs the
//     new file. dark-helper's exit code + stderr propagate back.
//
// Validation failures surface the sshd -t output verbatim so the
// user can see which directive the parser rejected.
func (b *OpenSSHBackend) SaveServerConfig(ctx context.Context, edit ServerConfigEdit) error {
	sc, err := b.LoadServerConfig(ctx)
	if err != nil {
		return err
	}
	if !sc.Readable {
		return fmt.Errorf("cannot edit %s: %s", sc.Path, sc.ParseError)
	}
	newLines := applyServerConfigEdit(sc.RawLines, edit)
	content := strings.Join(newLines, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return runSSHDHelperWrite(ctx, content)
}

// applyServerConfigEdit is the pure part of the splice-edit path.
// Given the current file's lines and a set of field updates,
// returns a new slice of lines with each directive either replaced
// in place or appended. Unrelated lines (comments, blanks, other
// directives, Match blocks) pass through untouched so round-trips
// preserve the file's structure.
func applyServerConfigEdit(lines []string, e ServerConfigEdit) []string {
	out := append([]string(nil), lines...)
	applyScalar := func(directive, value string, present bool) {
		out = setDirective(out, directive, value, present)
	}
	if e.Port != nil {
		applyScalar("Port", strconv.Itoa(*e.Port), *e.Port > 0)
	}
	if e.PermitRootLogin != nil {
		applyScalar("PermitRootLogin", *e.PermitRootLogin, *e.PermitRootLogin != "")
	}
	if e.PasswordAuthentication != nil {
		applyScalar("PasswordAuthentication", boolWord(*e.PasswordAuthentication), true)
	}
	if e.PubkeyAuthentication != nil {
		applyScalar("PubkeyAuthentication", boolWord(*e.PubkeyAuthentication), true)
	}
	if e.X11Forwarding != nil {
		applyScalar("X11Forwarding", boolWord(*e.X11Forwarding), true)
	}
	if e.AllowUsers != nil {
		applyScalar("AllowUsers", strings.Join(*e.AllowUsers, " "), len(*e.AllowUsers) > 0)
	}
	if e.AllowGroups != nil {
		applyScalar("AllowGroups", strings.Join(*e.AllowGroups, " "), len(*e.AllowGroups) > 0)
	}
	return out
}

// setDirective locates an existing `<directive> <value>` line in
// lines (active OR commented out) and replaces it, or appends a new
// one when none exists. When present=false and a line already
// exists, the line is deleted so the user can "unset" a directive.
//
// Match blocks are treated as a hard stop: directives inside a
// `Match` section stay in that section, so we only look at the
// global portion at the top of the file. sshd's defaults apply
// outside Match blocks which is the behavior the user expects.
func setDirective(lines []string, directive, value string, present bool) []string {
	inMatch := false
	foundIdx := -1
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		bare := strings.TrimPrefix(trimmed, "#")
		bare = strings.TrimSpace(bare)
		if bare == "" {
			continue
		}
		first, _, _ := strings.Cut(bare, " ")
		if strings.EqualFold(first, "Match") && !strings.HasPrefix(trimmed, "#") {
			inMatch = true
			continue
		}
		if inMatch {
			continue
		}
		if strings.EqualFold(first, directive) {
			foundIdx = i
			break
		}
	}
	newLine := directive + " " + value
	switch {
	case foundIdx >= 0 && present:
		lines[foundIdx] = newLine
	case foundIdx >= 0 && !present:
		// Delete the existing line by splicing it out.
		lines = append(lines[:foundIdx], lines[foundIdx+1:]...)
	case foundIdx < 0 && present:
		// Append at the global-section end. If the file has Match
		// blocks, insert just before the first one so the new
		// directive stays global.
		insertAt := len(lines)
		for i, raw := range lines {
			trimmed := strings.TrimSpace(raw)
			if strings.HasPrefix(trimmed, "#") {
				continue
			}
			first, _, _ := strings.Cut(trimmed, " ")
			if strings.EqualFold(first, "Match") {
				insertAt = i
				break
			}
		}
		lines = append(lines[:insertAt],
			append([]string{newLine}, lines[insertAt:]...)...)
	}
	return lines
}

// runSSHDHelperWrite shells out to `pkexec dark-helper
// sshd-config-write` and streams the new config into the helper's
// stdin. pkexec shows the polkit auth dialog on the user's session;
// in headless environments (no polkit agent) the call fails with a
// clear error. Stderr from the helper flows back so the caller can
// show sshd -t's exact complaint.
func runSSHDHelperWrite(ctx context.Context, content string) error {
	helper, err := exec.LookPath("dark-helper")
	if err != nil {
		return fmt.Errorf("dark-helper not in PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, "pkexec", helper, "sshd-config-write")
	cmd.Stdin = strings.NewReader(content)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("pkexec dark-helper sshd-config-write: %s", msg)
	}
	return nil
}

// runSSHDHelperRestore invokes `pkexec dark-helper sshd-config-restore`
// which validates /etc/ssh/sshd_config.bak with `sshd -t` before
// moving it into place. Same polkit prompt + stderr passthrough as
// the write path.
func runSSHDHelperRestore(ctx context.Context) error {
	helper, err := exec.LookPath("dark-helper")
	if err != nil {
		return fmt.Errorf("dark-helper not in PATH: %w", err)
	}
	cmd := exec.CommandContext(ctx, "pkexec", helper, "sshd-config-restore")
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(errBuf.String())
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("pkexec dark-helper sshd-config-restore: %s", msg)
	}
	return nil
}

// boolWord maps Go booleans to the yes/no tokens sshd_config expects.
func boolWord(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
