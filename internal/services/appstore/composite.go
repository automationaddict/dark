package appstore

import (
	"log/slog"
	"sync"
)

// compositeBackend fans requests out to a pacman backend and an AUR
// backend, merging the results into one snapshot/search/detail surface.
// Pacman is authoritative for the catalog (categories, counts, featured
// rows, installed state); AUR contributes only when the user explicitly
// opts in via SearchQuery.IncludeAUR or targets an AUR package by
// origin.
type compositeBackend struct {
	logger *slog.Logger
	pacman Backend
	aur    Backend
}

// NewCompositeBackend wires the two sources together. Either may be
// nil: a nil pacman means the composite is effectively the AUR alone
// (rare — the only way to hit this is if detect ran on a system
// without pacman), and a nil AUR means offline / no-network mode.
func NewCompositeBackend(logger *slog.Logger, pacman, aur Backend) Backend {
	if logger == nil {
		logger = slog.Default()
	}
	return &compositeBackend{
		logger: logger.With("backend", "composite"),
		pacman: pacman,
		aur:    aur,
	}
}

func (c *compositeBackend) Name() string {
	if c.pacman != nil && c.aur != nil {
		return BackendPacAUR
	}
	if c.pacman != nil {
		return BackendPacman
	}
	if c.aur != nil {
		return "aur"
	}
	return BackendNone
}

func (c *compositeBackend) Close() {
	if c.pacman != nil {
		c.pacman.Close()
	}
	if c.aur != nil {
		c.aur.Close()
	}
}

func (c *compositeBackend) Refresh() error {
	// Refresh both; surface the first error but still attempt the
	// other so a transient AUR failure doesn't leave pacman stale.
	var firstErr error
	if c.pacman != nil {
		if err := c.pacman.Refresh(); err != nil {
			firstErr = err
		}
	}
	if c.aur != nil {
		if err := c.aur.Refresh(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Snapshot returns the merged catalog payload. Pacman is the primary
// source: its categories, featured list, installed count, and repo
// count flow through verbatim. The AUR backend contributes rate-limit
// state and health info only.
func (c *compositeBackend) Snapshot() Snapshot {
	var snap Snapshot
	if c.pacman != nil {
		snap = c.pacman.Snapshot()
		snap.Backend = c.Name()
	} else {
		snap = Snapshot{
			Backend:    c.Name(),
			Categories: defaultCategories(),
		}
	}
	if c.aur != nil {
		aurSnap := c.aur.Snapshot()
		snap.AURHealthy = aurSnap.AURHealthy
		snap.AURLimit = aurSnap.AURLimit
	}
	return snap
}

// Search fans out to both backends concurrently when the caller opts
// into AUR results and the query has text. Results are merged with
// pacman rows first, then AUR rows, and the final list is re-sorted so
// installed and exact-match rows bubble to the top regardless of
// origin.
func (c *compositeBackend) Search(q SearchQuery) (SearchResult, error) {
	wantAUR := c.aur != nil && q.IncludeAUR && q.Text != ""

	if !wantAUR {
		if c.pacman == nil {
			return SearchResult{Query: q}, nil
		}
		return c.pacman.Search(q)
	}

	type pacResult struct {
		res SearchResult
		err error
	}
	type aurResult struct {
		res SearchResult
		err error
	}

	pacCh := make(chan pacResult, 1)
	aurCh := make(chan aurResult, 1)

	var wg sync.WaitGroup
	if c.pacman != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := c.pacman.Search(q)
			pacCh <- pacResult{res: res, err: err}
		}()
	} else {
		pacCh <- pacResult{}
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		res, err := c.aur.Search(q)
		aurCh <- aurResult{res: res, err: err}
	}()
	wg.Wait()
	close(pacCh)
	close(aurCh)

	pac := <-pacCh
	aur := <-aurCh
	if pac.err != nil {
		c.logger.Warn("search failed", "source", "pacman", "query", q.Text, "err", pac.err)
	}
	if aur.err != nil {
		c.logger.Warn("search failed", "source", "aur", "query", q.Text, "err", aur.err)
	}

	merged := make([]Package, 0, len(pac.res.Packages)+len(aur.res.Packages))
	merged = append(merged, pac.res.Packages...)
	merged = append(merged, aur.res.Packages...)
	SortResults(merged, q.Text)

	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	truncated := pac.res.Truncated || aur.res.Truncated
	if len(merged) > limit {
		merged = merged[:limit]
		truncated = true
	}
	return SearchResult{
		Query:     q,
		Packages:  merged,
		Truncated: truncated,
		AURLimit:  aur.res.AURLimit,
	}, nil
}

// Detail routes by origin. When origin is unset we try pacman first
// (the usual case — user clicked a row from the browse list that came
// from the official repos), then fall back to AUR on "not found".
func (c *compositeBackend) Detail(req DetailRequest) (Detail, error) {
	switch req.Origin {
	case OriginAUR:
		if c.aur == nil {
			return Detail{}, ErrBackendUnsupported
		}
		return c.aur.Detail(req)
	case OriginPacman:
		if c.pacman == nil {
			return Detail{}, ErrBackendUnsupported
		}
		return c.pacman.Detail(req)
	}
	if c.pacman != nil {
		if d, err := c.pacman.Detail(req); err == nil {
			return d, nil
		}
	}
	if c.aur != nil {
		return c.aur.Detail(req)
	}
	return Detail{}, ErrBackendUnsupported
}

// Install routes by the first package's origin when known. For
// pacman packages this goes through dark-helper + pkexec. For AUR
// packages it uses the detected AUR helper (paru/yay).
func (c *compositeBackend) Install(names []string) (string, error) {
	if c.pacman != nil {
		return c.pacman.Install(names)
	}
	return "", ErrBackendUnsupported
}

// InstallAUR installs AUR packages via the detected helper.
func (c *compositeBackend) installAUR(names []string) (string, error) {
	if c.aur != nil {
		return c.aur.Install(names)
	}
	return "", ErrBackendUnsupported
}

func (c *compositeBackend) Remove(names []string) (string, error) {
	if c.pacman != nil {
		return c.pacman.Remove(names)
	}
	return "", ErrBackendUnsupported
}

func (c *compositeBackend) Upgrade() (string, error) {
	if c.pacman != nil {
		return c.pacman.Upgrade()
	}
	return "", ErrBackendUnsupported
}

func (c *compositeBackend) AURHelper() string {
	if c.aur != nil {
		return c.aur.AURHelper()
	}
	return ""
}
