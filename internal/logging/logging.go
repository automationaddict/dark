// Package logging sets up the process-wide slog default logger. Both
// darkd and dark call Setup() early in main(). The handler writes to
// stderr with a text format that's readable in a terminal and parseable
// by journald when the process is managed by systemd.
//
// Log level is controlled by the DARK_LOG_LEVEL env var:
//
//	debug  — verbose output useful during development
//	info   — normal operations (default)
//	warn   — degraded state, recoverable errors
//	error  — failures the user should investigate
//
// When darkd runs as a systemd user service (`systemctl --user start
// darkd`), stderr is captured by journald automatically and the user
// gets log rotation, persistent storage, and `journalctl --user -u
// darkd` with level filtering — all for free.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func parseLevel() slog.Level {
	level := slog.LevelInfo
	if v := os.Getenv("DARK_LOG_LEVEL"); v != "" {
		switch strings.ToLower(v) {
		case "debug":
			level = slog.LevelDebug
		case "warn", "warning":
			level = slog.LevelWarn
		case "error", "err":
			level = slog.LevelError
		}
	}
	return level
}

// Setup configures the slog default logger. Call once at the top of
// main() before any slog calls. Safe to call multiple times (each
// call replaces the default).
func Setup(component string) {
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: parseLevel(),
	})
	slog.SetDefault(slog.New(handler).With("component", component))
}

// SetupFile configures slog to write to a log file under
// ~/.local/state/dark/<component>.log instead of stderr. Use this
// for the TUI process where stderr output corrupts the alt-screen.
// Returns a closer that the caller should defer. If the file cannot
// be opened, falls back to io.Discard silently.
func SetupFile(component string) io.Closer {
	stateDir := filepath.Join(os.Getenv("HOME"), ".local", "state", "dark")
	_ = os.MkdirAll(stateDir, 0o755)
	f, err := os.OpenFile(
		filepath.Join(stateDir, component+".log"),
		os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644,
	)
	if err != nil {
		f = nil
	}
	var w io.Writer = io.Discard
	if f != nil {
		w = f
	}
	handler := slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: parseLevel(),
	})
	slog.SetDefault(slog.New(handler).With("component", component))
	if f != nil {
		return f
	}
	return io.NopCloser(nil)
}
