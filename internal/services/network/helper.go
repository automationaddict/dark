package network

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// helperPath finds the dark-helper binary on disk. Resolution order:
//
//  1. The DARK_HELPER environment variable if set.
//  2. Same directory as the running darkd binary (the dev mode case
//     where everything sits in ./tmp/).
//  3. /usr/local/bin/dark-helper.
//  4. /usr/bin/dark-helper.
//
// We never search $PATH because we want a deterministic, well-known
// location for the privileged binary — and because pkexec validates
// against polkit policy by absolute path, so a fuzzy lookup defeats
// the security model.
func helperPath() (string, error) {
	if env := os.Getenv("DARK_HELPER"); env != "" {
		return env, nil
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "dark-helper")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	for _, p := range []string{"/usr/local/bin/dark-helper", "/usr/bin/dark-helper"} {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("dark-helper binary not found (set DARK_HELPER or install to /usr/local/bin)")
}

// runHelper invokes pkexec dark-helper with the given subcommand and
// arguments, optionally feeding stdin into the helper. pkexec is the
// only sanctioned escalation path — it triggers the user's polkit
// agent which shows the standard authentication dialog, then runs
// the helper as root if the user authenticates.
//
// Failures are translated through interpretHelperError so the TUI
// surfaces a clean one-line message (e.g. "authentication cancelled")
// rather than the raw "exit status 126: Error executing command as
// another user: Request dismissed" pkexec produces.
func runHelper(stdin []byte, args ...string) error {
	helper, err := helperPath()
	if err != nil {
		return err
	}
	full := append([]string{helper}, args...)
	cmd := exec.Command("pkexec", full...)
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr == nil {
		return nil
	}
	return interpretHelperError(runErr, stderr.Bytes())
}

// interpretHelperError converts a pkexec/dark-helper failure into a
// short user-facing string. We look at the exit code first because
// pkexec uses well-defined codes for authorization outcomes (126 for
// "authorization could not be obtained", 127 for "command not found")
// and at stderr second to refine within those buckets. Helper-side
// errors (path validation, write failures) are surfaced verbatim
// after stripping any "dark-helper: " prefix the helper added.
func interpretHelperError(runErr error, stderr []byte) error {
	msg := strings.TrimSpace(string(stderr))

	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		switch exitErr.ExitCode() {
		case 126:
			// pkexec authorization failure. Refine via stderr text.
			switch {
			case strings.Contains(msg, "dismissed"):
				return errors.New("authentication cancelled")
			case strings.Contains(msg, "Authentication failed"):
				return errors.New("authentication failed (wrong password)")
			case strings.Contains(msg, "Not authorized"):
				return errors.New("not authorized to make this change")
			default:
				return errors.New("authentication required but not granted")
			}
		case 127:
			return errors.New("pkexec or polkit is not available on this system")
		}
	}

	if msg != "" {
		// Drop a leading "dark-helper: " label if the helper added
		// one — keeps the TUI prefix from reading "action failed:
		// dark-helper: ...".
		msg = strings.TrimPrefix(msg, "dark-helper: ")
		return errors.New(msg)
	}
	return runErr
}

// writeNetworkdFile asks the privileged helper to atomically write
// the given content to a `.network` file. The path must already be
// validated by the caller against the same rules the helper enforces.
func writeNetworkdFile(path string, content []byte) error {
	return runHelper(content, "write-network-file", path)
}

// deleteNetworkdFile asks the privileged helper to remove a managed
// `.network` file. Already-absent files are treated as success by
// the helper.
func deleteNetworkdFile(path string) error {
	return runHelper(nil, "delete-network-file", path)
}
