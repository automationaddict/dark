# App Store

Browse the package repositories available on your Arch system: the official pacman repos (core, extra, multilib) and the Arch User Repository. Search, view full package details, see what's already installed, and learn about anything interesting before pulling it down.

The App Store is **read-only in this release.** Install, remove, and upgrade actions are planned for the next pass. You can browse and research packages today; actually installing them still happens from a regular terminal.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

The F2 tab lays out like a small Cosmic-style app store, translated into a strict TUI. From top to bottom:

1. A **Search bar** spanning the top of the tab. The cursor blinks here when you press `/`.
2. A **Categories** sidebar on the left listing the views you can drill into.
3. A **Results / Detail** pane on the right that tracks whatever the sidebar or search has selected.
4. A one-line **status footer** at the bottom showing either the key hint row, the current busy state, or a rate-limit warning.

On the first frame `darkd` has already pulled a fresh catalog, so the sidebar shows real counts for "All Packages" and "Installed". The Featured view populates from a small curated list of common apps that also exist on your system.

## The Search bar

The search bar is where you enter fuzzy queries against both pacman and the AUR. Its behavior:

- Press `/` from anywhere in the App Store to focus the bar. The border turns accent blue and the cursor block appears.
- Type freely. Every printable character including letters like `j`, `k`, `q` goes into the input — shortcut keys are suspended while the bar has focus.
- Press `enter` to run the search. Results come back as a merged list sorted with exact name matches first, installed packages next, pacman before AUR, then by name.
- Press `esc` to abandon the input without searching. The existing query (and its results) stay intact.
- Press `backspace` to erase the last character.
- The `[aur: on]` / `[aur: off]` badge on the right of the bar shows whether AUR results are included. Toggle it with `A` (shift+a) from outside search mode.

An empty query search returns you to whichever category is currently highlighted in the sidebar.

## The Categories sidebar

The sidebar owns the left column of the App Store body. Categories are divided into two tiers:

**Navigation views (always enabled)**

- **Featured** — A curated list of common end-user apps, filtered against your repos so nothing on the list is a ghost. Good for poking around without knowing what to search for.
- **All Packages** — Every package pacman knows about, in name order. Count in parentheses reflects your repo state.
- **Installed** — Every package currently on your system according to `pacman -Qq`. Also includes AUR packages you've already built locally.
- **AUR** — The Arch User Repository. Empty until you run a search with `[aur: on]` — the AUR has no browse-everything endpoint so we don't pull it wholesale.

**Content categories (enabled when populated)**

- **Development** — Compilers, IDEs, version control, databases, Docker, language runtimes.
- **Graphics** — Image editors, 3D modeling, photography, screenshot tools, vector graphics.
- **Internet** — Web browsers, email clients, chat apps, download tools, VPNs, file sync.
- **Multimedia** — Media players, video editors, audio workstations, codecs, streaming.
- **Office** — Document suites, PDF viewers, note-taking, finance, password managers.
- **System** — Terminals, shells, file managers, system monitors, boot loaders, firewalls, CLI tools.
- **Games** — Game launchers, emulators, native games, compatibility layers.
- **Other** — Everything that doesn't fit elsewhere (currently unused — packages without a category assignment are simply uncategorized rather than force-bucketed).

Categories are populated at catalog-build time from two sources: a Lua script (`categories.lua`) that maps package names to sidebar groups and maps XDG freedesktop category strings to dark's groups, and the `Categories=` field in installed packages' `.desktop` files. The Lua script takes priority, so you can override any categorization by editing your copy of the script.

### Customizing categories

The default `categories.lua` is compiled into the binary. To override it, create a file at:

    $XDG_CONFIG_HOME/dark/scripts/appstore/categories.lua

The user file completely replaces the default — dark does not merge them. The script must set two global tables: `xdg_map` (mapping XDG category strings like `"AudioVideo"` to dark sidebar IDs like `"multimedia"`) and `packages` (mapping package names like `"firefox"` to sidebar IDs). See the default script for the full format.

After editing the script, press `R` in the App Store to refresh the catalog and pick up the changes.

