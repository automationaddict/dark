package appstore

import (
	"log/slog"
	"os/exec"

	"github.com/johnnelson/dark/internal/scripting"
)

// Detect probes the host and returns a Backend wired to whichever
// sources are usable. The probe is cheap — one PATH lookup for pacman
// — and does no network calls, so Detect is safe to call at daemon
// startup without blocking.
//
// The scripting engine is passed through to the pacman backend so it
// can load categories.lua at catalog-build time. A nil engine is safe
// and means categories will be unpopulated.
func Detect(logger *slog.Logger, engine *scripting.Engine) Backend {
	if logger == nil {
		logger = slog.Default()
	}
	if _, err := exec.LookPath("pacman"); err != nil {
		logger.Info("appstore: pacman not found, using noop backend", "err", err)
		return NewNoopBackend(BackendNone)
	}
	pacman := NewPacmanBackend(logger, engine)
	aur := NewAURBackend(logger)
	return NewCompositeBackend(logger, pacman, aur)
}

// NewService constructs the user-facing Service wired to whichever
// backend Detect picks. The scripting engine enables Lua-driven
// category assignment and will host additional hooks in future passes.
func NewService(logger *slog.Logger, engine *scripting.Engine) *Service {
	return NewServiceWithBackend(Detect(logger, engine))
}
