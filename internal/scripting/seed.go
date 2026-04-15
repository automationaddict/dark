package scripting

// seedEventHooks records the event names that user scripts can
// register for. Every F1 service publishes a snapshot-typed event
// (e.g. on_wifi, on_bluetooth) that scripts can react to via
// `dark.on("on_wifi", function(snap) ... end)`. The snapshot is
// passed as a Lua table matching the service's JSON schema.
//
// The file lives separately from engine.go so the registry data
// stays browsable on its own without scrolling through VM setup.
func (e *Engine) seedEventHooks() {
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
