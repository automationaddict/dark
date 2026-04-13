package privacy

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

func SetDNSOverTLS(value string) error {
	return runHelper("resolved-set", "DNSOverTLS", value)
}

func SetDNSSEC(value string) error {
	return runHelper("resolved-set", "DNSSEC", value)
}

func SetFirewall(enable bool) error {
	action := "ufw-enable"
	if !enable {
		action = "ufw-disable"
	}
	return runHelper(action)
}

func SetSSH(enable bool) error {
	action := "sshd-enable"
	if !enable {
		action = "sshd-disable"
	}
	return runHelper(action)
}

func SetLocation(enable bool) error {
	action := "geoclue-enable"
	if !enable {
		action = "geoclue-disable"
	}
	return runHelper(action)
}

func SetMACRandomizationElevated(value string) error {
	return runHelper("iwd-mac-random", value)
}

func SetIndexer(enable bool) error {
	action := "indexer-enable"
	if !enable {
		action = "indexer-disable"
	}
	return runHelper(action)
}

func SetCoredumpStorage(value string) error {
	return runHelper("resolved-set-coredump", value)
}