### Navigating the sidebar

- `j` / `down` — move selection down, skipping disabled categories
- `k` / `up` — move selection up, skipping disabled categories
- `enter` — load the selected category into the Results pane and shift focus to it
- `esc` — no-op from the sidebar (the global `esc` quits the app from here)

Cursor movement in the sidebar does **not** fire a search on every keystroke. Nothing happens until you press `enter`, so you can scroll through categories without flooding the daemon with requests.

## The Results pane

Once a category is loaded or a search has returned, the right side of the body fills with a list of package cards. Each row shows:

- A `▸` selection caret when that row is highlighted and focus is on the Results pane.
- **Name** — accent color when installed, normal color otherwise.
- **Size** — humanized installed size (MiB or GiB). Zero renders as `—`, which is common for AUR packages since the AUR has no binary to measure.
- **Origin badge** — `installed` in green for packages on your system, `AUR` in gold for AUR results, or the repo name (`core`, `extra`, `multilib`) for official-repo packages.
- **Description** — the short package description, truncated to fit the remaining width on the row.

### Navigating results

- `enter` on the sidebar moves focus into the Results pane when there's something to show.
- `j` / `k` walks the selection within the list.
- `enter` on a highlighted row opens the Detail pane for that package.
- `esc` returns focus to the sidebar.

### Why you see what you see

Results are sorted in this order, globally:

1. Exact name match (case-insensitive) — useful when searching for a specific tool.
2. Installed packages.
3. Pacman repos before AUR.
4. Alphabetical by name.

