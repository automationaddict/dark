package appearance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// SetBackground points ~/.config/omarchy/current/background at the
// given image and relaunches swaybg so the new wallpaper takes
// effect immediately. name is the bare filename from the Snapshot
// Backgrounds list; we resolve it against the per-theme user
// backgrounds dir first, then fall back to the theme-shipped
// backgrounds dir — matching what omarchy-theme-bg-next does.
//
// The Omarchy shell wrapper omarchy-theme-bg-set does roughly the
// same dance; dark reimplements it in Go per the project's pure-
// Go policy so the backend stays directly testable and doesn't
// depend on that wrapper script being on PATH.
func SetBackground(name string) error {
	if name == "" {
		return fmt.Errorf("no background name")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}

	src, err := resolveBackgroundPath(home, name)
	if err != nil {
		return err
	}

	link := filepath.Join(home, ".config/omarchy/current/background")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return fmt.Errorf("create omarchy state dir: %w", err)
	}
	// ln -nsf semantics: remove any existing file or symlink, then
	// create a fresh symlink. Go's os.Symlink refuses to overwrite
	// an existing target so we do the remove-then-symlink dance.
	_ = os.Remove(link)
	if err := os.Symlink(src, link); err != nil {
		return fmt.Errorf("symlink %s -> %s: %w", link, src, err)
	}

	return restartSwaybg(link)
}

// resolveBackgroundPath returns the absolute filesystem path of the
// given background filename, checking the per-theme user directory
// first and falling back to the theme-shipped one. Returns an
// error when neither location has a readable file.
func resolveBackgroundPath(home, name string) (string, error) {
	themeName := readThemeName(home)
	candidates := []string{
		filepath.Join(home, ".config/omarchy/backgrounds", themeName, name),
		filepath.Join(home, ".config/omarchy/current/theme/backgrounds", name),
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("background %q not found in user or theme dir", name)
}

// restartSwaybg pkills any running swaybg and respawns it pointed
// at the current background symlink. The launch path prefers
// uwsm-app (Omarchy's session-aware launcher) and falls back to a
// bare swaybg invocation for vanilla Hyprland — same pattern as
// the hypridle and waybar helpers elsewhere in the service layer.
func restartSwaybg(backgroundLink string) error {
	if _, err := exec.LookPath("swaybg"); err != nil {
		return fmt.Errorf("swaybg not installed")
	}
	// Best-effort kill of any running instance. pkill exits
	// non-zero when nothing matches, which isn't fatal.
	_ = exec.Command("pkill", "-x", "swaybg").Run()

	var argv []string
	if _, err := exec.LookPath("uwsm-app"); err == nil {
		argv = []string{"uwsm-app", "--", "swaybg", "-i", backgroundLink, "-m", "fill"}
	} else {
		argv = []string{"swaybg", "-i", backgroundLink, "-m", "fill"}
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start swaybg: %w", err)
	}
	_ = cmd.Process.Release()
	return nil
}
