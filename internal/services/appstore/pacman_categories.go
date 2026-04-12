package appstore

import (
	"log/slog"

	lua "github.com/yuin/gopher-lua"

	"github.com/johnnelson/dark/internal/scripting"
)

const categoriesScript = "appstore/categories.lua"

// categoryMaps holds the two lookup tables loaded from Lua: the
// per-package overrides (highest priority) and the XDG-to-sidebar
// mapping used to interpret .desktop Categories= fields.
type categoryMaps struct {
	packages map[string]string // package name → sidebar ID
	xdgMap   map[string]string // XDG category → sidebar ID
}

// loadCategoryMaps loads categories.lua and extracts the two globals.
// Returns empty maps (not nil) on any error so callers don't need nil
// checks — categories just won't be populated, which degrades
// gracefully to all-disabled in the sidebar.
func loadCategoryMaps(engine *scripting.Engine, logger *slog.Logger) categoryMaps {
	empty := categoryMaps{
		packages: make(map[string]string),
		xdgMap:   make(map[string]string),
	}
	if engine == nil {
		return empty
	}
	if err := engine.LoadScript(categoriesScript); err != nil {
		logger.Warn("appstore: failed to load categories script", "err", err)
		return empty
	}
	cm := empty
	if v := engine.GetGlobal("packages"); v != lua.LNil {
		if t, ok := v.(*lua.LTable); ok {
			cm.packages = scripting.TableToStringMap(t)
		}
	}
	if v := engine.GetGlobal("xdg_map"); v != lua.LNil {
		if t, ok := v.(*lua.LTable); ok {
			cm.xdgMap = scripting.TableToStringMap(t)
		}
	}
	logger.Info("appstore: loaded category maps from Lua",
		"curated_packages", len(cm.packages),
		"xdg_entries", len(cm.xdgMap))
	return cm
}

// assignCategories tags every package in the catalog with a sidebar
// category ID. Priority order:
//
//  1. Lua per-package map (the curated overrides a user can customize)
//  2. .desktop file XDG categories mapped through the Lua xdg_map
//  3. Fallback to "" (uncategorized — counted under "other" by the
//     snapshot builder)
//
// The function modifies the catalog in place and returns a count map
// so the caller can populate Category.Count on the sidebar entries.
func assignCategories(cat []Package, cm categoryMaps, desktop map[string][]string) map[string]int {
	counts := make(map[string]int)
	for i := range cat {
		name := cat[i].Name
		if id, ok := cm.packages[name]; ok && namedCategoryIDs[id] {
			cat[i].Category = id
			counts[id]++
			continue
		}
		if xdgCats, ok := desktop[name]; ok {
			if id := firstXDGMatch(xdgCats, cm.xdgMap); id != "" {
				cat[i].Category = id
				counts[id]++
				continue
			}
		}
	}
	return counts
}

// firstXDGMatch iterates a package's XDG categories and returns the
// first dark sidebar ID that matches in the xdg_map. Returns "" when
// nothing matches.
func firstXDGMatch(xdgCats []string, xdgMap map[string]string) string {
	for _, xdg := range xdgCats {
		if id, ok := xdgMap[xdg]; ok && namedCategoryIDs[id] {
			return id
		}
	}
	return ""
}
