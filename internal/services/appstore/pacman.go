package appstore

import (
	"fmt"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/johnnelson/dark/internal/scripting"
)

const (
	pacmanCatalogTTL       = 6 * time.Hour
	pacmanInstalledTTL     = 30 * time.Second
	pacmanCatalogCacheFile = "pacman_catalog.json"
)

// pacmanBackend is the appstore Backend backed by the host pacman
// installation plus optional expac for fast batch metadata.
type pacmanBackend struct {
	logger *slog.Logger
	engine *scripting.Engine
	cache  string

	mu               sync.RWMutex
	catalog          []Package
	index            map[string]int // lower(name) -> catalog position
	installed        map[string]struct{}
	catCounts        map[string]int // sidebar ID → count of packages
	featured         []string       // curated package names from Lua
	lastLoad         time.Time
	lastInstallCheck time.Time
	expacAvailable   bool
}

// NewPacmanBackend constructs the pacman-backed Backend. It probes for
// expac at construction time; absence is non-fatal and logged at info.
// The first call to Snapshot or Search triggers catalog population, so
// construction is cheap.
func (p *pacmanBackend) Install(names []string) (string, error) {
	out, err := helperInstall(names)
	if err == nil {
		p.Refresh()
	}
	return out, err
}

func (p *pacmanBackend) Remove(names []string) (string, error) {
	out, err := helperRemove(names)
	if err == nil {
		p.Refresh()
	}
	return out, err
}

func (p *pacmanBackend) Upgrade() (string, error) {
	out, err := helperUpgrade()
	if err == nil {
		p.Refresh()
	}
	return out, err
}

func (p *pacmanBackend) AURHelper() string { return detectAURHelper() }

func NewPacmanBackend(logger *slog.Logger, engine *scripting.Engine) Backend {
	if logger == nil {
		logger = slog.Default()
	}
	dir, err := cacheDir()
	if err != nil {
		logger.Info("appstore: cache dir unavailable, running in-memory only", "err", err)
	}
	p := &pacmanBackend{
		logger: logger,
		engine: engine,
		cache:  dir,
	}
	if _, err := exec.LookPath("expac"); err == nil {
		p.expacAvailable = true
	} else {
		logger.Info("appstore: expac not found, catalog will ship without descriptions until phase 2")
	}
	return p
}

func (p *pacmanBackend) Name() string { return BackendPacman }

func (p *pacmanBackend) Close() {}

func (p *pacmanBackend) Refresh() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.catalog = nil
	p.index = nil
	p.installed = nil
	p.catCounts = nil
	p.lastLoad = time.Time{}
	if p.cache != "" {
		_ = removeIfExists(filepath.Join(p.cache, pacmanCatalogCacheFile))
	}
	p.logger.Info("appstore: pacman catalog cache cleared")
	return nil
}

func (p *pacmanBackend) Snapshot() Snapshot {
	p.ensureCatalog()
	p.mu.RLock()
	defer p.mu.RUnlock()

	cats := defaultCategories()
	for i := range cats {
		switch {
		case cats[i].ID == "all":
			cats[i].Count = len(p.catalog)
		case cats[i].ID == "installed":
			cats[i].Count = len(p.installed)
		case namedCategoryIDs[cats[i].ID]:
			if c, ok := p.catCounts[cats[i].ID]; ok && c > 0 {
				cats[i].Count = c
				cats[i].Enabled = true
			}
		}
	}

	return Snapshot{
		Backend:    BackendPacman,
		Categories: cats,
		Featured:   p.featuredLocked(),
		Installed:  len(p.installed),
		RepoCount:  len(p.catalog),
	}
}

func (p *pacmanBackend) Search(q SearchQuery) (SearchResult, error) {
	p.ensureCatalog()
	p.mu.RLock()
	defer p.mu.RUnlock()

	limit := q.Limit
	if limit <= 0 {
		limit = 200
	}
	text := strings.ToLower(strings.TrimSpace(q.Text))
	matches := make([]Package, 0, limit)
	truncated := false

	source := p.catalog
	if q.Category == "installed" {
		source = p.filterInstalledLocked()
	}
	if q.Category == "featured" && text == "" {
		return SearchResult{
			Query:    q,
			Packages: p.featuredLocked(),
		}, nil
	}
	filterByCat := namedCategoryIDs[q.Category]

	for _, pkg := range source {
		if filterByCat && pkg.Category != q.Category {
			continue
		}
		if text != "" {
			if !strings.Contains(strings.ToLower(pkg.Name), text) &&
				!strings.Contains(strings.ToLower(pkg.Description), text) {
				continue
			}
		}
		if len(matches) >= limit {
			truncated = true
			break
		}
		matches = append(matches, pkg)
	}
	SortResults(matches, q.Text)
	return SearchResult{
		Query:     q,
		Packages:  matches,
		Truncated: truncated,
	}, nil
}

