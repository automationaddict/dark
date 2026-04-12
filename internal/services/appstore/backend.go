package appstore

import "fmt"

// Backend abstracts the source of package data. The production
// implementation in phase 4 composes pacman and AUR behind one Backend
// value; detect.go picks between that composite and noop based on which
// tools are available on the host.
type Backend interface {
	Name() string

	// Snapshot returns the lightweight catalog payload: sidebar
	// categories, featured rows, installed count, and AUR health.
	Snapshot() Snapshot

	// Search runs a query against whichever backends this Backend
	// composes. Implementations must be safe to call concurrently with
	// Snapshot and Detail.
	Search(q SearchQuery) (SearchResult, error)

	// Detail returns the full readout for one package. The request's
	// Origin field disambiguates when the same name exists in both
	// pacman and the AUR.
	Detail(req DetailRequest) (Detail, error)

	// Refresh drops any cached state so the next Snapshot / Search sees
	// fresh data. Bound to the user-initiated refresh key in the TUI.
	Refresh() error

	// Install installs one or more packages via the privileged helper.
	// Returns the combined stdout/stderr output for the TUI status line.
	Install(names []string) (string, error)

	// Remove removes one or more packages via the privileged helper.
	Remove(names []string) (string, error)

	// Upgrade runs a full system upgrade (pacman -Syu) via the helper.
	Upgrade() (string, error)

	// AURHelper returns the name of the detected AUR helper (paru, yay)
	// or "" when none is installed. The TUI uses this to decide whether
	// to show an install button on AUR packages.
	AURHelper() string

	// Close releases any held resources (HTTP clients, file handles).
	Close()
}

// ErrBackendUnsupported is returned by backends that don't implement a
// particular operation. The TUI surfaces this as a calm "not available"
// rather than an error toast.
var ErrBackendUnsupported = fmt.Errorf("operation not supported by this appstore backend")
