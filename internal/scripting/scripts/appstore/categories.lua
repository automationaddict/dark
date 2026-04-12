-- categories.lua — loads curated App Store data from YAML and exposes
-- it as globals for the appstore backend.
--
-- This script is loaded by the daemon at catalog-build time. It reads
-- categories.yaml via load_yaml() and sets three globals:
--
--   xdg_map      XDG/freedesktop category string → dark sidebar ID
--   packages     package name → dark sidebar ID (highest-priority override)
--   featured     ordered list of package names for the Featured sidebar view
--
-- All curated data lives in categories.yaml — this script is the
-- processing layer. Users can override either file independently:
--
--   $XDG_CONFIG_HOME/dark/scripts/appstore/categories.yaml  (data)
--   $XDG_CONFIG_HOME/dark/scripts/appstore/categories.lua   (logic)
--
-- Override the YAML to add/remove packages or categories.
-- Override the Lua to change how the data is processed (e.g. merge
-- multiple YAML files, apply conditional logic, etc).

local data = load_yaml("appstore/categories.yaml")

xdg_map  = data.xdg_map  or {}
packages = data.packages  or {}
featured = data.featured  or {}

-- User hook: if you want to add entries without replacing the whole
-- YAML, override this script and append to the tables after loading:
--
--   local data = load_yaml("appstore/categories.yaml")
--   xdg_map  = data.xdg_map  or {}
--   packages = data.packages  or {}
--   featured = data.featured  or {}
--
--   -- Add your custom entries:
--   packages["my-custom-tool"] = "development"
--   table.insert(featured, 1, "my-custom-tool")
