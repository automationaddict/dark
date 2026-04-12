# App Store

Browse, search, install, and remove packages from the official Arch repos (core, extra, multilib) and the Arch User Repository (AUR). The App Store gives you a visual way to explore what's available on your system, see what's already installed, read full package details, and manage packages — all without leaving dark.

Press `?` at any time to open this help. Press `esc` to close it.

## What you see when you land here

Press `F2` to switch to the App Store tab. The screen is divided into four regions from top to bottom:

1. **Search bar** — spans the full width at the top. Shows your current search query (or a placeholder prompt) and an AUR toggle badge on the right.
2. **Categories sidebar** — a narrow column on the left listing the views you can browse: Featured, All Packages, Installed, AUR, plus content categories like Development, Graphics, Internet, and so on.
3. **Main pane** — the large area to the right of the sidebar. It shows either a **Results list** (package rows you can scroll through) or a **Detail panel** (the full readout for one package), depending on what you've selected.
4. **Status footer** — a single line at the bottom that shows keyboard hints, your position in the list (e.g. `3/142`), busy indicators, or error messages.

When you first open the tab, the daemon has already loaded the full catalog from pacman, so the sidebar shows real counts and you can start browsing immediately.

## Understanding focus — how to navigate

The App Store uses a **focus model** where one region at a time owns the keyboard. This is different from a mouse-driven GUI — you move focus between regions with specific keys and the status footer always shows which keys are available.

Here is the focus flow, step by step:

1. **You start on the sidebar.** The `j` and `k` keys (or arrow keys) move the highlight between categories. The status footer shows: `j/k categories · enter browse · / search · R refresh · A toggle AUR · U upgrade`.
2. **Press `enter` to browse a category.** Focus shifts to the Results list on the right. The sidebar highlight stays put so you can see which category you're viewing. The footer changes to: `j/k nav · enter detail · i install · X remove · / search · R refresh · U upgrade · 1/55`.
3. **Press `enter` on a result to see its details.** The Results list is replaced by the Detail panel for that package. The `j` and `k` keys now scroll the detail content up and down. The footer changes to: `i install · X remove · esc back to results · scroll 1/12`.
4. **Press `esc` to go back.** Each press of `esc` backs out one level: Detail → Results → Sidebar → quit the app. You can always press `esc` repeatedly to get back to the sidebar.

At any point, press `/` to jump to the search bar, `?` to open help, or `F1`–`F12` to switch tabs.

## The Search bar

The search bar lets you find packages by name or description across both pacman and the AUR.

### How to search

1. Press `/` from anywhere in the App Store. The search bar border turns accent blue and a cursor appears.
2. Type your query. While the search bar has focus, **every key you press goes into the search text** — letters like `j`, `k`, and `q` that are normally shortcuts are treated as regular characters so you can type package names freely.
3. Press `enter` to run the search. Results appear in the main pane, sorted with the best matches first.
4. Press `esc` to cancel without searching. Your previous query and results stay intact.
5. Press `backspace` to delete the last character.

### The AUR toggle

The `[aur: on]` / `[aur: off]` badge on the right side of the search bar controls whether the AUR is included in your searches. Press `A` (shift+a) from outside search mode to toggle it. When off, searches only cover the official pacman repos. When on, searches fan out to both pacman and the AUR and merge the results.

An empty search (pressing `enter` with nothing typed) returns you to whichever category is highlighted in the sidebar.

## The Categories sidebar

The sidebar has two kinds of entries:

### Navigation views (always available)

- **Featured** — A curated list of popular apps that also exist in your repos. This is a good starting point for exploring. The list is defined in a YAML configuration file that you can customize (see the Customization section below).
- **All Packages** — Every package pacman knows about. The count in parentheses reflects your current repo state (typically 14,000–16,000 packages depending on which repos are enabled).
- **Installed** — Everything currently on your system according to pacman, including AUR packages you've installed via a helper.
- **AUR** — The Arch User Repository. This only shows results when you run a search with the AUR toggle on, because the AUR has no "browse everything" endpoint.

