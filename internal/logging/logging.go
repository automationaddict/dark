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
	"log/slog"
	"os"
	"strings"
)

// Setup configures the slog default logger. Call once at the top of
// main() before any slog calls. Safe to call multiple times (each
// call replaces the default).
func Setup(component string) {
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
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(handler).With("component", component))
}
