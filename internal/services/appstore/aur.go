package appstore

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	aurSearchCacheTTL = 1 * time.Hour
	aurDetailCacheTTL = 1 * time.Hour
	aurMaxInfoArgs    = 150 // documented aurweb RPC v5 cap is ~200; leave headroom
)

// aurBackend talks to aurweb's RPC v5 endpoint. It is always composed
// with a pacman backend in phase 4 — on its own, Snapshot would be
// nearly empty because the AUR has no concept of a browsable catalog.
// The interesting bits here are: the HTTP client with rate-limit
// discipline, on-disk caching, and a RateLimitState that the composite
// backend surfaces up to the UI.
type aurBackend struct {
	logger *slog.Logger
	cache  string
	client *aurClient

	mu        sync.RWMutex
	lastLimit RateLimitState
}

// NewAURBackend constructs the AUR-backed Backend. It does not make
// any network calls during construction; rate-limit state starts
// cleared and lazily updates on first request.
func NewAURBackend(logger *slog.Logger) Backend {
	if logger == nil {
		logger = slog.Default()
	}
	dir, err := cacheDir()
	if err != nil {
		logger.Info("appstore: cache dir unavailable for AUR, running in-memory only", "err", err)
	}
	return &aurBackend{
		logger: logger,
		cache:  dir,
		client: newAURClient(logger),
	}
}

func (a *aurBackend) Name() string { return "aur" }

func (a *aurBackend) Close() {}

func (a *aurBackend) Install(names []string) (string, error) {
	helper := detectAURHelper()
	if helper == "" {
		return "", fmt.Errorf("no AUR helper installed (install paru or yay)")
	}
	var allOut string
	for _, name := range names {
		out, err := aurInstall(helper, name)
		allOut += out
		if err != nil {
			return allOut, err
		}
	}
	return allOut, nil
}

func (a *aurBackend) Remove(names []string) (string, error) {
	return helperRemove(names)
}

func (a *aurBackend) Upgrade() (string, error) { return "", ErrBackendUnsupported }

func (a *aurBackend) AURHelper() string { return detectAURHelper() }

func (a *aurBackend) Refresh() error {
	if a.cache == "" {
		return nil
	}
	dir := filepath.Join(a.cache, "aur")
	if err := removeDirContents(dir); err != nil {
		a.logger.Warn("appstore: failed to clear AUR cache", "err", err)
		return err
	}
	a.logger.Info("appstore: AUR cache cleared")
	return nil
}

// Snapshot returns AUR health info only — the AUR has no catalog, so
// counts and featured rows are left to the pacman side. The composite
// backend in phase 4 merges this into the main Snapshot.
func (a *aurBackend) Snapshot() Snapshot {
	a.mu.RLock()
	limit := a.lastLimit
	a.mu.RUnlock()
	return Snapshot{
		Backend:    "aur",
		Categories: defaultCategories(),
		AURHealthy: !limit.Active,
		AURLimit:   limit,
	}
}

