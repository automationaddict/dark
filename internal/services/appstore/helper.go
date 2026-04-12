package appstore

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// helperPath finds the dark-helper binary. Resolution order mirrors
// network/helper.go exactly so both services find the same binary.
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
// arguments. Mirrors network/helper.go's runHelper.
func runHelper(args ...string) (string, error) {
	helper, err := helperPath()
	if err != nil {
		return "", err
	}
	full := append([]string{helper}, args...)
	cmd := exec.Command("pkexec", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr == nil {
		return stdout.String(), nil
	}
	return stdout.String(), interpretHelperError(runErr, stderr.Bytes())
}

func interpretHelperError(runErr error, stderr []byte) error {
	msg := strings.TrimSpace(string(stderr))
	var exitErr *exec.ExitError
	if errors.As(runErr, &exitErr) {
		switch exitErr.ExitCode() {
		case 126:
			switch {
			case strings.Contains(msg, "dismissed"):
				return errors.New("authentication cancelled")
			case strings.Contains(msg, "Authentication failed"):
				return errors.New("authentication failed (wrong password)")
			case strings.Contains(msg, "Not authorized"):
				return errors.New("not authorized to perform this action")
			default:
				return errors.New("authentication required but not granted")
			}
		case 127:
			return errors.New("pkexec or polkit is not available on this system")
		}
	}
	if msg != "" {
		msg = strings.TrimPrefix(msg, "dark-helper: ")
		return errors.New(msg)
	}
	return runErr
}

// helperInstall asks dark-helper to install packages via pacman -S.
func helperInstall(names []string) (string, error) {
	args := append([]string{"pacman-install"}, names...)
	return runHelper(args...)
}

// helperRemove asks dark-helper to remove packages via pacman -R.
func helperRemove(names []string) (string, error) {
	args := append([]string{"pacman-remove"}, names...)
	return runHelper(args...)
}

// helperUpgrade asks dark-helper to run pacman -Syu.
func helperUpgrade() (string, error) {
	return runHelper("pacman-upgrade")
}

// detectAURHelper checks for paru or yay on PATH and returns the
// binary name. Returns "" when neither is found — the TUI uses this
// to show "install an AUR helper" instead of an install button.
func detectAURHelper() string {
	for _, name := range []string{"paru", "yay"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

// aurInstall runs the detected AUR helper to install a package. The
// helper runs as the current user (not via pkexec) because makepkg
// refuses to run as root. The helper handles sudo internally for the
// final pacman -U step.
func aurInstall(helper, name string) (string, error) {
	if helper == "" {
		return "", fmt.Errorf("no AUR helper installed (install paru or yay)")
	}
	cmd := exec.Command(helper, "-S", "--noconfirm", name)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return stdout.String(), fmt.Errorf("%s: %s", helper, msg)
		}
		return stdout.String(), fmt.Errorf("%s: %w", helper, err)
	}
	return stdout.String(), nil
}
