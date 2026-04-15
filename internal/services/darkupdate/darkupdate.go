// Package darkupdate owns dark's self-update flow: it queries the
// GitHub Releases API for the latest published version, compares
// it against the injected sysinfo.DarkVersion, and can atomically
// replace the installed binaries with a newer release tarball.
//
// Everything happens under the user's home — the install
// convention is ~/.local/bin per install.sh, and the update code
// writes into the same prefix. No privileged operations are
// needed, so no pkexec.
package darkupdate

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/automationaddict/dark/internal/services/sysinfo"
)

// GitHub repo we poll for releases. Kept as a package-level
// const so tests can override via NewClient.
const defaultRepo = "automationaddict/dark"

// binaryNames is the set of files the release tarball ships. Any
// future split (e.g. adding a `darkctl` CLI) would extend this.
var binaryNames = []string{"dark", "darkd", "dark-helper"}

// Snapshot is the state the TUI renders on the self-update panel.
// Current is always sysinfo.DarkVersion at snapshot time. Latest
// and LatestPublished come from the last CheckLatest; they stay
// zero-valued until the first check runs.
type Snapshot struct {
	Current           string    `json:"current"`
	Latest            string    `json:"latest,omitempty"`
	LatestPublished   time.Time `json:"latest_published,omitempty"`
	LatestNotes       string    `json:"latest_notes,omitempty"`
	UpdateAvailable   bool      `json:"update_available"`
	LastCheckedAt     time.Time `json:"last_checked_at,omitempty"`
	LastCheckError    string    `json:"last_check_error,omitempty"`
	Applying          bool      `json:"applying,omitempty"`
	ApplyError        string    `json:"apply_error,omitempty"`
	InstalledAt       string    `json:"installed_at,omitempty"` // resolved dark binary path
}

// Client wraps the GitHub API + local filesystem state. Stateful
// because the snapshot carries "last checked" fields that persist
// across individual commands from the TUI. Methods are safe to
// call from a single goroutine (darkd's bus handlers are serial).
type Client struct {
	repo   string
	http   *http.Client
	cache  Snapshot
}

// NewClient returns a Client talking to the default repo with a
// bounded HTTP timeout so a stalled network never blocks the
// daemon's event loop.
func NewClient() *Client {
	return &Client{
		repo: defaultRepo,
		http: &http.Client{Timeout: 15 * time.Second},
		cache: Snapshot{
			Current:     sysinfo.DarkVersion,
			InstalledAt: installedDarkPath(),
		},
	}
}

// Snapshot returns the current cached state. Callers serialize
// this on the bus to the TUI.
func (c *Client) Snapshot() Snapshot {
	// Keep Current fresh — sysinfo.DarkVersion is a package var
	// that doesn't move at runtime but the cache should reflect
	// the latest truth in case the daemon hot-reloaded.
	c.cache.Current = sysinfo.DarkVersion
	c.cache.InstalledAt = installedDarkPath()
	return c.cache
}

// CheckLatest queries the GitHub /releases/latest endpoint, parses
// the response, and updates the cached snapshot with the result.
// Returns the snapshot so the handler can reply without racing
// the cache.
func (c *Client) CheckLatest() Snapshot {
	c.cache.LastCheckedAt = time.Now()
	c.cache.LastCheckError = ""

	rel, err := c.fetchLatestRelease()
	if err != nil {
		c.cache.LastCheckError = err.Error()
		return c.Snapshot()
	}

	latestTag := strings.TrimPrefix(rel.TagName, "v")
	c.cache.Latest = latestTag
	c.cache.LatestPublished = rel.PublishedAt
	c.cache.LatestNotes = rel.Body
	c.cache.UpdateAvailable = compareVersions(latestTag, sysinfo.DarkVersion) > 0
	return c.Snapshot()
}

