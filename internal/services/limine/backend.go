package limine

import "fmt"

// Backend abstracts the concrete limine integration. There is currently
// one real implementation (snapper + pkexec) but the seam matches the
// other services so a future implementation (e.g. a privileged helper
// binary, or a different bootloader entirely) can drop in without
// touching the daemon or the TUI.
type Backend interface {
	Name() string
	Snapshot() Snapshot

	CreateSnapshot(description string) error
	DeleteSnapshot(number int) error
	Sync() error

	SetDefaultEntry(entry int) error
	SetBootConfig(key, value string) error
	SetSyncConfig(key, value string) error
	SetOmarchyConfig(key, value string) error
	SetOmarchyKernelCmdline(lines []string) error

	Close()
}

// ErrBackendUnsupported is returned by backends that don't implement a
// particular operation.
var ErrBackendUnsupported = fmt.Errorf("operation not supported by this limine backend")