### Content categories (show a count when populated)

- **Development** — Compilers, IDEs, version control tools, databases, Docker, language runtimes, and build systems.
- **Graphics** — Image editors (GIMP, Inkscape, Krita), 3D tools (Blender), photography apps, screenshot tools, and vector graphics.
- **Internet** — Web browsers (Firefox, Chromium), email clients, chat apps (Signal, Discord, Telegram), download managers, VPNs, and file sync tools.
- **Multimedia** — Media players (VLC, mpv), video editors (Kdenlive, OBS), audio workstations (Audacity), codecs, and streaming tools.
- **Office** — Document suites (LibreOffice), PDF viewers, note-taking apps (Obsidian), finance tools, and password managers.
- **System** — Terminal emulators (Ghostty, Alacritty, Kitty), shells, file managers, system monitors, boot loaders, firewalls, and command-line utilities.
- **Games** — Game launchers (Steam, Lutris), emulators (RetroArch), native games, and compatibility layers (Wine, Proton).
- **Other** — A catch-all for packages that don't fit elsewhere. Currently empty because uncategorized packages are left untagged rather than force-bucketed.

Categories are populated automatically from a curated YAML file plus the `Categories=` field in your installed packages' `.desktop` files. If a category shows a count of zero, it means no packages in your repos matched that category — this is normal for categories like Games if you don't have any game packages installed.

Disabled categories (shown dimmed and italic) are skipped when you press `j`/`k`, so the cursor only lands on entries you can actually browse.

## The Results list

When you select a category or run a search, the main pane fills with a list of packages. Each row shows:

- **Selection caret** (`▸`) — marks the currently highlighted row when focus is on the results.
- **Name** — displayed in accent color when the package is already installed.
- **Size** — the installed size in human-readable form (KiB, MiB, GiB). A dash (`—`) means the size is unknown, which is normal for AUR packages that haven't been built yet.
- **Origin badge** — shows where the package comes from:
  - `installed` (green) — already on your system.
  - `AUR` (gold) — from the Arch User Repository.
  - `core`, `extra`, `multilib` (dim) — the official pacman repository it lives in.
- **Description** — the short package description, truncated to fit the row.
- **Action hint** (on the selected row only) — shows `[i install]` in green for packages you don't have, or `[X remove]` in red for packages you do. This reminds you which key to press.

### Result sort order

Results are sorted to put the most relevant items first:

1. Exact name match — if you searched for `firefox`, the package literally named `firefox` appears first.
2. Installed packages — things you already have come before things you don't.
3. Pacman repos before AUR — official-repo packages are listed before AUR results.
4. Alphabetical by name within each tier.

### Position indicator

The status footer shows your position in the list, e.g. `3/142`, so you know where you are and how many results there are. If the search returned more than 200 results, the footer also shows `(truncated)` — refine your query to see the rest.

## The Detail panel

Press `enter` on a highlighted package to see its full details. The Results list is replaced by a scrollable readout that includes:

### Header

The package name in accent color, version in dim, the origin badge, and an action indicator (`[i install]` or `[X remove]`).

### Metadata fields

- **Repo** — which pacman repository the package lives in (or `aur`).
- **URL** — the upstream project website.
- **Licenses** — the software license(s).
- **Download** — the compressed download size. Only available for official-repo packages.
- **Installed** — the uncompressed on-disk size after installation. Only available for official-repo packages and packages you've already installed.
- **Updated** — how long ago the package was last built or modified (e.g. "3 days ago", "2 months ago").
- **Votes / Popularity** — AUR-only engagement metrics. Higher votes and popularity suggest a more widely-used and better-maintained package.
- **Maintainer / Packager** — who currently stewards the package.

### Dependency lists

Below the metadata, the detail panel shows:

- **Depends On** — packages this one requires to run. These are installed automatically.
- **Optional** — packages that add extra features but aren't required.
- **Make Deps** — packages needed only to build this package from source (AUR packages only).
- **Conflicts** — packages that can't coexist with this one. If you install this, pacman will ask to remove the conflicting package.

