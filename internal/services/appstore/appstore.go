// Package appstore browses the package repositories available on an Arch
// system: the official pacman repos (core, extra, multilib) and the Arch
// User Repository. The package is read-only in phase 1 — search, detail
// lookup, and installed-state tagging only. Install, remove, and upgrade
// verbs are deferred to phase 2 so the layout, help docs, and snapshot
// shape can settle first.
//
// Architecture follows the same BLoC discipline as the wifi and bluetooth
// domains: all parsing, sorting, formatting, and caching lives behind the
// Backend interface, and the TUI only renders snapshots.
package appstore

// Origin tags which backend a package came from. The UI uses it to draw a
// small badge on each row so users know whether a result comes from the
// official repos or the AUR.
type Origin string

const (
	OriginPacman Origin = "pacman"
	OriginAUR    Origin = "aur"
)

// Package is one row in a search result or catalog listing. Rich detail
// (long description, deps, conflicts, etc.) is fetched separately via
// Detail so the search path stays cheap.
type Package struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Repo        string `json:"repo,omitempty"`
	Description string `json:"description,omitempty"`
	Origin      Origin `json:"origin"`

	// InstalledSize is the on-disk size in bytes once installed. Pacman
	// reports this via -Si. AUR uses the download size from the RPC as a
	// rough proxy; it is approximate.
	InstalledSize int64 `json:"installed_size,omitempty"`

	// Votes / Popularity are AUR-only; zero for pacman packages.
	Votes      int     `json:"votes,omitempty"`
	Popularity float64 `json:"popularity,omitempty"`

	// LastUpdatedUnix is the upstream last-modified timestamp in Unix
	// seconds. Zero when unknown.
	LastUpdatedUnix int64 `json:"last_updated_unix,omitempty"`

	// Installed is true when this package name appears in the local
	// pacman database, regardless of origin. An AUR package is marked
	// installed if the user installed it via makepkg / an AUR helper.
	Installed bool `json:"installed,omitempty"`
}

// Detail is the full readout for one package shown in the detail panel.
// Anything the backend can cheaply produce goes here — the panel
// degrades gracefully on empty fields.
type Detail struct {
	Package

	URL           string   `json:"url,omitempty"`
	Licenses      []string `json:"licenses,omitempty"`
	Maintainer    string   `json:"maintainer,omitempty"`
	Packager      string   `json:"packager,omitempty"`
	Groups        []string `json:"groups,omitempty"`
	Provides      []string `json:"provides,omitempty"`
	Depends       []string `json:"depends,omitempty"`
	OptDepends    []string `json:"opt_depends,omitempty"`
	MakeDepends   []string `json:"make_depends,omitempty"`
	CheckDepends  []string `json:"check_depends,omitempty"`
	Conflicts     []string `json:"conflicts,omitempty"`
	Replaces      []string `json:"replaces,omitempty"`
	DownloadSize  int64    `json:"download_size,omitempty"`
	BuildDateUnix int64    `json:"build_date_unix,omitempty"`
	LongDesc      string   `json:"long_desc,omitempty"`
}

// Category is a sidebar entry. In phase 1 only Featured, All, Installed,
// and AUR are populated; the named categories (Development, Graphics,
// etc.) remain in the snapshot as empty placeholders so the sidebar
// layout stays stable. Phase 2 will wire real category data from
// appstream metadata.
//
// TODO(appstore): revisit categories in phase 2 — parse /usr/share/metainfo
// and add a curated fallback map.
type Category struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Count   int    `json:"count"`
	Enabled bool   `json:"enabled"`
}

// RateLimitState lets the UI surface AUR throttling to the user. Active is
// true while the backend is in backoff; RetryAfterUnix is the earliest
// wall-clock time (Unix seconds) another request will be attempted.
// Message is a human-readable status line.
type RateLimitState struct {
	Active         bool   `json:"active,omitempty"`
	RetryAfterUnix int64  `json:"retry_after_unix,omitempty"`
	Message        string `json:"message,omitempty"`
}

// Snapshot is the catalog payload published on dark.appstore.catalog. It
// is intentionally light: counts and featured rows only. Full search
// results and detail are pulled on demand via the command subjects.
type Snapshot struct {
	Backend    string         `json:"backend"`
	Categories []Category     `json:"categories"`
	Featured   []Package      `json:"featured,omitempty"`
	Installed  int            `json:"installed_count,omitempty"`
	RepoCount  int            `json:"repo_count,omitempty"`
	AURHealthy bool           `json:"aur_healthy,omitempty"`
	AURLimit   RateLimitState `json:"aur_limit,omitempty"`
}

