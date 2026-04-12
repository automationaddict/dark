package main

import (
	"log/slog"
	"sync"
	"time"

	"github.com/johnnelson/dark/internal/services/notify"
)

// daemonNotifier wraps the notify.Notifier with debouncing so
// repeated publish failures during a bad bus tick don't spam the
// user's desktop with 20 identical notifications.
type daemonNotifier struct {
	n       *notify.Notifier
	mu      sync.Mutex
	lastMsg string
	lastAt  time.Time
}

const daemonNotifyDebounce = 30 * time.Second

// newDaemonNotifier creates a best-effort notifier for the daemon
// process. If the session bus is unreachable, returns a stub that
// logs instead of notifying — the daemon still runs fine, just
// without desktop notifications.
func newDaemonNotifier() *daemonNotifier {
	n, err := notify.New("darkd")
	if err != nil {
		slog.Warn("notifications disabled", "error", err)
		return &daemonNotifier{}
	}
	return &daemonNotifier{n: n}
}

func (d *daemonNotifier) Close() {
	if d != nil && d.n != nil {
		d.n.Close()
	}
}

// Warn fires a normal-urgency notification for a degraded but
// non-fatal condition (e.g., a service backend couldn't connect).
// Always fires — no debounce — because startup warnings happen once.
func (d *daemonNotifier) Warn(section, message string) {
	slog.Warn(message, "service", section)
	if d == nil || d.n == nil {
		return
	}
	d.n.Send(notify.Message{
		Summary: "darkd · " + section,
		Body:    message,
		Urgency: notify.UrgencyNormal,
		Icon:    "dialog-warning",
	})
}

// Error fires a critical-urgency notification for a runtime failure.
// Debounced: the same message within 30 seconds is suppressed so a
// repeated publish failure doesn't flood the notification daemon.
func (d *daemonNotifier) Error(section, message string) {
	slog.Error(message, "service", section)
	if d == nil || d.n == nil {
		return
	}
	d.mu.Lock()
	if d.lastMsg == message && time.Since(d.lastAt) < daemonNotifyDebounce {
		d.mu.Unlock()
		return
	}
	d.lastMsg = message
	d.lastAt = time.Now()
	d.mu.Unlock()

	d.n.Send(notify.Message{
		Summary: "darkd · " + section,
		Body:    message,
		Urgency: notify.UrgencyCritical,
		Icon:    "dialog-error",
	})
}