Empty sections are hidden so packages with sparse metadata don't waste screen space.

### Scrolling the detail panel

When the detail content is taller than the screen, use `j`/`k` (or arrow keys) to scroll up and down. The title bar shows a position indicator like `[3/12]` so you know where you are in the content. Press `esc` to close the detail panel and return to the Results list.

## Installing, removing, and upgrading packages

### Install a package (`i`)

1. Navigate to the package in the Results list or open its Detail panel.
2. Press `i`. A confirmation dialog appears: "Install <name>?"
3. Press `enter` to confirm, or `esc` to cancel.
4. For **official-repo packages**: dark invokes the privileged helper (`dark-helper`) via `pkexec`. Your system's polkit agent shows an authentication dialog — type your password and press enter. Pacman downloads and installs the package.
5. For **AUR packages**: dark shells out to your AUR helper (`paru` or `yay`), which clones the PKGBUILD, builds the package, and installs it. The AUR helper handles sudo internally.
6. After a successful install, the catalog refreshes automatically and the package's badge flips from its repo name to `installed`.

If the package is already installed, pressing `i` does nothing. The action hint on the row will show `[X remove]` instead.

### Remove a package (`X`)

1. Navigate to an installed package (its badge says `installed` in green).
2. Press `X` (shift+x). A confirmation dialog appears: "Remove <name>?"
3. Press `enter` to confirm, or `esc` to cancel.
4. Dark invokes the privileged helper via `pkexec`. Authenticate in the polkit dialog. Pacman removes the package.
5. The catalog refreshes and the package's badge reverts to its repo name.

If the package is not installed, pressing `X` does nothing.

### Run a full system upgrade (`U`)

1. Press `U` (shift+u) from anywhere in the App Store.
2. A confirmation dialog appears: "Run system upgrade (pacman -Syu)?"
3. Press `enter` to confirm. This is equivalent to running `sudo pacman -Syu` in a terminal.
4. The status line shows `working…` while the upgrade runs. This can take a while depending on how many packages need updating and your download speed.
5. When complete, the catalog refreshes with the new package versions.

### About the authentication dialog

Every install, remove, and upgrade requires root access. Dark uses `pkexec` (part of polkit) to request it. When you confirm an action, your desktop's polkit agent pops up an authentication dialog where you type your password. Dark **never** sees, stores, or handles your password — polkit manages the entire authentication flow.

The `dark-helper` binary that runs as root validates every input before touching pacman. Package names must contain only letters, numbers, `@`, `.`, `_`, `+`, and `-`. No more than 20 packages can be installed or removed in a single operation. These restrictions exist to prevent the helper from being misused.

### AUR packages — what's different

The **Arch User Repository (AUR)** is a community-maintained collection of package build scripts, not pre-compiled binaries. When you install an AUR package, your machine downloads the build script, fetches the upstream source code, compiles it locally, and then installs the result. This means:

- **Sizes are unknown until after you install.** The AUR doesn't know how big the compiled package will be because it depends on your compile settings, architecture, and optional features. Size shows as `—` in the results list.
- **You need an AUR helper.** Dark uses `paru` or `yay` (whichever is installed) to handle the AUR build process. If neither is installed, AUR packages show `[i install]` but pressing `i` will give you an error explaining that you need to install one.
- **Builds can take a long time.** Compiling software from source ranges from seconds (small tools) to an hour or more (large projects like browsers or game engines). The status line shows `working…` during the build.
- **AUR packages are not vetted by Arch.** Anyone can upload a PKGBUILD. Always review the votes, popularity, maintainer, and last-updated date in the Detail panel before installing.

## Customization

All curated data — the Featured list, category assignments, and XDG-to-sidebar mappings — lives in a YAML file that you can override.

### The data file: `categories.yaml`

The default is compiled into the dark binary, but you can create your own at:

    $XDG_CONFIG_HOME/dark/scripts/appstore/categories.yaml

