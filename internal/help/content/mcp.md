# MCP

`dark mcp` turns darkd into a Model Context Protocol server so an LLM host — Claude Desktop, Cursor, anything that speaks MCP — can drive every dark action and read every system snapshot without writing a single line of Lua.

The F5 `MCP` tab enumerates the tools and resources this server exposes, with the same per-entry doc layout the `Lua` and `API` tabs use. The mental model you already have for those tabs transfers directly.

## What MCP gives you that Lua doesn't

Lua is great when you want dark to react to itself: battery drops, Wi-Fi changes, package installed. MCP is the opposite direction — you want an outside intelligence to drive dark on your behalf. An LLM that can ask "what's the current volume" and then call `audio_sink_volume` to change it doesn't need a hand-written script for each case; it picks the right tool from the catalog.

Typical reasons to point an LLM at `dark mcp`:

- **Natural-language system control.** "Turn on bluetooth and pair my headphones" → the LLM picks `bluetooth_power`, `bluetooth_discover_on`, `bluetooth_pair` in sequence.
- **Troubleshooting and diagnostics.** "Why is my Wi-Fi slow?" → the LLM reads `dark://snapshots/wifi` and `dark://snapshots/network` and explains what it sees.
- **Bulk configuration.** "Set up my machine for presenting — disable notifications, max brightness, disable screensaver" → three tool calls the LLM can plan on its own.

If you prefer scripted automation with no LLM in the loop, stick with F5 Lua — it's faster, cheaper, and deterministic. MCP earns its place when the instructions are fuzzy or the plan is context-dependent.

## The mental model: tools and resources

MCP has two primitives `dark mcp` uses:

- **Tools** are the action surface. Every `dark.cmd.<domain>.<verb>` bus subject becomes an MCP tool named `<domain>_<verb>` (dots aren't valid in MCP tool names). Calling `wifi_scan` from an MCP host is identical to calling `dark.actions.wifi.scan` from Lua — the same JSON payload hits the same handler. The full parameter schema is advertised to the LLM so it can validate arguments before sending a request.
- **Resources** are the read-only state surface. Snapshot subjects like `dark.cmd.wifi.adapters` get a second identity as a resource at `dark://snapshots/wifi`. An MCP host can call `resources/read` with that URI instead of the equivalent tool — same JSON reply, but it lets the host keep a conceptual separation between "fetching state" and "taking action."

Open the F5 `MCP` tab for the canonical list. Tools are grouped by domain (`AUDIO`, `WIFI`, `BLUETOOTH`, …) exactly like the API tab. Resources are flat — one per F1 snapshot.

## Setup — Claude Desktop

Add dark to your Claude Desktop config (`~/.config/Claude/claude_desktop_config.json` on Linux):

```json
{
  "mcpServers": {
    "dark": {
      "command": "dark",
      "args": ["mcp"]
    }
  }
}
```

Restart Claude Desktop. The `dark` server should appear in the MCP menu and you can start issuing commands in the chat window. `dark mcp` connects to a running darkd daemon the same way the TUI does, so make sure `systemctl --user status darkd` looks healthy first.

Other MCP hosts (Cursor, Zed, custom clients) take a similar config — `command: dark`, `args: ["mcp"]` is all they need.

## Patterns

### 1. Ask the LLM to pick tools by description

The tool descriptions come straight from the bus catalog — the same one-liners you see in the F5 API tab. They're short and purpose-specific, which is what MCP hosts rely on when choosing between tools. You don't have to rewrite them; just keep `internal/bus/catalog.go` up to date when you add new commands and the LLM will pick the right tool without prompting hints.

### 2. Resources for "what's currently going on"

When the user asks "is my Wi-Fi on?" the right move for the LLM is `resources/read dark://snapshots/wifi`, not `tools/call wifi_adapters`. Functionally identical, but resources/read is semantically "fetch state" — most hosts cache the response and won't re-fetch unless the user asks for fresh data. Tools carry the opposite meaning ("perform an action") and are typically not cached.

### 3. Chain tools for multi-step plans

MCP hosts are good at chaining. If you ask for "connect to the strongest open network," the LLM will:

1. `tools/call wifi_scan` — kick off a scan.
2. `resources/read dark://snapshots/wifi` — fetch the results.
3. Pick the best SSID from the scan list.
4. `tools/call wifi_connect_hidden` or `wifi_connect` with the chosen SSID.

You write nothing. The catalog is enough vocabulary for the LLM to compose the plan.

### 4. Run alongside Lua and the TUI

`dark mcp` is a separate process that connects to the same darkd daemon as the TUI. All three — TUI, Lua engine inside darkd, and MCP stdio bridge — issue NATS requests against the same subjects. There's no shared state and no locking surprises; darkd serializes command handlers the same way it does for the TUI.

You can have the TUI open, a Lua script handling `on_wifi`, and an LLM driving bluetooth — all at once. They see each other's changes through the snapshot publish loop the same way any two TUI windows would.

## Gotchas and best practices

- **`dark mcp` doesn't start the daemon.** It's a one-shot stdio process. If darkd isn't running, `dark mcp` fails immediately with a "daemon not running" error and the MCP host surfaces it as a startup failure. Start darkd first (the systemd unit should be enabled).
- **LLMs will call destructive tools if asked.** `users_delete`, `limine_delete`, `firmware_apply`, `update_run` all live in the same tool catalog. There's no allowlist today — if you don't trust your host's tool-approval flow, don't wire `dark mcp` into it. Claude Desktop and most MCP hosts prompt for approval on every tool call by default; keep that on.
- **Tools block while darkd is working.** Long-running commands (firmware updates, package installs) hold the MCP tool call open until they finish. If the host's timeout is shorter than the command, it'll see a cancellation. There's no async/job pattern yet.
- **No event subscription yet.** MCP has a `resources/subscribe` primitive for "ping me when this changes," and darkd could plug its snapshot publishers into it — but v1 doesn't. Today an LLM only sees state when it explicitly reads a resource. If you want reactive behavior, write it in Lua with `dark.on(...)`.
- **The catalog is compile-time.** `dark mcp` builds its tool and resource lists from `bus.APICommandCatalog()` at startup — the same source the F5 MCP tab uses. A daemon upgrade won't magically give the MCP bridge new tools; the `dark` binary has to match.

## Key reference

| Key            | Action                              |
| -------------- | ----------------------------------- |
| `j` `k` `↑` `↓`| Move between tools / resources      |
| `enter`        | Focus inner sub-nav from outer      |
| `esc`          | Back out to outer sidebar           |
| `?`            | This help                           |

## Under the hood

- Transport: JSON-RPC 2.0 over stdio via `github.com/mark3labs/mcp-go`.
- Tool handlers: `internal/mcp/server.go` — one generic shim per subject, reads arguments, marshals JSON, issues a NATS request, returns the reply as `text/plain` content.
- Catalog derivation: `internal/mcp/catalog.go` — pure function over `bus.APICommandCatalog()`; same data the Lua `dark.actions.*` tree is built from.
- Daemon connection: reuses `bus.ConnectClient` — identical to what the TUI and `dark script call` use, so it shows up in the daemon log with `name=dark-mcp`.

If you want to add a new tool, don't touch the MCP package. Add the bus subject, write a handler in `cmd/darkd/`, add its entry to `internal/bus/catalog.go` and `internal/bus/schemas.go` — both F5 Lua and F5 MCP pick it up automatically.
