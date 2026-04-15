package scripting

// seedEventHooks records the event names that user scripts can
// register for. Every F1 service publishes a snapshot-typed event
// (e.g. on_wifi, on_bluetooth) that scripts can react to via
// `dark.on("on_wifi", function(snap) ... end)`. The snapshot is
// passed as a Lua table matching the service's JSON schema.
//
// The file lives separately from engine.go so the registry data
// stays browsable on its own without scrolling through VM setup.
// seedHostFunctions registers every `dark.*` host function in the
// Lua registry so the F5 reference browser can enumerate them with
// a signature and summary. The actual Lua bindings are attached in
// registerDarkModule; this just advertises them.
func (e *Engine) seedHostFunctions() {
	r := &e.registry
	r.RegisterFunction("dark.on",
		"(event, fn)",
		"Register a Lua function to run when the named event fires.")
	r.RegisterFunction("dark.log",
		"(message)",
		"Write a line to the dark daemon log at info level.")
	r.RegisterFunction("dark.cmd",
		"(subject, payload)",
		"Issue a NATS request against a dark.cmd.* subject. Returns (reply_table, nil) on success or (nil, error_string) on failure.")
	r.RegisterFunction("dark.notify",
		"(summary, body, urgency?)",
		"Send a desktop notification through the daemon notifier. Urgency accepts \"low\", \"normal\", or \"critical\" (default \"normal\").")
	r.RegisterFunction("dark.env",
		"(name)",
		"Return the value of an environment variable visible to darkd, or an empty string when unset.")
	r.RegisterFunction("dark.now",
		"()",
		"Return the current Unix timestamp (seconds since epoch) as a Lua number.")
	r.RegisterFunction("dark.hostname",
		"()",
		"Return the host's network name, or an empty string on error.")
	r.RegisterFunction("dark.read_file",
		"(path)",
		"Read a file and return its contents as a string. Returns (content, nil) on success or (nil, error_string) on failure.")
	r.RegisterFunction("dark.write_file",
		"(path, content)",
		"Write content to a file, creating parent directories as needed. Returns (true, nil) on success or (false, error_string) on failure.")
	r.RegisterFunction("dark.run",
		"(cmd, args?)",
		"Run a command synchronously with its arguments as an array table. Returns a result table {stdout, stderr, code} on success or (nil, error_string) when the command couldn't be started.")
	r.RegisterFunction("dark.spawn",
		"(cmd, args?)",
		"Start a command in the background and return its PID immediately. Returns (pid, nil) on success or (nil, error_string) on failure.")
	r.RegisterFunction("dark.home",
		"()",
		"Return the user's home directory, or an empty string when unavailable.")
	r.RegisterFunction("dark.scripts_dir",
		"()",
		"Return the user scripts directory (XDG_CONFIG_HOME/dark/scripts by default).")
	r.RegisterFunction("dark.config_dir",
		"()",
		"Return XDG_CONFIG_HOME, falling back to $HOME/.config.")
	r.RegisterFunction("dark.cache_dir",
		"()",
		"Return XDG_CACHE_HOME, falling back to $HOME/.cache.")
	r.RegisterFunction("dark.json_encode",
		"(value)",
		"Encode a Lua table or scalar as a JSON string. Returns (json_string, nil) on success or (nil, error_string) on failure.")
	r.RegisterFunction("dark.json_decode",
		"(string)",
		"Decode a JSON string into a Lua table or scalar. Returns (value, nil) on success or (nil, error_string) on failure.")
}

