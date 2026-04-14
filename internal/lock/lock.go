// Package lock provides flock-based singleton enforcement for the dark
// binaries. The lock file lives under $XDG_RUNTIME_DIR/dark/<name>.lock and
// contains the holder's PID for diagnostics.
//
// flock is per–open-file-description, so the lock is automatically released
// when the holding process exits (clean exit, crash, or kill). That makes it
// crash-safe with no stale-pid cleanup to write.
package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Lock represents an acquired singleton lock. Callers must Release on exit.
type Lock struct {
	file *os.File
	path string
}

// LogWarn is the sink for non-fatal lock lifecycle errors (PID-write
// failures, cleanup problems on Release). The lock package intentionally
// does not import slog; main.go wires its logger here on startup so the
// package stays dependency-free. Default is a no-op.
var LogWarn = func(op string, err error) {}

// Acquire takes the named lock or returns an error describing who holds it.
// Callers should treat any non-nil error from Acquire as "another instance
// is running" and exit, since the only failure modes are filesystem errors
// (which mean we can't enforce singleton anyway) or contention.
func Acquire(name string) (*Lock, error) {
	dir, err := lockDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create lock dir: %w", err)
	}

	path := filepath.Join(dir, name+".lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if err == syscall.EWOULDBLOCK {
			holder := readPID(path)
			if holder > 0 {
				return nil, fmt.Errorf("%s already running (pid %d)", name, holder)
			}
			return nil, fmt.Errorf("%s already running", name)
		}
		return nil, fmt.Errorf("flock: %w", err)
	}

	// The lock is held either way; PID-write failures only impact
	// HolderPID() diagnostics, so we keep the lock and warn.
	if err := f.Truncate(0); err != nil {
		LogWarn("truncate-lockfile", err)
	} else {
		if _, err := f.WriteString(strconv.Itoa(os.Getpid()) + "\n"); err != nil {
			LogWarn("write-pid", err)
		} else if err := f.Sync(); err != nil {
			LogWarn("sync-lockfile", err)
		}
	}

	return &Lock{file: f, path: path}, nil
}

// Release unlocks the file and removes it. Safe to call multiple times.
// The kernel drops the flock on process exit even if Release is never
// called, so Release errors are only logged — they are never fatal.
func (l *Lock) Release() {
	if l == nil || l.file == nil {
		return
	}
	if err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN); err != nil {
		LogWarn("unlock", err)
	}
	if err := l.file.Close(); err != nil {
		LogWarn("close-lockfile", err)
	}
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		LogWarn("remove-lockfile", err)
	}
	l.file = nil
}

// HolderPID reads the PID stored in the named lock file without acquiring
// the lock. Returns 0 if the file doesn't exist or is unreadable. Used by
// the Taskfile dev tasks to find what to kill.
func HolderPID(name string) int {
	dir, err := lockDir()
	if err != nil {
		return 0
	}
	return readPID(filepath.Join(dir, name+".lock"))
}

// LockPath returns the absolute path of the named lock file. Used by the
// Taskfile so it can pgrep / pkill against the recorded pid.
func LockPath(name string) string {
	dir, err := lockDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, name+".lock")
}

func readPID(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return 0
	}
	return pid
}

func lockDir() (string, error) {
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = filepath.Join(os.TempDir(), fmt.Sprintf("dark-%d", os.Getuid()))
	}
	return filepath.Join(base, "dark"), nil
}
