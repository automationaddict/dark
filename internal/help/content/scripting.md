# Scripting

Write Lua scripts that react to system events and drive every command dark knows how to run. The F5 Scripting tab is a workbench: a file manager for your script files, an editor hand-off to `$EDITOR`, and a searchable reference browser for every Lua function, event, and bus action you can call.

Press `?` at any time to open this help. Press `esc` to close it.

## What F5 is for

F5 is a power-user surface. You don't need it to use dark — everything you can script is already reachable by keyboard in F1–F4. Scripting earns its place when you want behavior that the UI can't express: reacting to state changes, chaining commands, binding actions to keys outside dark, persisting data, or making dark talk to external tools.

Typical reasons to write a script:

- **Keybindings that call dark actions.** `dark script call volume_up` in a Hyprland bind is the easiest way to wire media keys to `dark.actions.audio.sink_volume`.
- **Reactive automation.** Auto-dim the screen when battery drops below 20%, re-enable the firewall when you leave your home network, silence notifications while a specific app is running.
- **Glue between dark and external tools.** Call `notify-send`, pipe output to another program, post to a webhook, run a shell command when a package installs.
- **Persistent per-user settings or counters** that survive across daemon restarts — e.g. a "times dark has booted today" counter, or a custom config file that changes dark behavior.

If you just want to change a setting once, use F1. If you want "every time X happens, do Y," write a script.

## What you see when you land here

Press `F5`. The screen is three columns:

1. **Outer sidebar** — four top-level groups: `Scripts`, `MCP`, `Lua`, `API`.
2. **Inner sub-nav** — the entries inside the currently-focused group. Your script files under Scripts; every host function, event, and auto-generated action under Lua; every NATS subject under API.
3. **Content pane** — glamour-rendered markdown docs for the highlighted inner entry.

A dim menu bar pinned to the bottom of the content pane shows the keys that work from your current position.

## Navigating — the focus model

F5 uses the same two-level focus model as F1 Settings. One region owns the keyboard at a time:

1. **You start on the outer sidebar.** `j`/`k` (or arrows) move between `Scripts`, `MCP`, `Lua`, `API`. The inner sub-nav renders as context but arrows don't drive it yet.
2. **Press `enter` to focus the inner sub-nav.** The outer sidebar dims, the inner sub-nav lights up. Arrows now walk the entries inside the selected group, and the doc pane updates live.
3. **Press `esc` to back out.** Focus returns to the outer sidebar so you can switch groups. Repeated `esc` eventually quits dark.

Per-group selection is remembered — switching `Scripts → API → Scripts` returns you to the same file you were on.

## Managing scripts

Scripts live under `~/.config/dark/scripts/` (or `$XDG_CONFIG_HOME/dark/scripts/`). darkd loads every `.lua` file in that directory at startup and again on every save, so hooks declared at script top level register automatically — no daemon restart needed.

**Create a script:** focus the Scripts group, press `enter` to move into the inner sub-nav, highlight `+ New script`, press `enter`, type the filename (`.lua` is appended if you forget), press `enter` again, and `$EDITOR` opens on the new empty file. Save and exit your editor to commit.

**Edit a script:** highlight a file row and press `enter`. `$EDITOR` opens on the real on-disk path, so your plugins, undo history, and filetype detection all work. On exit, darkd reloads every user script automatically.

**Delete a script:** press `d`, confirm in the dialog. darkd reloads so any hooks the file registered stop firing.

**Preview:** while a file row is highlighted the content pane shows the full file with syntax highlighting, plus the file's path/size/modtime at the top. `PgUp`/`PgDn` scroll the preview when it's longer than the viewport. The menu bar lists the keys.

## The mental model: functions, events, actions

Everything user scripts can do is built from three categories of primitive. The **F5 → Lua** sub-nav is the complete, always-up-to-date reference for each one, with signatures, parameter schemas, and copy-pasteable examples. This help doc won't duplicate that — browse Lua in F5 when you want specifics. What follows is how to think about them.

### Functions — things you call

`dark.*` is a table of host functions the engine injects as globals. Some are building blocks (`dark.on`, `dark.log`), some are bridges to system services (`dark.notify`, `dark.cmd`), and some are plain stdlib conveniences (`dark.env`, `dark.now`, `dark.read_file`, `dark.json_decode`, `dark.run`, `dark.spawn`). You call them from top-level script code, from inside event hooks, or from helpers your own script defines.

Rough categories to know they exist:

