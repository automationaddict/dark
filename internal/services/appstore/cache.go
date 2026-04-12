package appstore

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// cacheDir returns $XDG_CACHE_HOME/dark/appstore (or ~/.cache/dark/appstore
// when XDG_CACHE_HOME is unset). The directory is created if missing.
// Returns an empty string and an error on failure; callers that rely on
// disk cache degrade gracefully to in-memory only when this returns ""
// because a missing cache dir is never fatal to browsing.
func cacheDir() (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".cache")
	}
	dir := filepath.Join(base, "dark", "appstore")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// cacheEntry is the on-disk envelope wrapping any JSON payload with a
// stored-at timestamp used for TTL checks.
type cacheEntry[T any] struct {
	StoredAtUnix int64 `json:"stored_at_unix"`
	Payload      T     `json:"payload"`
}

// readCache loads a typed payload from disk if it exists and is within
// ttl. ok is false when there is no usable cache entry, whether because
// the file is missing, unreadable, malformed, or stale. Callers treat
// ok=false as "fetch fresh data."
func readCache[T any](path string, ttl time.Duration) (payload T, ok bool) {
	b, err := os.ReadFile(path)
	if err != nil {
		return payload, false
	}
	var env cacheEntry[T]
	if err := json.Unmarshal(b, &env); err != nil {
		return payload, false
	}
	if ttl > 0 {
		age := time.Since(time.Unix(env.StoredAtUnix, 0))
		if age > ttl {
			return payload, false
		}
	}
	return env.Payload, true
}

// writeCache serializes payload with a current timestamp and writes it
// atomically via a temp-file + rename. Errors are returned but callers
// generally log-and-continue since a cache miss is always survivable.
func writeCache[T any](path string, payload T) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	env := cacheEntry[T]{
		StoredAtUnix: time.Now().Unix(),
		Payload:      payload,
	}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
