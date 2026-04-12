package appstore

import (
	"bufio"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const desktopDir = "/usr/share/applications"

// desktopCategories reads every .desktop file under /usr/share/applications,
// extracts the Categories= field, resolves the owning package via
// `pacman -Qo`, and returns a map from package name to the raw XDG
// category strings (semicolon-delimited in the spec, returned as a
// cleaned slice here).
//
// The caller maps the XDG strings to dark sidebar IDs using the Lua
// xdg_map table. This function only handles the parsing — it doesn't
// import the scripting package so the dependency stays one-directional.
func desktopCategories(logger *slog.Logger) map[string][]string {
	result := make(map[string][]string)

	ownership := desktopOwnership(logger)
	if len(ownership) == 0 {
		return result
	}

	pattern := filepath.Join(desktopDir, "*.desktop")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return result
	}

	for _, path := range files {
		cats := parseDesktopCategories(path)
		if len(cats) == 0 {
			continue
		}
		base := filepath.Base(path)
		pkg, ok := ownership[base]
		if !ok {
			continue
		}
		if _, exists := result[pkg]; !exists {
			result[pkg] = cats
		}
	}
	logger.Debug("appstore: parsed .desktop categories",
		"files", len(files),
		"packages_with_categories", len(result))
	return result
}

// desktopOwnership runs `pacman -Qo /usr/share/applications/*.desktop`
// once and builds a map from desktop filename (e.g. "firefox.desktop")
// to the owning package name (e.g. "firefox"). This avoids N separate
// pacman invocations.
func desktopOwnership(logger *slog.Logger) map[string]string {
	pattern := filepath.Join(desktopDir, "*.desktop")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		return nil
	}

	args := append([]string{"-Qo"}, files...)
	out, err := runCommand("pacman", args...)
	if err != nil {
		logger.Debug("appstore: pacman -Qo failed for .desktop files", "err", err)
		return nil
	}

	// Output format: "/usr/share/applications/foo.desktop is owned by pkgname version"
	ownership := make(map[string]string, len(files))
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		idx := strings.Index(line, " is owned by ")
		if idx < 0 {
			continue
		}
		path := strings.TrimSpace(line[:idx])
		rest := strings.TrimSpace(line[idx+len(" is owned by "):])
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			continue
		}
		base := filepath.Base(path)
		ownership[base] = fields[0]
	}
	return ownership
}

// parseDesktopCategories reads a single .desktop file and returns the
// Categories= value split into individual strings. The XDG spec uses
// semicolons as separators with an optional trailing semicolon.
func parseDesktopCategories(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Categories=") {
			val := strings.TrimPrefix(line, "Categories=")
			val = strings.TrimSpace(val)
			if val == "" {
				return nil
			}
			parts := strings.Split(val, ";")
			cats := make([]string, 0, len(parts))
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					cats = append(cats, p)
				}
			}
			return cats
		}
	}
	return nil
}
