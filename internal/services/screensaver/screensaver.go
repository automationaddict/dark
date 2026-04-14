// Package screensaver wraps the Omarchy screensaver: the tte-driven
// ASCII-art display launched by hypridle and managed through a handful
// of user-owned files. Everything this package touches lives under the
// user's HOME — no pkexec, no /etc edits.
package screensaver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Flag file path: presence means "screensaver disabled". The launch
// script checks for this file on every trigger. Matches the behavior
// of omarchy-toggle-screensaver.
const stateFlagPath = ".local/state/omarchy/toggles/screensaver-off"

// Content file path: the ASCII art the tte effect loop reads from.
const contentRelPath = ".config/omarchy/branding/screensaver.txt"

// Maximum bytes we'll read from the content file. Practical ASCII art
// banners are well under this; the cap prevents a misconfigured file
// from eating megabytes of snapshot payload.
const maxContentBytes = 64 * 1024

// previewTimeout is the failsafe window for LaunchPreview. If the user
// never presses a key, the screensaver stays up forever — we send a
// SIGTERM at this point so the TUI is never permanently stuck.
const previewTimeout = 60 * time.Second

// Snapshot is the state dark displays on the Appearance → Screensaver
// sub-section. Dark publishes it on dark.screensaver.snapshot.
type Snapshot struct {
	// Enabled reflects the kill-switch flag: true when the flag file
	// is absent (launch script will run normally).
	Enabled bool `json:"enabled"`

	// ContentPath is the absolute path of the branding file for display.
	ContentPath string `json:"content_path"`

	// Content is the full ASCII art text. May be empty if the file is
	// missing or unreadable — the view degrades to "(no content)".
	Content string `json:"content,omitempty"`

	// TTEInstalled reports whether `tte` is on PATH. The launch script
	// exits silently if it isn't, which would be a confusing no-op from
	// the user's perspective — surfacing the dependency state lets the
	// UI warn instead.
	TTEInstalled bool `json:"tte_installed"`

	// TerminalName is the binary name of the xdg-terminal-exec target
	// ("alacritty", "ghostty", "kitty", "foot", ...). Empty when
	// xdg-terminal-exec isn't installed or couldn't report.
	TerminalName string `json:"terminal_name,omitempty"`

	// Supported is true when both tte is installed AND the detected
	// terminal is one of the three omarchy-launch-screensaver knows
	// how to configure (alacritty, ghostty, kitty).
	Supported bool `json:"supported"`
}

// ReadSnapshot builds a Snapshot from the filesystem. All reads are
// best-effort — missing files produce zero values rather than errors.
func ReadSnapshot() Snapshot {
	home, _ := os.UserHomeDir()

	flagFull := filepath.Join(home, stateFlagPath)
	_, flagErr := os.Stat(flagFull)
	// Enabled when the flag file does NOT exist.
	enabled := os.IsNotExist(flagErr)

	contentFull := filepath.Join(home, contentRelPath)
	content := readContentFile(contentFull)

	_, tteErr := exec.LookPath("tte")
	tteOK := tteErr == nil

	terminal := detectTerminal()
	supported := tteOK && isSupportedTerminal(terminal)

	return Snapshot{
		Enabled:      enabled,
		ContentPath:  contentFull,
		Content:      content,
		TTEInstalled: tteOK,
		TerminalName: terminal,
		Supported:    supported,
	}
}

// SetEnabled creates or removes the kill-switch flag file.
func SetEnabled(enabled bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	flagFull := filepath.Join(home, stateFlagPath)

	if enabled {
		if err := os.Remove(flagFull); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove flag file: %w", err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(flagFull), 0o755); err != nil {
		return fmt.Errorf("create flag dir: %w", err)
	}
	f, err := os.OpenFile(flagFull, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("create flag file: %w", err)
	}
	return f.Close()
}

