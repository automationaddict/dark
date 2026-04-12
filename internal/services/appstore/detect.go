package appstore

import (
	"log/slog"
	"os/exec"
)

// Detect probes the host and returns a Backend wired to whichever
// sources are usable. The probe is cheap — one PATH lookup for pacman
// — and does no network calls, so Detect is safe to call at daemon
// startup without blocking.
//
// The selection rules are:
//
//  1. Pacman present → compose a pacmanBackend with an aurBackend. The
//     AUR side handles its own availability via rate-limit state and
//     network errors at request time, so we don't probe it here.
//  2. Pacman absent → return a noopBackend tagged "none". The TUI will
//     render an explanation in the app store panel.
//
// The function takes a logger so backends can emit structured events
// from the moment they're constructed.
func Detect(logger *slog.Logger) Backend {
	if logger == nil {
		logger = slog.Default()
	}
	if _, err := exec.LookPath("pacman"); err != nil {
		logger.Info("appstore: pacman not found, using noop backend", "err", err)
		return NewNoopBackend(BackendNone)
	}
	pacman := NewPacmanBackend(logger)
	aur := NewAURBackend(logger)
	return NewCompositeBackend(logger, pacman, aur)
}

// NewService constructs the user-facing Service wired to whichever
// backend Detect picks. Callers use this in place of
// NewServiceWithBackend when they don't need to inject a specific
// backend for tests.
func NewService(logger *slog.Logger) *Service {
	return NewServiceWithBackend(Detect(logger))
}