// Apply downloads the release tarball for the given tag (or the
// cached Latest if tag is empty), verifies its SHA256 against the
// release's SHA256SUMS file, extracts the binaries, and atomically
// renames each one over the installed path at
// ~/.local/bin/<binary>. Returns an error only when a step fails;
// on success the binaries are replaced and the caller is expected
// to restart darkd so the new code takes over.
//
// The running darkd (which may be this very process) stays alive
// until its next fork/exec because Linux keeps the old inode
// alive for processes that have it mapped. The caller can exit
// cleanly after Apply returns and a supervisor (systemd user
// unit) will restart against the new binary.
func (c *Client) Apply(tag string) error {
	c.cache.Applying = true
	c.cache.ApplyError = ""
	defer func() { c.cache.Applying = false }()

	if tag == "" {
		tag = c.cache.Latest
	}
	if tag == "" {
		return c.failApply("no target tag — run CheckLatest first")
	}
	// Normalize — accept both "0.1.0" and "v0.1.0".
	fullTag := tag
	if !strings.HasPrefix(fullTag, "v") {
		fullTag = "v" + fullTag
	}

	tarballName := fmt.Sprintf("dark-%s-linux-amd64.tar.gz", fullTag)
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", c.repo, fullTag)

	cacheDir, err := cacheDir()
	if err != nil {
		return c.failApply(err.Error())
	}
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return c.failApply(fmt.Sprintf("create cache dir: %v", err))
	}

	tarballPath := filepath.Join(cacheDir, tarballName)
	sumsPath := filepath.Join(cacheDir, "SHA256SUMS")
	defer func() {
		// Best-effort cleanup. Leaving the tarball on disk is
		// fine if we can't remove it; the cache dir is user-
		// scope and bounded.
		_ = os.Remove(tarballPath)
		_ = os.Remove(sumsPath)
	}()

	if err := c.downloadTo(baseURL+"/"+tarballName, tarballPath); err != nil {
		return c.failApply(fmt.Sprintf("download tarball: %v", err))
	}
	if err := c.downloadTo(baseURL+"/SHA256SUMS", sumsPath); err != nil {
		return c.failApply(fmt.Sprintf("download checksums: %v", err))
	}

	if err := verifyChecksum(tarballPath, sumsPath, tarballName); err != nil {
		return c.failApply(fmt.Sprintf("checksum: %v", err))
	}

	extractDir := filepath.Join(cacheDir, "stage")
	_ = os.RemoveAll(extractDir)
	if err := os.MkdirAll(extractDir, 0o755); err != nil {
		return c.failApply(fmt.Sprintf("create stage dir: %v", err))
	}
	defer os.RemoveAll(extractDir)

	if err := extractTarball(tarballPath, extractDir); err != nil {
		return c.failApply(fmt.Sprintf("extract: %v", err))
	}

	if err := installBinaries(extractDir); err != nil {
		return c.failApply(fmt.Sprintf("install: %v", err))
	}

	return nil
}

func (c *Client) failApply(msg string) error {
	c.cache.ApplyError = msg
	return fmt.Errorf("%s", msg)
}

// ─── GitHub API ───────────────────────────────────────────────────

