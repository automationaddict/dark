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

	if err := f.Truncate(0); err == nil {
		_, _ = f.WriteString(strconv.Itoa(os.Getpid()) + "\n")
		_ = f.Sync()
	}

	return &Lock{file: f, path: path}, nil
}

// Release unlocks the file and removes it. Safe to call multiple times.
func (l *Lock) Release() {
	if l == nil || l.file == nil {
		return
	}
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	_ = l.file.Close()
	_ = os.Remove(l.path)
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