- **Core** — register hooks, log, dispatch bus commands.
- **Notifications** — pop a desktop toast.
- **Filesystem** — read/write files, XDG path lookups.
- **Processes** — run a command synchronously and capture output, or fire-and-forget.
- **Environment** — hostname, env vars, current time.
- **JSON** — encode/decode for talking to external APIs or persisting structured data.

Open F5 → Lua → FUNCTIONS to see the full list with signatures.

### Events — things that happen

`dark.on("event_name", fn)` registers a callback. There are four families:

- **Lifecycle** — `on_app_start`, `on_app_exit`, `on_bus_connected`, `on_bus_disconnected`. Fire once per state transition, no payload.
- **UI navigation** — `on_f1`..`on_f5` when the user switches tabs, plus `on_f1_section` / `on_f2_category` / `on_f3_section` when the highlighted entry inside a tab changes. Good for "do X when the user opens the App Store" or "log every section the user visits."
- **Scripting lifecycle** — `on_script_loaded`, `on_package_installed`, `on_package_removed`.
- **Service snapshots** — `on_wifi`, `on_bluetooth`, `on_audio`, `on_display`, `on_network`, `on_power`, `on_input`, `on_notify`, `on_datetime`, `on_privacy`, `on_users`, `on_appearance`, `on_workspaces`, `on_sysinfo`. Fire on every daemon publish (both periodic ticks and on-change updates) with the full snapshot as a Lua table.

Open F5 → Lua → EVENTS to see the full list with payload schemas.

**Important:** snapshot events fire frequently. Don't do expensive work inside a hook without debouncing it yourself — `dark.now()` is the easy way to compare timestamps and skip work that ran recently.

### Actions — things you can ask dark to do

Every `dark.cmd.*` NATS subject in the F1-F4 UI has a matching Lua wrapper at `dark.actions.<domain>.<verb>`. There are ~150 of them covering wifi, bluetooth, audio, display, network, power, input, notifications, datetime, privacy, users, appearance, workspaces, keybindings, updates, firmware, limine, screensaver, topbar, workspaces, appstore, dark self-update, and the scripting surface itself.

Every action returns `(reply_table, nil)` on success or `(nil, error_string)` on failure so you can destructure with `local reply, err = dark.actions.xyz({...})`. A non-zero handler error (e.g. "missing adapter name") comes through as the second value; a successful command returns the refreshed service snapshot as the first value, same as the bus reply.

Open F5 → Lua → ACTIONS to browse by domain. Each entry shows the payload schema (required vs optional fields, types, descriptions) and a ready-to-use Lua call example.

## Patterns

### Pattern 1: relative bumps over a cached snapshot

Services publish snapshots; you cache the bits you care about in a module-level variable; helper functions read from the cache and call an action with a relative delta. This is how `volume.lua`, `brightness.lua`, and the seeded examples work.

```lua
local default_sink = nil

dark.on("on_audio", function(snap)
  for _, sink in ipairs(snap.sinks or {}) do
    if sink.default then default_sink = sink; return end
  end
end)

function volume_up(step)
  if not default_sink then return end
  dark.actions.audio.sink_volume({
    index  = default_sink.index,
    volume = (default_sink.volume or 0) + (step or 5),
  })
end
```

Wire `volume_up` to a keybinding and you have media-key control in a dozen lines of Lua. The cache is updated on every audio publish, so the script always operates on fresh state without racing the daemon.

### Pattern 2: stateful toggles

When a setting is "on or off" rather than "increase or decrease," read the current value from a snapshot and flip it with `not`. `dnd_toggle.lua` does this for notifications:

```lua
local dnd_enabled = false

dark.on("on_notify", function(snap)
  if snap.dnd ~= nil then dnd_enabled = snap.dnd end
end)

function dnd_toggle()
  local new_state = not dnd_enabled
  dark.actions.notify.dnd({ enabled = new_state })
  return new_state
end
```

Returning the new state means `dark script call dnd_toggle` prints `true` or `false` to stdout — which is handy if you want to pipe it to another command in a shell script.

### Pattern 3: reacting to UI navigation

Sometimes the trigger you want isn't a service snapshot — it's "the user just looked at their display settings." The `on_f1_section` family exists for exactly this:

```lua
dark.on("on_f1_section", function(info)
  if info.name == "display" then
    dark.log("user opened display settings, refreshing monitor list")
  end
end)
```

