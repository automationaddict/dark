package appstore

// noopBackend is the fallback returned by detect when neither pacman nor
// any other source is usable. It keeps the TUI renderable by returning
// an empty but well-formed Snapshot, and surfaces ErrBackendUnsupported
// on any actionable call so error states are unambiguous.
type noopBackend struct {
	name string
}

// NewNoopBackend returns a Backend that does nothing. The name is
// embedded in Snapshot.Backend so the TUI can show what was detected
// (or the absence of anything) in the status line.
func NewNoopBackend(name string) Backend {
	if name == "" {
		name = BackendNone
	}
	return &noopBackend{name: name}
}

func (n *noopBackend) Name() string { return n.name }

func (n *noopBackend) Snapshot() Snapshot {
	return Snapshot{
		Backend:    n.name,
		Categories: defaultCategories(),
	}
}

func (n *noopBackend) Search(q SearchQuery) (SearchResult, error) {
	return SearchResult{Query: q}, nil
}

func (n *noopBackend) Detail(req DetailRequest) (Detail, error) {
	return Detail{}, ErrBackendUnsupported
}

func (n *noopBackend) Refresh() error { return nil }

func (n *noopBackend) Install([]string) (string, error) { return "", ErrBackendUnsupported }
func (n *noopBackend) Remove([]string) (string, error)  { return "", ErrBackendUnsupported }
func (n *noopBackend) Upgrade() (string, error)         { return "", ErrBackendUnsupported }
func (n *noopBackend) AURHelper() string                { return "" }

func (n *noopBackend) Close() {}