(On most systems this is `~/.config/dark/scripts/appstore/categories.yaml`.)

The file has three sections:

```yaml
# Maps XDG desktop categories to dark sidebar groups
xdg_map:
  WebBrowser: internet
  AudioVideo: multimedia
  # ... add your own

# Maps specific package names to sidebar groups
packages:
  my-custom-tool: development
  some-game: games
  # ... add your own

# Curated list for the Featured sidebar view (order matters)
featured:
  - firefox
  - ghostty
  - my-favorite-app
  # ... add your own
```

Your override file **completely replaces** the default — dark does not merge them. Copy the full default from the source if you want to start from the built-in list and make changes.

### The logic file: `categories.lua`

The Lua script controls how the YAML data is loaded and processed. The default script simply loads the YAML and sets the globals:

```lua
local data = load_yaml("appstore/categories.yaml")
xdg_map  = data.xdg_map  or {}
packages = data.packages  or {}
featured = data.featured  or {}
```

Override it at `$XDG_CONFIG_HOME/dark/scripts/appstore/categories.lua` if you want to do something more advanced, like merge multiple YAML files or add conditional logic:

```lua
local data = load_yaml("appstore/categories.yaml")
xdg_map  = data.xdg_map  or {}
packages = data.packages  or {}
featured = data.featured  or {}

-- Add a custom entry
packages["my-tool"] = "development"
table.insert(featured, 1, "my-tool")
```

After changing either file, press `R` in the App Store to refresh and pick up your changes.

## Global key reference

### From outside search mode

| Key | What it does |
|-----|-------------|
| `j` / `↓` | Move selection down (sidebar, results, or scroll detail) |
| `k` / `↑` | Move selection up |
| `enter` | Activate: sidebar → results, results → detail |
| `esc` | Back out: detail → results → sidebar → quit |
| `/` | Open the search bar |
| `i` | Install the highlighted package |
| `X` | Remove the highlighted package (shift+x) |
| `U` | Run a full system upgrade (shift+u) |
| `A` | Toggle AUR inclusion in searches (shift+a) |
| `R` | Refresh the catalog (shift+r) |
| `?` | Open this help panel |
| `q` | Quit dark |
| `ctrl+c` | Quit from anywhere |
| `ctrl+r` | Rebuild dark in place |
| `f1`–`f12` | Switch tabs |

### From inside search mode

| Key | What it does |
|-----|-------------|
| Any letter/number | Types into the search query |
| `enter` | Run the search |
| `esc` | Cancel and keep existing results |
| `backspace` | Delete the last character |
| `ctrl+c` | Quit |
| `f1`–`f12` | Switch tabs (leaves search mode) |
| `?` | Open help |

## Status footer reference

The one-line footer at the bottom always shows what's happening:

| What you see | What it means |
|-------------|--------------|
| Key hints (e.g. `j/k nav · enter detail · ...`) | Normal idle state. Shows which keys are available for the current focus. |
| `working…` (accent color) | An operation is in progress: search, detail fetch, install, remove, or upgrade. |
| `3/142` | Your cursor position and total result count. |
| `scroll 5/28` | Your scroll position in the detail panel. |
| `(truncated)` | The search returned more than 200 results. Refine your query. |
| `AUR limited` (red) | The AUR backend is in rate-limit backoff. Wait for the timer or try again later. |
| A red error message | Something failed. The text comes from the daemon and describes the problem. |

## AUR rate limiting

The AUR's web API doesn't publish rate limits, but dark is a good citizen and imposes its own:

- At most **2 requests in flight** at any time.
- On HTTP 429 (Too Many Requests) or 503 (Service Unavailable), dark backs off exponentially: 10 seconds, then 20, then 40, up to a maximum of 5 minutes. If the server sends a `Retry-After` header, dark uses that instead.
- While in backoff, any search that needs AUR data returns immediately with cached results (if available) or an empty result plus a status warning.
- The backoff clears on the first successful request after the timer expires.