Or react to tab switches with `on_f1`..`on_f5`, or to the generic `on_tab_change` if you don't care which one. The lifecycle events `on_app_start` / `on_app_exit` are the bookends for the whole session.

### Pattern 4: persisting data across daemon restarts

The `dark.cache_dir()` and `dark.scripts_dir()` helpers give you a stable place to write files. Combine with `dark.read_file` and `dark.write_file` for simple counter / timestamp / last-seen persistence:

```lua
local stamp_path = dark.cache_dir() .. "/dark/last_seen.json"

dark.on("on_app_start", function()
  local content, _ = dark.read_file(stamp_path)
  local state = {}
  if content then state = dark.json_decode(content) or {} end

  local now = dark.now()
  if state.last_seen and (now - state.last_seen) > 86400 then
    dark.notify("Welcome back", "It's been over a day since you ran dark.")
  end
  state.last_seen = now

  local encoded, _ = dark.json_encode(state)
  dark.write_file(stamp_path, encoded)
end)
```

### Pattern 5: shelling out to external tools

Anything Lua can't do directly, you can do through `dark.spawn` (fire-and-forget) or `dark.run` (sync with captured output).

```lua
-- Fire an external notification daemon command
dark.on("on_package_installed", function(name)
  dark.spawn("notify-send", { "Package installed", name })
end)

-- Capture the output of a command and decide based on it
local result, err = dark.run("pactl", { "list", "sinks", "short" })
if not err and result.code == 0 then
  for line in result.stdout:gmatch("[^\n]+") do
    dark.log("sink: " .. line)
  end
end
```

`dark.run` returns `{stdout, stderr, code}` on success. A non-zero exit is a successful return with a non-zero `code` — the caller decides whether that's an error. `dark.spawn` returns the PID immediately and the engine reaps the child in the background.

### Pattern 6: debouncing a noisy event

Snapshot events fire on every publish, so "when X happens, do Y" needs to guard against Y running dozens of times for a single user action. The cheap way is to record the last run timestamp and bail if it was too recent:

```lua
local last_run = 0

dark.on("on_wifi", function(snap)
  local now = dark.now()
  if now - last_run < 5 then return end
  last_run = now

  -- do the thing no more than once every 5 seconds
end)
```

For more complex cases, debounce on the specific field that changes — e.g. only act when `snap.adapters[1].connected_ssid` actually differs from the last value you saw.

## Running scripts from the CLI

The `dark script call <fn> [args...]` subcommand invokes any Lua global function in the running engine without launching the TUI. This is the bridge between your scripts and the outside world — Hyprland keybindings, cron jobs, shell aliases, systemd services, or other programs.

```bash
dark script call volume_up
dark script call volume_set 40
dark script call dnd_toggle
```

Positional arguments are parsed as JSON literals when possible, so `40` is a number, `true` is a bool, and unquoted words fall back to strings. Return values print to stdout as JSON — a helper that returns a number or bool gives you something you can pipe into another command.

**Wire to Hyprland:**

```
bind = , XF86AudioRaiseVolume, exec, dark script call volume_up
bind = , XF86AudioLowerVolume, exec, dark script call volume_down
bind = , XF86AudioMute,        exec, dark script call volume_set 0
bind = , XF86MonBrightnessUp,  exec, dark script call brightness_up
bind = SUPER, N,               exec, dark script call dnd_toggle
```

## Example scripts that ship

On first run, darkd seeds four example scripts into `~/.config/dark/scripts/`. None are essential — delete or edit any of them:

- **`volume.lua`** — canonical action-wrapper pattern built around a cached default sink.
- **`brightness.lua`** — same pattern applied to display backlight.
- **`dnd_toggle.lua`** — stateful toggle; returns the new state to stdout.
- **`wifi_watch.lua`** — pure event observer with no CLI surface; logs SSID changes.

The files are seeded only when they don't already exist, so editing or deleting any of them is permanent across daemon restarts.

## Gotchas and best practices

- **All engine calls serialize on one mutex.** One long-running hook blocks every other hook and every `dark.cmd` / `dark.actions.*` call until it finishes. Don't sleep or loop inside a hook; use `dark.now()` to debounce instead.
- **`dark.cmd` runs synchronously from the engine goroutine.** It blocks the hook that called it until the daemon replies. For fast service commands this is fine (milliseconds); for slow ones (e.g. package installs) consider dispatching from outside a hook or accepting the latency.
- **Errors don't crash the engine.** A Lua runtime error inside a hook is logged with the event name and the remaining hooks still run. Your script stays loaded and the next event triggers it normally.
- **Top-level code runs once at load.** Defining globals and calling `dark.on` at the top level is the correct pattern. If you need something to happen "on every startup," put it in an `on_app_start` hook so it also fires when the TUI restarts against an already-running daemon.
- **Naming collisions are real.** Every user script shares a single global namespace. If two scripts both define `function volume_up`, the last one loaded wins. Use unique names or wrap helpers in a module-like local table.

