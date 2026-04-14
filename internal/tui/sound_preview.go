package tui

import (
	"os/exec"
	"sync"
)

// The sound preview is a TUI-local side effect: when the user scrolls
// through the notification sound picker we spawn mpv to play the
// focused sample, killing the previous preview so only one sound is
// ever audible at a time. This lives in its own file — not in the
// notify_actions dispatch table — because the preview has no state
// that dark-core or the daemon need to see; it's purely a UX aid that
// belongs next to the view layer rather than the command pipeline.

var (
	previewMu  sync.Mutex
	previewCmd *exec.Cmd
)

// previewSound starts (or replaces) the current preview. Errors from
// mpv start are intentionally swallowed — a missing mpv or bad file
// should not break the dialog, and the user gets immediate audible
// feedback either way.
func previewSound(name string) {
	previewMu.Lock()
	defer previewMu.Unlock()
	killPreviewLocked()
	path := "/usr/share/sounds/freedesktop/stereo/" + name + ".oga"
	cmd := exec.Command("mpv", "--no-terminal", "--no-video", path)
	if err := cmd.Start(); err != nil {
		return
	}
	previewCmd = cmd
}

// stopPreview tears down any running preview. Safe to call even when
// no preview is active.
func stopPreview() {
	previewMu.Lock()
	defer previewMu.Unlock()
	killPreviewLocked()
}

// killPreviewLocked assumes previewMu is held by the caller. Reads
// previewCmd under the lock and, if a process is present, kills it and
// clears the handle. The nil-guard on Process prevents a panic if mpv
// exits between Start and our Kill.
func killPreviewLocked() {
	if previewCmd == nil {
		return
	}
	if previewCmd.Process != nil {
		_ = previewCmd.Process.Kill()
	}
	previewCmd = nil
}
