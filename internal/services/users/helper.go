package users

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
	return "", fmt.Errorf("dark-helper binary not found")
}

func runHelper(args ...string) error {
	helper, err := helperPath()
	if err != nil {
		return err
	}
	full := append([]string{helper}, args...)
	cmd := exec.Command("pkexec", full...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr == nil {
		return nil
	}
	return interpretHelperError(runErr, stderr.Bytes())
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
				return errors.New("not authorized to make this change")
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

func AddUser(username, fullName, shell string, isAdmin bool) error {
	args := []string{"user-add", username}
	if fullName != "" {
		args = append(args, "--comment", fullName)
	}
	if shell != "" {
		args = append(args, "--shell", shell)
	}
	if isAdmin {
		args = append(args, "--admin")
	}
	return runHelper(args...)
}

func RemoveUser(username string, removeHome bool) error {
	args := []string{"user-remove", username}
	if removeHome {
		args = append(args, "--remove-home")
	}
	return runHelper(args...)
}

func SetShell(username, shell string) error {
	return runHelper("user-shell", username, shell)
}

func SetFullName(username, fullName string) error {
	return runHelper("user-comment", username, fullName)
}

func LockUser(username string) error {
	return runHelper("user-lock", username)
}

func UnlockUser(username string) error {
	return runHelper("user-unlock", username)
}

func AddToGroup(username, group string) error {
	return runHelper("user-group-add", username, group)
}

func RemoveFromGroup(username, group string) error {
	return runHelper("user-group-remove", username, group)
}

// SetPassword changes a user's password. If currentPass is non-empty,
// the helper verifies it before setting the new password (used for the
// current user changing their own password).
func SetPassword(username, currentPass, newPass string) error {
	args := []string{"user-passwd", username}
	if currentPass != "" {
		args = append(args, "--verify")
	}

	helper, err := helperPath()
	if err != nil {
		return err
	}
	full := append([]string{helper}, args...)
	cmd := exec.Command("pkexec", full...)

	var stdinData string
	if currentPass != "" {
		stdinData = currentPass + "\n" + newPass + "\n"
	} else {
		stdinData = newPass + "\n"
	}
	cmd.Stdin = strings.NewReader(stdinData)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if runErr == nil {
		return nil
	}
	return interpretHelperError(runErr, stderr.Bytes())
}

// ElevatedSnapshot reads shadow data via pkexec and returns a full snapshot.
// This triggers a polkit auth prompt.
func ElevatedSnapshot() (Snapshot, error) {
	helper, err := helperPath()
	if err != nil {
		return Snapshot{}, err
	}
	full := []string{helper, "read-shadow"}
	cmd := exec.Command("pkexec", full...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if runErr := cmd.Run(); runErr != nil {
		return Snapshot{}, interpretHelperError(runErr, stderr.Bytes())
	}

	// Parse shadow data from the elevated read.
	shadowData := stdout.Bytes()
	shadow := parseShadowData(shadowData)

	// Build a normal snapshot and enrich with shadow.
	s := ReadSnapshot()
	for i := range s.Users {
		enrichShadow(&s.Users[i], shadow)
	}
	return s, nil
}