## Global key reference

### Outer sidebar focused

| Key | What it does |
|-----|-------------|
| `↑` / `↓` / `k` / `j` | Walk the four groups (Scripts, MCP, Lua, API) |
| `enter` | Focus the inner sub-nav for the selected group |
| `esc` | Back out; repeated presses eventually quit |
| `?` | Open / close this help panel |
| `F1`–`F12` | Switch tabs |
| `q` / `ctrl+c` | Quit dark |

### Inner sub-nav focused — Scripts group

| Key | What it does |
|-----|-------------|
| `↑` / `↓` / `k` / `j` | Move between `+ New script` and script files |
| `pgup` / `pgdn` | Scroll the preview content |
| `enter` | Create a new script (on `+ New script`) or open the highlighted file in `$EDITOR` |
| `d` | Delete the highlighted script (with confirmation) |
| `esc` | Return focus to the outer sidebar |

### Inner sub-nav focused — MCP / Lua / API groups

| Key | What it does |
|-----|-------------|
| `↑` / `↓` / `k` / `j` | Walk entries (functions, events, actions, subjects) |
| `esc` | Return focus to the outer sidebar |

## Data sources

Everything on this page is served by `darkd`:

- **User scripts** — files under `$XDG_CONFIG_HOME/dark/scripts/*.lua`. Walked on every `scripting.list` and `scripting.reload` bus request. Seeded on first daemon start from examples compiled into the binary via `//go:embed`.
- **Lua registry** — populated at engine startup. Host functions (`dark.on`, `dark.log`, `dark.cmd`, `dark.notify`, `dark.read_file`, `dark.run`, ...) are registered by the scripting package; every `dark.cmd.*` subject in the bus catalog is converted into a `dark.actions.<domain>.<verb>` wrapper and added to the registry with its schema.
- **Bus catalog** — `internal/bus/catalog.go` enumerates every subject, `internal/bus/schemas.go` attaches per-command field schemas. Single source of truth feeding both the F5 API browser and the `dark.actions.*` Lua wrappers.
- **Lua runtime** — `github.com/yuin/gopher-lua` embedded in the daemon. One persistent VM per darkd process; all calls serialize on a mutex for safety, and errors inside hooks are caught with `pcall` so a bad script can't crash a service.

## Backend notes

The scripting surface is split across packages:

- **`internal/scripting`** — the engine. Owns the VM, the registry, the event dispatcher, the dark module (`dark.on`, `dark.log`, `dark.cmd`, `dark.actions.*`, stdlib helpers), and the script file list/read/save/delete helpers. Stays free of the bus import — talks to NATS through injected `RequesterFunc` / `NotifyFunc` callbacks installed by darkd at startup.
- **`internal/bus/catalog.go`** and **`internal/bus/schemas.go`** — enumerate the `dark.cmd.*` surface and attach per-command field schemas. The catalog is the single source of truth for the F5 API browser and for registering Lua action wrappers.
- **`cmd/darkd/scripting.go`** — the daemon-side bus handlers: `scripting.list`, `scripting.registry`, `scripting.api_catalog`, `scripting.read`, `scripting.save`, `scripting.delete`, `scripting.call`, `scripting.reload`.
- **`cmd/darkd/script_events.go`** — subscribes to every F1 snapshot subject and to `dark.client.>` (UI events published by the TUI) and forwards both into `Engine.DispatchEvent` so `dark.on("on_wifi", ...)`, `dark.on("on_f1", ...)`, and siblings all fire. Also registers every bus command as a Lua action on daemon startup.
- **`cmd/dark/script.go`** — the `dark script call` subcommand. Skips the TUI process lock so it can run alongside a long-lived session.
- **`internal/tui/editor_external.go`** — the `$EDITOR` hand-off. Uses `tea.ExecProcess` to suspend bubbletea, spawn the editor inheriting stdin/stdout/stderr, and resume on exit. The same helper backs F1 Top Bar config/style editing and screensaver content editing.