A search for `firefox` therefore returns `firefox` first (if it's installed), then other pacman packages containing "firefox", then any AUR results. The status footer shows `(results truncated)` if the search returned more rows than the 200-row window; refine the query to narrow it down.

## The Detail pane

Pressing `enter` on a highlighted package replaces the results list with a full detail readout. The top of the pane is a header with the package name and version plus the origin badge. Below that, a two-column metadata block shows:

- **Repo** — the pacman repository the package lives in, or `aur`
- **URL** — the upstream project URL
- **Licenses** — comma-separated license identifiers
- **Download** — compressed download size (pacman only — AUR has no binary to measure)
- **Installed** — uncompressed on-disk size after install
- **Updated** — how long ago the package was last built or modified
- **Votes / Popularity** — AUR-only engagement metrics
- **Maintainer** / **Packager** — who currently stewards the package

Below the metadata, longer list fields render one per line: **Description** (full text if distinct from the short one), **Depends On**, **Optional**, **Make Deps**, and **Conflicts**. Empty fields are omitted so packages with sparse metadata don't waste screen space.

Press `esc` to close the detail pane and return to the Results list.

## Global key reference

### From outside search mode

- `↑` / `↓` or `k` / `j` — move the selection within the currently focused region
- `enter` — activate the current region (sidebar → results, results → detail)
- `esc` — back out of the current region (detail → results → sidebar → quit)
- `/` — open the search bar
- `A` (shift+a) — toggle whether AUR results are included in searches
- `R` (shift+r) — refresh the catalog, bypassing caches
- `?` — open this help panel
- `q` — quit
- `ctrl+c` — quit from anywhere
- `ctrl+r` — rebuild dark in place
- `f1` … `f12` — switch tabs

### From inside search mode

While the search bar is focused, every printable key goes into the input. Only these commands are reserved:

- `enter` — commit the search and show results
- `esc` — cancel input, keep existing query and results
- `backspace` — delete the last character
- `f1` … `f12` — switch tabs (leaves search mode)
- `ctrl+c` — quit
- `?` — open help

## Common tasks

### Find a specific package

Press `/`, type the package name, press `enter`. The exact match lands first in the list. Press `enter` again to open the detail pane.

### See what's already installed on your system

Highlight **Installed** in the sidebar and press `enter`. The list loads from the local pacman database and matches pacman's own `-Qq` output, including AUR packages you've installed via makepkg or a helper.

### Include AUR results in a search

Press `A` from outside search mode to flip the AUR toggle on. The badge on the search bar flips from `[aur: off]` to `[aur: on]`. Your next search will fan out to both pacman and the aurweb RPC and merge the results.

### Refresh after installing something in another terminal

Press `R`. The daemon drops its catalog cache, runs `pacman -Sl` and `pacman -Qq` again, rebuilds the installed-set map, and pushes an updated snapshot. This also clears any AUR rate-limit state.

### Inspect a package before installing it

Navigate to the package, press `enter` to open detail, and read the Depends / Optional Deps / Conflicts lists and the Updated timestamp. A package last updated two years ago on an AUR entry with three votes is a different proposition from one updated last week with four thousand votes — the detail view puts both numbers on the same screen so you can decide quickly.

## AUR rate limiting

The aurweb RPC has no published rate limits but we treat it as a shared resource. Dark enforces these limits on itself:

- At most **2 inflight requests** to aurweb at any time.
- On HTTP 429 or 503 from aurweb, dark enters an exponential backoff starting at 10 seconds, doubling per consecutive failure, clamped to 5 minutes. If aurweb sends a `Retry-After` header we honor it instead.
- During backoff, search and detail calls that miss the local cache return immediately with a status-line warning rather than piling more requests on a throttled server.
- The cache clears a throttle on the first successful call after the retry window.

You'll know you've hit a limit when the status footer shows `AUR rate limited — retrying in Ns` in red. Dark does not retry automatically; the next search you run after the window elapses will try again.

## Status line meanings

The one-line footer under the main pane shows different messages depending on what's happening:

- `↑↓ nav  /  search  enter open  R refresh  A toggle AUR  ?  help` — the ambient hint row, shown when nothing is in flight.
- `working…` in accent — a search, detail fetch, or refresh is in flight.
- `(results truncated)` — the search returned more rows than the 200-row window; refine the query to see more.
- `AUR limited` in red — the AUR backend is currently in backoff. See the rate-limiting section above.
- A red error string — the most recent command reported an error; the exact text comes from darkd.

The hint row returns the next time you move the cursor or trigger another action.

## Data sources

Everything on this page comes from `darkd`, which reads:

- **pacman** via the `pacman -Sl`, `pacman -Si`, and `pacman -Qq` commands for repo enumeration, per-package detail, and installed state.
- **expac** when available, used to batch descriptions and installed sizes in a single call instead of running `pacman -Si` N times. Dark runs without expac but the browse list will show empty descriptions.
- **aurweb RPC v5** at `https://aur.archlinux.org/rpc/v5` for AUR search and detail lookups. All AUR calls go over HTTPS; dark does no unencrypted network traffic.
- **Lua scripting** via GopherLua. The `categories.lua` script is loaded at catalog-build time and provides the XDG-to-sidebar category mapping plus a curated per-package override table. The embedded default ships ~290 curated package entries and ~90 XDG category mappings. Users can override the script at `$XDG_CONFIG_HOME/dark/scripts/appstore/categories.lua`.
- **.desktop files** at `/usr/share/applications/*.desktop`. The `Categories=` field is parsed for installed packages and mapped through the Lua `xdg_map` table to assign categories to packages that ship a desktop entry but aren't in the curated list.
- **On-disk caches** under `$XDG_CACHE_HOME/dark/appstore` — 6 hours for the pacman catalog and 1 hour for AUR search/detail results. Categories are re-assigned from the Lua script on every catalog load (including cache hits) so script edits take effect on the next refresh without clearing the cache.

The daemon publishes a fresh catalog snapshot on `dark.appstore.catalog` every 60 seconds (much slower than wifi or bluetooth because the repo list barely moves) and pushes an updated snapshot immediately after a user-initiated refresh.

## Backend notes

The appstore service follows the same pluggable-Backend pattern as wifi and bluetooth. The production backend is a composite that fans requests out to a `pacmanBackend` and an `aurBackend`. If `pacman` is not present on PATH the service falls back to a `noopBackend` and the App Store renders an explanation instead of an empty catalog.

Install / remove / upgrade verbs are explicitly out of scope for this release and will arrive in the next pass, gated behind an elevation strategy that is still being designed.
