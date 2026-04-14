package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// setChannel updates /etc/pacman.d/mirrorlist and /etc/pacman.conf to
// point at the given release channel (stable, rc, or edge).
func setChannel(channel string) error {
	mirrors := map[string]string{
		"stable": "https://stable-mirror.omarchy.org/$repo/os/$arch",
		"rc":     "https://rc-mirror.omarchy.org/$repo/os/$arch",
		"edge":   "https://mirror.omarchy.org/$repo/os/$arch",
		"dev":    "https://mirror.omarchy.org/$repo/os/$arch",
	}
	pkgs := map[string]string{
		"stable": "https://pkgs.omarchy.org/stable/$arch",
		"rc":     "https://pkgs.omarchy.org/rc/$arch",
		"edge":   "https://pkgs.omarchy.org/edge/$arch",
		"dev":    "https://pkgs.omarchy.org/edge/$arch",
	}

	mirrorURL, ok := mirrors[channel]
	if !ok {
		return fmt.Errorf("unknown channel %q (must be stable, rc, or edge)", channel)
	}
	pkgURL := pkgs[channel]

	// Write mirrorlist
	mirrorContent := "Server = " + mirrorURL + "\n"
	if err := os.WriteFile("/etc/pacman.d/mirrorlist", []byte(mirrorContent), 0o644); err != nil {
		return fmt.Errorf("write mirrorlist: %w", err)
	}

	// Update pacman.conf — replace the Server line under [omarchy]
	data, err := os.ReadFile("/etc/pacman.conf")
	if err != nil {
		return fmt.Errorf("read pacman.conf: %w", err)
	}
	lines := strings.Split(string(data), "\n")
	inOmarchy := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[omarchy]" {
			inOmarchy = true
			continue
		}
		if inOmarchy && strings.HasPrefix(trimmed, "[") {
			break
		}
		if inOmarchy && strings.HasPrefix(trimmed, "Server") {
			lines[i] = "Server = " + pkgURL
			break
		}
	}
	if err := os.WriteFile("/etc/pacman.conf", []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return fmt.Errorf("write pacman.conf: %w", err)
	}

	return nil
}

// updateFull runs all privileged update steps in a single invocation
// so only one pkexec prompt is needed.
func updateFull() error {
	fmt.Fprintln(os.Stdout, "Syncing time...")
	_ = runCmd("systemctl", "restart", "systemd-timesyncd")

	fmt.Fprintln(os.Stdout, "Updating keyring...")
	if err := updateKeyring(); err != nil {
		return fmt.Errorf("keyring: %w", err)
	}

	fmt.Fprintln(os.Stdout, "Updating system packages...")
	if err := runPacman("-Syyu", "--noconfirm"); err != nil {
		return fmt.Errorf("pacman: %w", err)
	}

	fmt.Fprintln(os.Stdout, "Removing orphan packages...")
	_ = removeOrphans() // best-effort

	return nil
}

// updateKeyring ensures the omarchy-keyring and archlinux-keyring are
// up to date. Mirrors omarchy-update-keyring.
func updateKeyring() error {
	const keyID = "40DFB630FF42BCFFB047046CF0134EE680CAC571"
	// Check if key is already present
	if err := exec.Command("pacman-key", "--list-keys", keyID).Run(); err != nil {
		// Import and sign the key
		if err := runCmd("pacman-key", "--recv-keys", keyID, "--keyserver", "keys.openpgp.org"); err != nil {
			return fmt.Errorf("recv-keys: %w", err)
		}
		if err := runCmd("pacman-key", "--lsign-key", keyID); err != nil {
			return fmt.Errorf("lsign-key: %w", err)
		}
		// Partial sync + install omarchy-keyring
		if err := runPacman("-Sy"); err != nil {
			return fmt.Errorf("pacman -Sy: %w", err)
		}
		if err := runPacman("-S", "--noconfirm", "--needed", "omarchy-keyring"); err != nil {
			return fmt.Errorf("install omarchy-keyring: %w", err)
		}
	}
	// Update archlinux-keyring
	return runPacman("-Sy", "--noconfirm", "archlinux-keyring")
}

// removeOrphans removes packages that were installed as dependencies
// but are no longer required by any installed package.
func removeOrphans() error {
	out, err := exec.Command("pacman", "-Qtdq").Output()
	if err != nil {
		// Exit code 1 means no orphans — that's fine
		return nil
	}
	pkgs := strings.Fields(strings.TrimSpace(string(out)))
	if len(pkgs) == 0 {
		return nil
	}
	for _, pkg := range pkgs {
		// Best-effort removal; skip failures (dependency chains)
		exec.Command("pacman", "-Rs", "--noconfirm", pkg).Run()
	}
	return nil
}