// Search hits the AUR RPC search endpoint. Empty queries are a no-op —
// the AUR has no "browse everything" affordance and pulling every
// package would be a multi-megabyte transfer. The phase 8 TUI gates
// AUR lookups behind a user-typed query.
func (a *aurBackend) Search(q SearchQuery) (SearchResult, error) {
	text := strings.TrimSpace(q.Text)
	if text == "" {
		return SearchResult{Query: q, AURLimit: a.currentLimit()}, nil
	}

	cachePath := a.searchCachePath(text)
	if cachePath != "" {
		if cached, ok := readCache[SearchResult](cachePath, aurSearchCacheTTL); ok {
			a.logger.Debug("appstore: AUR search cache hit", "query", text, "count", len(cached.Packages))
			cached.Query = q
			cached.AURLimit = a.currentLimit()
			return cached, nil
		}
	}

	if limit := a.currentLimit(); limit.Active {
		a.logger.Info("appstore: AUR search skipped due to rate limit", "query", text, "retry_after_unix", limit.RetryAfterUnix)
		return SearchResult{Query: q, AURLimit: limit}, nil
	}

	rows, err := a.client.search(text)
	if err != nil {
		a.recordError(err)
		return SearchResult{Query: q, AURLimit: a.currentLimit()}, nil
	}
	a.clearLimit()

	pkgs := make([]Package, 0, len(rows))
	for _, r := range rows {
		pkgs = append(pkgs, r.toPackage())
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	truncated := false
	if len(pkgs) > limit {
		pkgs = pkgs[:limit]
		truncated = true
	}

	result := SearchResult{
		Query:     q,
		Packages:  pkgs,
		Truncated: truncated,
	}
	if cachePath != "" {
		if err := writeCache(cachePath, result); err != nil {
			a.logger.Warn("appstore: AUR search cache write failed", "err", err)
		}
	}
	result.AURLimit = a.currentLimit()
	return result, nil
}

// Detail fetches full package info via the RPC info endpoint. A single
// call handles the detail case; the batched Info path is available for
// future callers that need to enrich many packages at once.
func (a *aurBackend) Detail(req DetailRequest) (Detail, error) {
	if req.Name == "" {
		return Detail{}, fmt.Errorf("appstore: empty AUR detail request")
	}
	cachePath := a.detailCachePath(req.Name)
	if cachePath != "" {
		if cached, ok := readCache[Detail](cachePath, aurDetailCacheTTL); ok {
			a.logger.Debug("appstore: AUR detail cache hit", "name", req.Name)
			return cached, nil
		}
	}
	if limit := a.currentLimit(); limit.Active {
		return Detail{}, fmt.Errorf("AUR rate limited: %s", limit.Message)
	}

	rows, err := a.client.info([]string{req.Name})
	if err != nil {
		a.recordError(err)
		return Detail{}, err
	}
	a.clearLimit()
	if len(rows) == 0 {
		return Detail{}, fmt.Errorf("appstore: AUR has no package named %q", req.Name)
	}
	detail := rows[0].toDetail()
	if cachePath != "" {
		if err := writeCache(cachePath, detail); err != nil {
			a.logger.Warn("appstore: AUR detail cache write failed", "err", err)
		}
	}
	return detail, nil
}

// currentLimit is a read-locked accessor for the rate limit state.
func (a *aurBackend) currentLimit() RateLimitState {
	a.mu.RLock()
	defer a.mu.RUnlock()
	limit := a.lastLimit
	if limit.Active && time.Now().Unix() >= limit.RetryAfterUnix {
		return RateLimitState{}
	}
	return limit
}

func (a *aurBackend) clearLimit() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lastLimit = RateLimitState{}
}

// recordError classifies the given error and, if it is a rate-limit
// signal, sets the active backoff on the backend. Non-throttle errors
// (network down, DNS failure, 5xx that isn't 503) are logged but do
// not change the limit state — the next request tries again.
func (a *aurBackend) recordError(err error) {
	if err == nil {
		return
	}
	retryAfter, throttled := classifyAURError(err)
	if !throttled {
		a.logger.Warn("appstore: AUR request failed", "err", err)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if retryAfter <= 0 {
		// Exponential backoff: double from the previous window,
		// clamped to 5 minutes. If we weren't already throttled,
		// start at 10 seconds.
		prev := time.Until(time.Unix(a.lastLimit.RetryAfterUnix, 0))
		if prev <= 0 {
			prev = 10 * time.Second
		} else {
			prev *= 2
		}
		if prev > 5*time.Minute {
			prev = 5 * time.Minute
		}
		retryAfter = prev
	}
	a.lastLimit = RateLimitState{
		Active:         true,
		RetryAfterUnix: time.Now().Add(retryAfter).Unix(),
		Message:        fmt.Sprintf("AUR rate limited — retrying in %s", retryAfter.Round(time.Second)),
	}
	a.logger.Warn("appstore: AUR backing off",
		"retry_after", retryAfter,
		"err", err)
}

// searchCachePath derives a stable on-disk path for a search query.
// The key is a sha256 of the normalized lowercased query text, which
// gives us unique paths without worrying about path-unsafe characters
// in user input.
func (a *aurBackend) searchCachePath(text string) string {
	if a.cache == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.ToLower(text)))
	return filepath.Join(a.cache, "aur", "search", hex.EncodeToString(sum[:])+".json")
}

func (a *aurBackend) detailCachePath(name string) string {
	if a.cache == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(strings.ToLower(name)))
	return filepath.Join(a.cache, "aur", "detail", hex.EncodeToString(sum[:])+".json")
}