// SearchQuery describes a search request sent on dark.cmd.appstore.search.
// An empty Text returns the Featured rows for the given category (or all
// packages when Category is also empty). IncludeAUR gates whether the AUR
// backend is consulted — false means pacman-only, useful when the user
// has disabled network lookups or AUR is rate-limited.
type SearchQuery struct {
	Text       string `json:"text,omitempty"`
	Category   string `json:"category,omitempty"`
	IncludeAUR bool   `json:"include_aur,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// SearchResult is the response to a SearchQuery.
type SearchResult struct {
	Query    SearchQuery    `json:"query"`
	Packages []Package      `json:"packages"`
	Truncated bool          `json:"truncated,omitempty"`
	AURLimit RateLimitState `json:"aur_limit,omitempty"`
}

// DetailRequest asks the backend owning Origin for a full Detail on Name.
type DetailRequest struct {
	Name   string `json:"name"`
	Origin Origin `json:"origin,omitempty"`
}

// Backend identifiers used in Snapshot.Backend and detection.
const (
	BackendNone   = "none"
	BackendPacman = "pacman"
	BackendPacAUR = "pacman+aur"
)

// Service is the single entry point the daemon uses. It is constructed by
// detect.go in phase 4; phase 1 only declares the type so the interface
// surface is stable.
type Service struct {
	backend Backend
}

// NewServiceWithBackend is an internal constructor that lets detect.go
// wire a specific backend. External callers should use the package-level
// Detect helper (phase 4).
func NewServiceWithBackend(b Backend) *Service {
	return &Service{backend: b}
}

// Close releases backend resources. Safe to call on a nil or already-
// closed Service.
func (s *Service) Close() {
	if s == nil || s.backend == nil {
		return
	}
	s.backend.Close()
	s.backend = nil
}

// Snapshot returns the light catalog payload for the periodic publish.
func (s *Service) Snapshot() Snapshot {
	if s == nil || s.backend == nil {
		return Snapshot{Backend: BackendNone, Categories: defaultCategories()}
	}
	return s.backend.Snapshot()
}

// Search dispatches to the backend. A nil or noop service returns an
// empty result rather than an error so the UI renders a calm "no
// backend" state.
func (s *Service) Search(q SearchQuery) (SearchResult, error) {
	if s == nil || s.backend == nil {
		return SearchResult{Query: q}, nil
	}
	return s.backend.Search(q)
}

// Detail fetches the full readout for one package.
func (s *Service) Detail(req DetailRequest) (Detail, error) {
	if s == nil || s.backend == nil {
		return Detail{}, ErrBackendUnsupported
	}
	return s.backend.Detail(req)
}

// Refresh forces the backend to invalidate its caches. Used by the
// refresh command handler so users can bypass TTLs on demand.
func (s *Service) Refresh() error {
	if s == nil || s.backend == nil {
		return nil
	}
	return s.backend.Refresh()
}

// defaultCategories is the sidebar layout shared by all backends. The
// "real" named categories stay disabled in phase 1 — they render in the
// sidebar so the layout is stable but users can't focus them.
//
// TODO(appstore): enable named categories in phase 2 once appstream
// parsing is wired.
func defaultCategories() []Category {
	return []Category{
		{ID: "featured", Title: "Featured", Enabled: true},
		{ID: "all", Title: "All Packages", Enabled: true},
		{ID: "installed", Title: "Installed", Enabled: true},
		{ID: "aur", Title: "AUR", Enabled: true},
		{ID: "development", Title: "Development", Enabled: false},
		{ID: "graphics", Title: "Graphics", Enabled: false},
		{ID: "internet", Title: "Internet", Enabled: false},
		{ID: "multimedia", Title: "Multimedia", Enabled: false},
		{ID: "office", Title: "Office", Enabled: false},
		{ID: "system", Title: "System", Enabled: false},
		{ID: "games", Title: "Games", Enabled: false},
		{ID: "other", Title: "Other", Enabled: false},
	}
}

// DefaultCategories exposes the sidebar layout to callers that need to
// render a stable skeleton before the first snapshot arrives.
func DefaultCategories() []Category {
	return defaultCategories()
}
