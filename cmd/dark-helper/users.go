package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// validUsername matches valid Linux usernames.
var validUsername = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func validateUsername(name string) error {
	if name == "" {
		return fmt.Errorf("empty username")
	}
	if len(name) > 32 {
		return fmt.Errorf("username too long (max 32)")
	}
	if !validUsername.MatchString(name) {
		return fmt.Errorf("invalid username %q", name)
	}
	// Reject system-critical names.
	switch name {
	case "root", "nobody", "daemon", "bin", "sys":
		return fmt.Errorf("cannot modify system user %q", name)
	}
	return nil
}

var validGroupName = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func validateGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("empty group name")
	}
	if !validGroupName.MatchString(name) {
		return fmt.Errorf("invalid group name %q", name)
	}
	return nil
}

func validateShell(shell string) error {
	if shell == "" {
		return fmt.Errorf("empty shell path")
	}
	if !filepath.IsAbs(shell) {
		return fmt.Errorf("shell %q must be absolute path", shell)
	}
	if _, err := os.Stat(shell); err != nil {
		return fmt.Errorf("shell %q does not exist", shell)
	}
	return nil
}

func userAdd(username string, flags []string) error {
	if err := validateUsername(username); err != nil {
		return err
	}

	args := []string{"-m"} // create home directory
	admin := false
	for i := 0; i < len(flags); i++ {
		switch flags[i] {
		case "--comment":
			if i+1 >= len(flags) {
				return fmt.Errorf("--comment requires a value")
			}
			i++
			args = append(args, "-c", flags[i])
		case "--shell":
			if i+1 >= len(flags) {
				return fmt.Errorf("--shell requires a value")
			}
			i++
			if err := validateShell(flags[i]); err != nil {
				return err
			}
			args = append(args, "-s", flags[i])
		case "--admin":
			admin = true
		default:
			return fmt.Errorf("unknown flag %q", flags[i])
		}
	}

	args = append(args, username)
	cmd := exec.Command("useradd", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("useradd: %w", err)
	}

	if admin {
		cmd := exec.Command("gpasswd", "-a", username, "wheel")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("gpasswd (add to wheel): %w", err)
		}
	}

	return nil
}

func userRemove(username string, flags []string) error {
	if err := validateUsername(username); err != nil {
		return err
	}

	args := []string{}
	for _, f := range flags {
		switch f {
		case "--remove-home":
			args = append(args, "-r")
		default:
			return fmt.Errorf("unknown flag %q", f)
		}
	}
	args = append(args, username)

	cmd := exec.Command("userdel", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("userdel: %w", err)
	}
	return nil
}

func userModify(username string, flags ...string) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	// Validate shell if present.
	for i, f := range flags {
		if f == "-s" && i+1 < len(flags) {
			if err := validateShell(flags[i+1]); err != nil {
				return err
			}
		}
	}
	args := append(flags, username)
	cmd := exec.Command("usermod", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("usermod: %w", err)
	}
	return nil
}

func userGroupChange(username, group string, add bool) error {
	if err := validateUsername(username); err != nil {
		return err
	}
	if err := validateGroupName(group); err != nil {
		return err
	}
	flag := "-a"
	if !add {
		flag = "-d"
	}
	cmd := exec.Command("gpasswd", flag, username, group)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gpasswd: %w", err)
	}
	return nil
}

func userSetPassword(username string, verify bool) error {
	if err := validateUsername(username); err != nil {
		return err
	}

	scanner := bufio.NewScanner(os.Stdin)

	if verify {
		// Read and verify the current password first.
		if !scanner.Scan() {
			return fmt.Errorf("no current password provided")
		}
		currentPass := scanner.Text()
		if err := verifyPassword(username, currentPass); err != nil {
			return err
		}
	}

	// Read the new password.
	if !scanner.Scan() {
		return fmt.Errorf("no new password provided")
	}
	newPass := scanner.Text()
	if newPass == "" {
		return fmt.Errorf("password cannot be empty")
	}

	// Set password via chpasswd.
	cmd := exec.Command("chpasswd")
	cmd.Stdin = strings.NewReader(username + ":" + newPass + "\n")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("chpasswd: %w", err)
	}
	return nil
}

// verifyPassword checks the current password by attempting su.
func verifyPassword(username, password string) error {
	cmd := exec.Command("su", "-c", "true", username)
	cmd.Stdin = strings.NewReader(password + "\n")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("current password is incorrect")
	}
	return nil
}