func (e *Engine) seedEventHooks() {
	// Lifecycle / TUI client events. Published by the dark client
	// on the `dark.client.<event>` subject; darkd's
	// wireScriptClientEvents handler forwards each publish into
	// DispatchEvent so scripts receive them without needing a
	// second connection of their own.
	e.registry.RegisterEvent("on_app_start",
		"()",
		"Fires when the dark TUI finishes loading its initial snapshots and is about to render.")
	e.registry.RegisterEvent("on_app_exit",
		"()",
		"Fires when the dark TUI quits (best effort — may not fire on hard kill).")
	e.registry.RegisterEvent("on_bus_connected",
		"()",
		"Fires when the client's NATS connection reconnects after a drop.")
	e.registry.RegisterEvent("on_bus_disconnected",
		"()",
		"Fires when the client's NATS connection drops.")

	// Tab switch events — one per function key. A generic
	// on_tab_change also fires alongside with the tab name.
	e.registry.RegisterEvent("on_f1", "()", "Fires when the user presses F1 to switch to the Settings tab.")
	e.registry.RegisterEvent("on_f2", "()", "Fires when the user presses F2 to switch to the App Store tab.")
	e.registry.RegisterEvent("on_f3", "()", "Fires when the user presses F3 to switch to the System tab.")
	e.registry.RegisterEvent("on_f4", "()", "Fires when the user presses F4 to switch to the Dark tab.")
	e.registry.RegisterEvent("on_f5", "()", "Fires when the user presses F5 to switch to the Scripting tab.")
	e.registry.RegisterEvent("on_tab_change",
		"(info)",
		"Fires on every tab switch with `info.tab` set to the destination event name (on_f1..on_f5).")

	// In-tab section change events.
	e.registry.RegisterEvent("on_f1_section",
		"(info)",
		"Fires when the highlighted F1 Settings section changes. `info.name` is the section ID (wifi, bluetooth, display, sound, network, power, input, notifications, datetime, privacy, users, appearance, workspaces, about).")
	e.registry.RegisterEvent("on_f2_category",
		"(info)",
		"Fires when the highlighted F2 App Store category changes. `info.id` is the category ID.")
	e.registry.RegisterEvent("on_f3_section",
		"(info)",
		"Fires when the highlighted F3 System section changes. `info.name` is the section ID (limine, keybindings, links).")

	// Scripting lifecycle events.
	e.registry.RegisterEvent("on_script_loaded",
		"(path)",
		"Fires when a Lua script file is loaded into the engine.")
	e.registry.RegisterEvent("on_package_installed",
		"(name)",
		"Fires when the App Store finishes installing a package.")
	e.registry.RegisterEvent("on_package_removed",
		"(name)",
		"Fires when the App Store finishes removing a package.")

	// F1 Settings snapshot events. Each one fires on every daemon
	// publish (periodic + on-change) with the full snapshot as a
	// Lua table mirroring the service's JSON schema.
	e.registry.RegisterEvent("on_sysinfo",
		"(snapshot)",
		"Fires on system info snapshot publishes (hostname, uptime, load, memory, etc.).")
	e.registry.RegisterEvent("on_wifi",
		"(snapshot)",
		"Fires on Wi-Fi adapter snapshot publishes (adapters, networks, known networks).")
	e.registry.RegisterEvent("on_bluetooth",
		"(snapshot)",
		"Fires on Bluetooth adapter snapshot publishes (adapters and their devices).")
	e.registry.RegisterEvent("on_audio",
		"(snapshot)",
		"Fires on audio device snapshot publishes (sinks, sources, cards, streams).")
	e.registry.RegisterEvent("on_network",
		"(snapshot)",
		"Fires on network snapshot publishes (interfaces, routes, DNS).")
	e.registry.RegisterEvent("on_display",
		"(snapshot)",
		"Fires on display monitor snapshot publishes (resolutions, scales, layout).")
	e.registry.RegisterEvent("on_datetime",
		"(snapshot)",
		"Fires on date/time snapshot publishes (timezone, NTP, format).")
	e.registry.RegisterEvent("on_notify",
		"(snapshot)",
		"Fires on notification config snapshot publishes.")
	e.registry.RegisterEvent("on_input",
		"(snapshot)",
		"Fires on input device snapshot publishes (keyboards, mice, touchpads).")
	e.registry.RegisterEvent("on_power",
		"(snapshot)",
		"Fires on power snapshot publishes (profiles, governors, idle behavior).")
	e.registry.RegisterEvent("on_privacy",
		"(snapshot)",
		"Fires on privacy snapshot publishes (idle, DNS, firewall, SSH, location).")
	e.registry.RegisterEvent("on_users",
		"(snapshot)",
		"Fires on users snapshot publishes (accounts, groups, shells).")
	e.registry.RegisterEvent("on_appearance",
		"(snapshot)",
		"Fires on appearance snapshot publishes (gaps, borders, blur, theme, font).")
	e.registry.RegisterEvent("on_workspaces",
		"(snapshot)",
		"Fires on workspaces snapshot publishes (layout, options, monitor map).")
}
