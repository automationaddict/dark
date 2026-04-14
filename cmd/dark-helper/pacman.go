package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// validPkgName matches the characters pacman allows in package names.
// Anything outside this set is rejected before we hand it to pacman
// so shell metacharacters and path traversal are impossible.
var validPkgName = regexp.MustCompile(`^[a-zA-Z0-9@._+-]+$`)

// validatePackageNames checks that every name in the list is a legal
// pacman package name and that the batch size is within our cap. The
// cap exists to prevent abuse — a misbehaving caller could otherwise
// ask us to install the entire repo.
func validatePackageNames(names []string) error {
	if len(names) == 0 {
		return fmt.Errorf("no package names provided")
	}
	if len(names) > maxPacmanPackages {
		return fmt.Errorf("too many packages (%d, max %d)", len(names), maxPacmanPackages)
	}
	for _, name := range names {
		if name == "" {
			return fmt.Errorf("empty package name")
		}
		if !validPkgName.MatchString(name) {
			return fmt.Errorf("invalid package name %q", name)
		}
	}
	return nil
}

// runPacman executes pacman with the given arguments and streams its
// stdout/stderr to our own stdout/stderr so the daemon can capture
// progress output for the TUI status line.
func runPacman(args ...string) error {
	cmd := exec.Command("pacman", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pacman %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func pacmanInstall(names []string) error {
	if err := validatePackageNames(names); err != nil {
		return err
	}
	args := append([]string{"-S", "--noconfirm"}, names...)
	return runPacman(args...)
}

func pacmanRemove(names []string) error {
	if err := validatePackageNames(names); err != nil {
		return err
	}
	args := append([]string{"-R", "--noconfirm"}, names...)
	return runPacman(args...)
}

func pacmanUpgrade() error {
	return runPacman("-Syu", "--noconfirm")
}