func (p *pacmanBackend) Detail(req DetailRequest) (Detail, error) {
	if req.Name == "" {
		return Detail{}, fmt.Errorf("appstore: empty detail request")
	}
	out, err := runCommand("pacman", "-Si", req.Name)
	if err != nil {
		return Detail{}, fmt.Errorf("pacman -Si %s: %w", req.Name, err)
	}
	detail, err := parsePacmanSi(out)
	if err != nil {
		return Detail{}, err
	}
	detail.Origin = OriginPacman
	p.mu.RLock()
	_, installed := p.installed[detail.Name]
	p.mu.RUnlock()
	detail.Installed = installed
	return detail, nil
}

// ensureCatalog lazily populates the catalog on first use. Subsequent
// calls within pacmanCatalogTTL are no-ops. Concurrent callers are
// serialized on the write lock so only one population runs.
func (p *pacmanBackend) ensureCatalog() {
	p.mu.RLock()
	catalogFresh := p.catalog != nil && time.Since(p.lastLoad) < pacmanCatalogTTL
	installedStale := p.catalog != nil && time.Since(p.lastInstallCheck) >= pacmanInstalledTTL
	p.mu.RUnlock()
	if catalogFresh && !installedStale {
		return
	}
	if catalogFresh && installedStale {
		p.mu.Lock()
		p.refreshInstalledLocked()
		p.lastInstallCheck = time.Now()
		p.mu.Unlock()
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.catalog != nil && time.Since(p.lastLoad) < pacmanCatalogTTL {
		return
	}

	if p.cache != "" {
		if cached, ok := readCache[[]Package](filepath.Join(p.cache, pacmanCatalogCacheFile), pacmanCatalogTTL); ok {
			p.logger.Debug("appstore: loaded pacman catalog from disk cache", "count", len(cached))
			cm := loadCategoryMaps(p.engine, p.logger)
			desktop := desktopCategories(p.logger)
			p.catCounts = assignCategories(cached, cm, desktop)
			p.featured = cm.featured
			p.installCatalogLocked(cached)
			p.lastLoad = time.Now()
			p.refreshInstalledLocked()
			p.lastInstallCheck = p.lastLoad
			return
		}
	}

	start := time.Now()
	cat, err := p.buildCatalogLocked()
	if err != nil {
		p.logger.Error("appstore: failed to build pacman catalog", "err", err)
		return
	}
	p.installCatalogLocked(cat)
	p.lastLoad = time.Now()
	p.refreshInstalledLocked()
	p.lastInstallCheck = p.lastLoad
	p.logger.Info("appstore: built pacman catalog",
		"packages", len(cat),
		"installed", len(p.installed),
		"expac", p.expacAvailable,
		"elapsed", time.Since(start))

	if p.cache != "" {
		if err := writeCache(filepath.Join(p.cache, pacmanCatalogCacheFile), cat); err != nil {
			p.logger.Warn("appstore: failed to write pacman catalog cache", "err", err)
		}
	}
}

func (p *pacmanBackend) buildCatalogLocked() ([]Package, error) {
	slOut, err := runCommand("pacman", "-Sl")
	if err != nil {
		return nil, fmt.Errorf("pacman -Sl: %w", err)
	}
	cat := parsePacmanSl(slOut)
	if p.expacAvailable && len(cat) > 0 {
		enrichWithExpac(cat, p.logger)
	}
	cm := loadCategoryMaps(p.engine, p.logger)
	desktop := desktopCategories(p.logger)
	p.catCounts = assignCategories(cat, cm, desktop)
	p.featured = cm.featured
	p.logger.Info("appstore: categories assigned",
		"categorized", countCategorized(cat),
		"featured", len(cm.featured),
		"counts", p.catCounts)
	return cat, nil
}

func countCategorized(cat []Package) int {
	n := 0
	for _, pkg := range cat {
		if pkg.Category != "" {
			n++
		}
	}
	return n
}

func (p *pacmanBackend) installCatalogLocked(cat []Package) {
	p.catalog = cat
	p.index = make(map[string]int, len(cat))
	for i, pkg := range cat {
		p.index[strings.ToLower(pkg.Name)] = i
	}
}

func (p *pacmanBackend) refreshInstalledLocked() {
	out, err := runCommand("pacman", "-Qq")
	if err != nil {
		p.logger.Warn("appstore: pacman -Qq failed", "err", err)
		return
	}
	installed := make(map[string]struct{}, 512)
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" {
			continue
		}
		installed[name] = struct{}{}
	}
	p.installed = installed
	for i := range p.catalog {
		if _, ok := installed[p.catalog[i].Name]; ok {
			p.catalog[i].Installed = true
		}
	}
}

func (p *pacmanBackend) filterInstalledLocked() []Package {
	out := make([]Package, 0, len(p.installed))
	for _, pkg := range p.catalog {
		if pkg.Installed {
			out = append(out, pkg)
		}
	}
	return out
}

func (p *pacmanBackend) featuredLocked() []Package {
	out := make([]Package, 0, len(p.featured))
	for _, name := range p.featured {
		if idx, ok := p.index[name]; ok {
			out = append(out, p.catalog[idx])
		}
	}
	return out
}