// WriteContent overwrites the ASCII art file. The input is clamped to
// maxContentBytes — anything larger is rejected to keep snapshot
// payloads bounded.
func WriteContent(content string) error {
	if len(content) > maxContentBytes {
		return fmt.Errorf("content too large (%d bytes, max %d)", len(content), maxContentBytes)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	path := filepath.Join(home, contentRelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create branding dir: %w", err)
	}
	// Atomic via write-to-tmp + rename so a crash or kill mid-write
	// can't leave the file truncated and blank the screensaver.
	tmp := path + ".dark-tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename to %s: %w", path, err)
	}
	return nil
}

// LaunchPreview runs `omarchy-launch-screensaver force` and blocks
// until the script exits (either because the user pressed a key or
// because the previewTimeout expired and we SIGTERMed our child).
// The script handles cleanup on signals via its own trap.
func LaunchPreview() error {
	if _, err := exec.LookPath("omarchy-launch-screensaver"); err != nil {
		return errors.New("omarchy-launch-screensaver not found on PATH")
	}

	ctx, cancel := context.WithTimeout(context.Background(), previewTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "omarchy-launch-screensaver", "force")
	// The parent launcher forks terminal instances and returns quickly;
	// we want to block until all screensaver windows have exited, so
	// we also poll pgrep as a belt-and-braces wait on the grandchild.
	if err := cmd.Run(); err != nil {
		// Timeout expired → cancel the context → kill the child. That
		// bubbles up as exec.ExitError; translate to a clearer message.
		if ctx.Err() == context.DeadlineExceeded {
			_ = exec.Command("pkill", "-f", "org.omarchy.screensaver").Run()
			return fmt.Errorf("preview timed out after %s — any key or signal would normally exit", previewTimeout)
		}
		// Non-zero exit from the launcher isn't fatal: the script exits
		// non-zero when tte is missing or when the terminal isn't
		// supported. Callers already surface those conditions via
		// Snapshot.TTEInstalled / Supported, so propagate the error
		// text for logging but don't panic.
		return fmt.Errorf("launch screensaver: %w", err)
	}

	// The launcher spawned terminal instances and returned; those are
	// the actual screensaver windows. Wait until they all exit by
	// polling for the window class.
	pollCtx, pollCancel := context.WithTimeout(context.Background(), previewTimeout)
	defer pollCancel()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-pollCtx.Done():
			_ = exec.Command("pkill", "-f", "org.omarchy.screensaver").Run()
			return fmt.Errorf("preview timed out after %s", previewTimeout)
		case <-ticker.C:
			if err := exec.Command("pgrep", "-f", "org.omarchy.screensaver").Run(); err != nil {
				// No matching process: preview has exited cleanly.
				return nil
			}
		}
	}
}

// readContentFile reads up to maxContentBytes of the given file. Any
// error (missing, unreadable, too big) is treated as "no content".
func readContentFile(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	buf := make([]byte, maxContentBytes+1)
	n, _ := f.Read(buf)
	if n > maxContentBytes {
		n = maxContentBytes
	}
	return string(buf[:n])
}

// detectTerminal asks xdg-terminal-exec which terminal would be used,
// then maps the returned .desktop file to a friendly binary name. The
// launch script uses the same mechanism, so dark's view stays in sync
// with what would actually run.
func detectTerminal() string {
	out, err := exec.Command("xdg-terminal-exec", "--print-id").Output()
	if err != nil {
		return ""
	}
	id := strings.TrimSpace(string(out))
	id = strings.ToLower(id)
	switch {
	case strings.Contains(id, "alacritty"):
		return "alacritty"
	case strings.Contains(id, "ghostty"):
		return "ghostty"
	case strings.Contains(id, "kitty"):
		return "kitty"
	case strings.Contains(id, "foot"):
		return "foot"
	case strings.Contains(id, "wezterm"):
		return "wezterm"
	}
	return id
}

func isSupportedTerminal(name string) bool {
	switch name {
	case "alacritty", "ghostty", "kitty":
		return true
	}
	return false
}