type ghRelease struct {
	TagName     string    `json:"tag_name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
}

func (c *Client) fetchLatestRelease() (ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", c.repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return ghRelease{}, err
	}
	// The v3 Accept header is what GitHub recommends for the
	// Releases API. Anonymous requests are rate-limited but the
	// update check runs on user demand, not a tick, so the limit
	// (60/hr) is plenty.
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "dark-self-update/"+sysinfo.DarkVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return ghRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return ghRelease{}, fmt.Errorf("GitHub API HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return ghRelease{}, fmt.Errorf("decode release: %w", err)
	}
	if rel.TagName == "" {
		return ghRelease{}, fmt.Errorf("release has no tag_name")
	}
	return rel, nil
}

func (c *Client) downloadTo(url, dest string) error {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "dark-self-update/"+sysinfo.DarkVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, url)
	}
	// Write to .part + rename so a partial download never lands
	// at the canonical path. The caller's verify step reads the
	// final path so the rename is all-or-nothing.
	tmp := dest + ".part"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

// ─── verification + extraction ────────────────────────────────────

// verifyChecksum reads SHA256SUMS and checks the line for tarball
// against the actual SHA256 of the downloaded file. SHA256SUMS
// uses the sha256sum format: "<hex>  <filename>".
func verifyChecksum(tarball, sumsFile, tarballName string) error {
	sumsBytes, err := os.ReadFile(sumsFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", sumsFile, err)
	}
	var expected string
	for _, line := range strings.Split(string(sumsBytes), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Format: "<hex>  <filename>" (two spaces for binary
		// mode, one for text; accept both via Fields).
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		if parts[1] == tarballName || strings.HasSuffix(parts[1], "/"+tarballName) {
			expected = parts[0]
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("no checksum for %s in SHA256SUMS", tarballName)
	}

	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("mismatch: expected %s, got %s", expected, got)
	}
	return nil
}

// extractTarball pulls every regular file out of the gzipped tar
// into destDir. It refuses entries with path traversal (`..`,
// absolute) so a malicious tarball can't write outside destDir.
func extractTarball(tarball, destDir string) error {
	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Clean(hdr.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			return fmt.Errorf("unsafe path in tarball: %s", hdr.Name)
		}
		out := filepath.Join(destDir, name)
		// Make sure the target directory exists — the release
		// tarball is flat today but defensive code makes it
		// safe if we ever add subdirs.
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		dst, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, tr); err != nil {
			dst.Close()
			return err
		}
		if err := dst.Close(); err != nil {
			return err
		}
	}
}

// installBinaries renames each expected binary from stageDir over
// the installed path in ~/.local/bin. The rename is atomic on the
// same filesystem, which is what we get since both the stage dir
// and ~/.local/bin live under $HOME.
//
// Linux allows replacing a running binary via rename — the kernel
// keeps the old inode alive for the already-exec'd process, and
// the new file takes the path. The running darkd will continue
// to execute its old code until it restarts.
func installBinaries(stageDir string) error {
	prefix, err := installPrefix()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(prefix, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", prefix, err)
	}

	for _, bin := range binaryNames {
		src := filepath.Join(stageDir, bin)
		if _, err := os.Stat(src); err != nil {
			return fmt.Errorf("tarball missing %s: %w", bin, err)
		}
		dst := filepath.Join(prefix, bin)
		// Two-phase install: write to a sibling ".new" path,
		// then rename over the live file. Keeps the swap atomic
		// and leaves the old file in place if the copy fails
		// halfway (no observable partial state).
		tmp := dst + ".new"
		if err := copyFile(src, tmp, 0o755); err != nil {
			return fmt.Errorf("stage %s: %w", bin, err)
		}
		if err := os.Rename(tmp, dst); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("install %s: %w", bin, err)
		}
	}
	return nil
}

// ─── helpers ──────────────────────────────────────────────────────

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func installPrefix() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "bin"), nil
}

func cacheDir() (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "dark", "update"), nil
}

// installedDarkPath tries to resolve the filesystem path of the
// currently-running dark binary so the UI can display it. Best
// effort — returns "" if the executable isn't resolvable.
func installedDarkPath() string {
	prefix, err := installPrefix()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(prefix, "dark")
	if _, err := os.Stat(candidate); err == nil {
		return candidate
	}
	return ""
}

// compareVersions returns -1/0/1 if a is less than / equal to /
// greater than b. Both inputs are plain semver with optional `v`
// prefix and optional `-dev` / `-rc1` suffix. Dev suffixes sort
// lower than the same base version.
//
// We roll our own instead of pulling in golang.org/x/mod/semver
// because the full package handles a lot of cases dark doesn't
// hit (build metadata, prerelease rules), and a 40-line helper
// keeps the service dependency-free.
func compareVersions(a, b string) int {
	return compareSemver(normalizeVersion(a), normalizeVersion(b))
}

func normalizeVersion(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	return s
}

func compareSemver(a, b string) int {
	// Split off any -dev / -rc / etc. suffix. A plain release
	// sorts higher than a prerelease of the same base.
	aBase, aPre := splitPrerelease(a)
	bBase, bPre := splitPrerelease(b)
	if c := compareNumericTriple(aBase, bBase); c != 0 {
		return c
	}
	switch {
	case aPre == "" && bPre == "":
		return 0
	case aPre == "":
		return 1
	case bPre == "":
		return -1
	}
	return strings.Compare(aPre, bPre)
}

func splitPrerelease(s string) (string, string) {
	if idx := strings.Index(s, "-"); idx >= 0 {
		return s[:idx], s[idx+1:]
	}
	return s, ""
}

func compareNumericTriple(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < 3; i++ {
		var ai, bi int
		if i < len(aParts) {
			ai = atoi(aParts[i])
		}
		if i < len(bParts) {
			bi = atoi(bParts[i])
		}
		if ai != bi {
			if ai < bi {
				return -1
			}
			return 1
		}
	}
	return 0
}

func atoi(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		n = n*10 + int(r-'0')
	}
	return n
}