You'll see `AUR rate limited — retrying in Ns` in the status footer when this happens. You don't need to do anything — just wait or keep browsing official-repo packages.

## Troubleshooting

### The catalog is empty or shows 0 packages

This usually means `pacman` is not on your PATH or the sync databases haven't been initialized. Run `sudo pacman -Sy` in a terminal to sync the databases, then press `R` in the App Store to refresh.

### Categories all show zero counts

The catalog may have loaded from an old disk cache that predates the category system. Press `R` to force a fresh rebuild. If categories still show zero, check that your Lua/YAML override files are valid — a syntax error in either will cause categories to fail silently (the daemon logs a warning at info level).

### Install fails with "authentication cancelled"

You dismissed the polkit authentication dialog without typing your password. Press `i` again and complete the authentication this time.

### Install fails with "dark-helper binary not found"

The privileged helper is not installed. It should be at `/usr/local/bin/dark-helper` or `/usr/bin/dark-helper`, or you can set the `DARK_HELPER` environment variable to its path. If you built dark from source, the helper binary is in the same directory as the daemon.

### AUR install fails with "no AUR helper installed"

Dark needs `paru` or `yay` on your PATH to install AUR packages. Install one with `sudo pacman -S paru` (if it's in your repos) or build it manually from the AUR. Once installed, AUR installs will work automatically.

### A package I know exists doesn't show up in results

The catalog refreshes every 6 hours from `pacman -Sl`. If a package was recently added to the repos, press `R` to force a refresh. For AUR packages, make sure the AUR toggle is on (`[aur: on]` in the search bar) — AUR results only appear when you explicitly include them.

### The detail panel shows "—" for everything

This happens for packages where pacman has limited metadata. It's common for meta-packages, package groups, and very old packages. The information shown is exactly what `pacman -Si` reports — dark doesn't add or remove fields.

## Data sources

Everything on this page comes from `darkd`, which reads:

- **pacman** via `pacman -Sl` (repo enumeration), `pacman -Si` (per-package detail), and `pacman -Qq` (installed state).
- **expac** when installed, for fast batch enrichment of descriptions and sizes. Without expac, the browse list shows empty descriptions but detail still works. Install expac with `sudo pacman -S expac` for the best experience.
- **aurweb RPC v5** at `https://aur.archlinux.org/rpc/v5` for AUR search and detail. All calls use HTTPS.
- **categories.yaml** — the curated data file defining category mappings and the Featured list. Loaded by the Lua script at catalog-build time. Ships embedded in the binary; user overrides at `$XDG_CONFIG_HOME/dark/scripts/appstore/categories.yaml`.
- **categories.lua** — the Lua script that reads the YAML and exposes it to the backend. Ships embedded; user overrides at `$XDG_CONFIG_HOME/dark/scripts/appstore/categories.lua`.
- **.desktop files** at `/usr/share/applications/*.desktop` — the `Categories=` field is parsed for installed packages and mapped through the YAML's `xdg_map` table to assign categories automatically.
- **dark-helper** — the privileged companion binary invoked via `pkexec` for install, remove, and upgrade operations. Validates all inputs strictly before running pacman.
- **On-disk caches** under `$XDG_CACHE_HOME/dark/appstore` — 6 hours for the pacman catalog and 1 hour for AUR results. Categories are re-applied from the Lua/YAML pipeline on every load, so configuration changes take effect on the next refresh without clearing the cache.

## Backend notes

The appstore service follows the same pluggable-Backend pattern as wifi and bluetooth. The production backend composes a `pacmanBackend` (official repos) and an `aurBackend` (AUR) behind a single interface. If `pacman` is not on PATH, the service falls back to a `noopBackend` and the App Store shows an explanation.

Install and remove route through `dark-helper` + `pkexec` for official packages. AUR installs run through `paru` or `yay` as the current user. System upgrade runs `pacman -Syu` through the helper. The helper enforces strict input validation: package names must match `[a-zA-Z0-9@._+-]` and batch size is capped at 20.
