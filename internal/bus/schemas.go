package bus

// CommandField describes one payload parameter accepted by a
// `dark.cmd.*` handler. Type uses a friendly label ("string", "int",
// "bool", "float", "[]string", etc.) rather than reflection-accurate
// Go types so the F5 Scripting doc stays readable. Required is true
// when the handler returns an error on the missing/empty value.
type CommandField struct {
	Name     string
	Type     string
	Required bool
	Desc     string
}

// CommandSchema returns the documented parameters for a given bus
// command subject. An unknown subject returns a nil slice; snapshot
// subjects that take no payload return an empty slice.
func CommandSchema(subject string) []CommandField {
	return commandSchemas[subject]
}

// commandSchemas maps every `dark.cmd.*` subject to its parameter
// list. The list is hand-curated from the darkd handler bodies in
// cmd/darkd/*.go. When you add a new command, append its schema
// here so scripts get discoverable parameter docs.
var commandSchemas = map[string][]CommandField{
	// System info
	SubjectSystemInfoCmd: {},

	// Wi-Fi
	SubjectWifiAdaptersCmd: {},
	SubjectWifiScanCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Wireless interface name (e.g. wlan0)."},
	},
	SubjectWifiConnectCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Wireless interface name."},
		{Name: "ssid", Type: "string", Required: true, Desc: "Target network SSID."},
		{Name: "passphrase", Type: "string", Required: false, Desc: "WPA passphrase; empty for open networks."},
	},
	SubjectWifiDisconnectCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Wireless interface name."},
	},
	SubjectWifiForgetCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Wireless interface name."},
		{Name: "ssid", Type: "string", Required: true, Desc: "SSID to forget."},
	},
	SubjectWifiPowerCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Wireless interface name."},
		{Name: "powered", Type: "bool", Required: true, Desc: "True to enable the radio, false to disable."},
	},
	SubjectWifiAutoconnectCmd: {
		{Name: "ssid", Type: "string", Required: true, Desc: "Known network SSID."},
		{Name: "powered", Type: "bool", Required: true, Desc: "True to enable autoconnect, false to disable."},
	},
	SubjectWifiConnectHiddenCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Wireless interface name."},
		{Name: "ssid", Type: "string", Required: true, Desc: "Hidden network SSID."},
		{Name: "passphrase", Type: "string", Required: false, Desc: "WPA passphrase if the network is encrypted."},
	},
	SubjectWifiAPStartCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Wireless interface name."},
		{Name: "ssid", Type: "string", Required: true, Desc: "Access point SSID."},
		{Name: "passphrase", Type: "string", Required: false, Desc: "WPA passphrase for the AP."},
	},
	SubjectWifiAPStopCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Wireless interface name."},
	},

	// Bluetooth
	SubjectBluetoothAdaptersCmd: {},
	SubjectBluetoothPowerCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
		{Name: "on", Type: "bool", Required: true, Desc: "True to power on, false to power off."},
	},
	SubjectBluetoothDiscoverOnCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
	},
	SubjectBluetoothDiscoverOffCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
	},
	SubjectBluetoothConnectCmd: {
		{Name: "device", Type: "string", Required: true, Desc: "Bluetooth device address."},
	},
	SubjectBluetoothDisconnectCmd: {
		{Name: "device", Type: "string", Required: true, Desc: "Bluetooth device address."},
	},
	SubjectBluetoothPairCmd: {
		{Name: "device", Type: "string", Required: true, Desc: "Bluetooth device address."},
		{Name: "pin", Type: "string", Required: false, Desc: "PIN if required by the device."},
	},
	SubjectBluetoothRemoveCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
		{Name: "device", Type: "string", Required: true, Desc: "Device address to forget."},
	},
	SubjectBluetoothTrustCmd: {
		{Name: "device", Type: "string", Required: true, Desc: "Bluetooth device address."},
		{Name: "on", Type: "bool", Required: true, Desc: "True to trust, false to untrust."},
	},
	SubjectBluetoothDiscoverableCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
		{Name: "on", Type: "bool", Required: true, Desc: "True to make the adapter discoverable."},
	},
	SubjectBluetoothAliasCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
		{Name: "alias", Type: "string", Required: false, Desc: "Friendly name to advertise; empty to reset."},
	},
	SubjectBluetoothPairableCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
		{Name: "on", Type: "bool", Required: true, Desc: "True to allow incoming pairings."},
	},
	SubjectBluetoothBlockCmd: {
		{Name: "device", Type: "string", Required: true, Desc: "Bluetooth device address."},
		{Name: "on", Type: "bool", Required: true, Desc: "True to block, false to unblock."},
	},
	SubjectBluetoothCancelPairCmd: {
		{Name: "device", Type: "string", Required: true, Desc: "Bluetooth device address."},
	},
	SubjectBluetoothDiscoverableTimeoutCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
		{Name: "seconds", Type: "int", Required: false, Desc: "Timeout in seconds; 0 is indefinite."},
	},
	SubjectBluetoothDiscoveryFilterCmd: {
		{Name: "adapter", Type: "string", Required: true, Desc: "Bluetooth adapter path."},
		{Name: "filter", Type: "table", Required: false, Desc: "Discovery filter (uuids, rssi, transport, duplicate_data, discoverable, pattern)."},
	},

	// Audio
	SubjectAudioDevicesCmd: {},
	SubjectAudioSinkVolumeCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink index from the current snapshot."},
		{Name: "volume", Type: "int", Required: true, Desc: "Target volume 0–150 (100 = 0 dB)."},
	},
	SubjectAudioSinkMuteCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink index."},
		{Name: "mute", Type: "bool", Required: true, Desc: "True to mute, false to unmute."},
	},
	SubjectAudioSinkBalanceCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink index."},
		{Name: "volume", Type: "int", Required: true, Desc: "Balance -100 (left) to 100 (right)."},
	},
	SubjectAudioSourceVolumeCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source index from the current snapshot."},
		{Name: "volume", Type: "int", Required: true, Desc: "Target volume 0–150 (100 = 0 dB)."},
	},
	SubjectAudioSourceMuteCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source index."},
		{Name: "mute", Type: "bool", Required: true, Desc: "True to mute, false to unmute."},
	},
	SubjectAudioSourceBalanceCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source index."},
		{Name: "volume", Type: "int", Required: true, Desc: "Balance -100 (left) to 100 (right)."},
	},
	SubjectAudioDefaultSinkCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Sink device name (not description)."},
	},
	SubjectAudioDefaultSourceCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Source device name (not description)."},
	},
	SubjectAudioCardProfileCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Card index."},
		{Name: "profile", Type: "string", Required: true, Desc: "Card profile name (e.g. analog-stereo)."},
	},
	SubjectAudioSinkPortCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink index."},
		{Name: "port", Type: "string", Required: true, Desc: "Port name (e.g. analog-output-headphones)."},
	},
	SubjectAudioSourcePortCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source index."},
		{Name: "port", Type: "string", Required: true, Desc: "Port name (e.g. analog-input-microphone)."},
	},
	SubjectAudioSinkInputVolumeCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink input (per-application stream) index."},
		{Name: "volume", Type: "int", Required: true, Desc: "Target volume 0–150."},
	},
	SubjectAudioSinkInputMuteCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink input index."},
		{Name: "mute", Type: "bool", Required: true, Desc: "True to mute, false to unmute."},
	},
	SubjectAudioSinkInputMoveCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink input index."},
		{Name: "target_index", Type: "int", Required: true, Desc: "Destination sink index."},
	},
	SubjectAudioSourceOutputVolumeCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source output index."},
		{Name: "volume", Type: "int", Required: true, Desc: "Target volume 0–150."},
	},
	SubjectAudioSourceOutputMuteCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source output index."},
		{Name: "mute", Type: "bool", Required: true, Desc: "True to mute, false to unmute."},
	},
	SubjectAudioSourceOutputMoveCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source output index."},
		{Name: "target_index", Type: "int", Required: true, Desc: "Destination source index."},
	},
	SubjectAudioSinkInputKillCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink input index to terminate."},
	},
	SubjectAudioSourceOutputKillCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source output index to terminate."},
	},
	SubjectAudioSuspendSinkCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Sink index."},
		{Name: "suspend", Type: "bool", Required: true, Desc: "True to suspend, false to resume."},
	},
	SubjectAudioSuspendSourceCmd: {
		{Name: "index", Type: "int", Required: true, Desc: "Source index."},
		{Name: "suspend", Type: "bool", Required: true, Desc: "True to suspend, false to resume."},
	},

	// Network
	SubjectNetworkSnapshotCmd: {},
	SubjectNetworkReconfigureCmd: {
		{Name: "interface", Type: "string", Required: true, Desc: "Interface name (e.g. eth0)."},
	},
	SubjectNetworkConfigureIPv4Cmd: {
		{Name: "interface", Type: "string", Required: true, Desc: "Interface name."},
		{Name: "ipv4", Type: "table", Required: true, Desc: "IPv4 config: {method='dhcp'|'manual', address, netmask, gateway, dns={...}}."},
	},
	SubjectNetworkResetCmd: {
		{Name: "interface", Type: "string", Required: true, Desc: "Interface name."},
	},
	SubjectNetworkAirplaneCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable airplane mode, false to disable."},
	},

	// Display
	SubjectDisplayMonitorsCmd: {},
	SubjectDisplayResolutionCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Monitor name (e.g. eDP-1)."},
		{Name: "width", Type: "int", Required: true, Desc: "Resolution width in pixels."},
		{Name: "height", Type: "int", Required: true, Desc: "Resolution height in pixels."},
		{Name: "refresh_rate", Type: "float", Required: false, Desc: "Refresh rate in Hz."},
	},
	SubjectDisplayScaleCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Monitor name."},
		{Name: "scale", Type: "float", Required: true, Desc: "Scale factor (e.g. 1.0, 1.5, 2.0)."},
	},
	SubjectDisplayTransformCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Monitor name."},
		{Name: "transform", Type: "int", Required: true, Desc: "Rotation code 0–7 (Hyprland transform values)."},
	},
	SubjectDisplayPositionCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Monitor name."},
		{Name: "x", Type: "int", Required: true, Desc: "X offset in pixels."},
		{Name: "y", Type: "int", Required: true, Desc: "Y offset in pixels."},
	},
	SubjectDisplayDpmsCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Monitor name."},
		{Name: "on", Type: "bool", Required: true, Desc: "True to power on, false to power off."},
	},
	SubjectDisplayVrrCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Monitor name."},
		{Name: "mode", Type: "int", Required: true, Desc: "VRR mode (0=off, 1=on, 2=fullscreen-only)."},
	},
	SubjectDisplayMirrorCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Monitor name."},
		{Name: "mirror_of", Type: "string", Required: false, Desc: "Source monitor name to mirror, or empty to unset."},
	},
	SubjectDisplayToggleCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Monitor name."},
		{Name: "on", Type: "bool", Required: true, Desc: "True to enable the monitor, false to disable."},
	},
	SubjectDisplayIdentifyCmd: {},
	SubjectDisplayBrightnessCmd: {
		{Name: "pct", Type: "int", Required: true, Desc: "Brightness percentage 0–100."},
	},
	SubjectDisplayKbdBrightnessCmd: {
		{Name: "pct", Type: "int", Required: true, Desc: "Keyboard backlight percentage 0–100."},
	},
	SubjectDisplayNightLightCmd: {
		{Name: "enable", Type: "bool", Required: false, Desc: "True to turn on night light, false to turn off."},
		{Name: "temperature", Type: "int", Required: false, Desc: "Color temperature in Kelvin (default 4500)."},
		{Name: "gamma", Type: "int", Required: false, Desc: "Gamma percentage (default 100)."},
	},
	SubjectDisplayGammaCmd: {
		{Name: "pct", Type: "int", Required: true, Desc: "Gamma percentage."},
	},
	SubjectDisplaySaveProfileCmd: {
		{Name: "profile", Type: "string", Required: true, Desc: "Profile name."},
	},
	SubjectDisplayApplyProfileCmd: {
		{Name: "profile", Type: "string", Required: true, Desc: "Profile name."},
	},
	SubjectDisplayDeleteProfileCmd: {
		{Name: "profile", Type: "string", Required: true, Desc: "Profile name."},
	},
	SubjectDisplayGPUModeCmd: {
		{Name: "gpu_mode", Type: "string", Required: true, Desc: "GPU mode (e.g. hybrid, integrated, nvidia)."},
	},

	// Date & Time
	SubjectDateTimeSnapshotCmd: {},
	SubjectDateTimeTZCmd: {
		{Name: "timezone", Type: "string", Required: true, Desc: "IANA timezone (e.g. America/Los_Angeles)."},
	},
	SubjectDateTimeNTPCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable NTP sync, false to disable."},
	},
	SubjectDateTimeFormatCmd: {
		{Name: "format", Type: "string", Required: true, Desc: "Clock format string."},
	},
	SubjectDateTimeSetTimeCmd: {
		{Name: "time", Type: "string", Required: true, Desc: "RFC3339 timestamp."},
	},
	SubjectDateTimeRTCCmd: {
		{Name: "local", Type: "bool", Required: true, Desc: "True for local RTC, false for UTC."},
	},

	// Input
	SubjectInputSnapshotCmd: {},
	SubjectInputRepeatRateCmd: {
		{Name: "rate", Type: "int", Required: true, Desc: "Repeats per second."},
	},
	SubjectInputRepeatDelayCmd: {
		{Name: "delay", Type: "int", Required: true, Desc: "Delay before repeat in milliseconds."},
	},
	SubjectInputSensitivityCmd: {
		{Name: "sens", Type: "float", Required: true, Desc: "Pointer sensitivity -1.0 to 1.0."},
	},
	SubjectInputNatScrollCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True for natural scroll, false for classic."},
	},
	SubjectInputScrollFactorCmd: {
		{Name: "factor", Type: "float", Required: true, Desc: "Scroll speed multiplier."},
	},
	SubjectInputKBLayoutCmd: {
		{Name: "layout", Type: "string", Required: true, Desc: "Keyboard layout code (e.g. us, de, us,de)."},
	},
	SubjectInputAccelProfileCmd: {
		{Name: "profile", Type: "string", Required: true, Desc: "Acceleration profile (flat or adaptive)."},
	},
	SubjectInputForceNoAccelCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to disable mouse acceleration entirely."},
	},
	SubjectInputLeftHandedCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True for left-handed button mapping."},
	},
	SubjectInputDisableTypingCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to disable touchpad while typing."},
	},
	SubjectInputTapToClickCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable tap-to-click."},
	},
	SubjectInputTapAndDragCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable tap-and-drag."},
	},
	SubjectInputDragLockCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable drag lock."},
	},
	SubjectInputMiddleBtnCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable middle-button emulation."},
	},
	SubjectInputClickfingerCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True for clickfinger behavior, false for button areas."},
	},

	// Notifications
	SubjectNotifySnapshotCmd: {},
	SubjectNotifyDNDCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable do-not-disturb, false to resume."},
	},
	SubjectNotifyDismissCmd: {},
	SubjectNotifyAnchorCmd: {
		{Name: "anchor", Type: "string", Required: true, Desc: "Screen anchor (top-left, top-right, bottom-left, bottom-right, center-*)."},
	},
	SubjectNotifyTimeoutCmd: {
		{Name: "timeout", Type: "int", Required: true, Desc: "Default timeout in milliseconds."},
	},
	SubjectNotifyWidthCmd: {
		{Name: "width", Type: "int", Required: true, Desc: "Notification width in pixels."},
	},
	SubjectNotifyLayerCmd: {
		{Name: "layer", Type: "string", Required: true, Desc: "Layer name (overlay, top, bottom)."},
	},
	SubjectNotifySoundCmd: {
		{Name: "sound", Type: "string", Required: true, Desc: "Sound file path or name."},
	},
	SubjectNotifyAddRuleCmd: {
		{Name: "app_name", Type: "string", Required: true, Desc: "Application name to match."},
		{Name: "hide", Type: "bool", Required: false, Desc: "True to hide notifications from this app."},
	},
	SubjectNotifyRemoveRuleCmd: {
		{Name: "criteria", Type: "string", Required: true, Desc: "Rule criteria/app name to remove."},
	},

	// Power
	SubjectPowerSnapshotCmd: {},
	SubjectPowerProfileCmd: {
		{Name: "profile", Type: "string", Required: true, Desc: "Power profile (performance, balanced, power-saver)."},
	},
	SubjectPowerGovernorCmd: {
		{Name: "governor", Type: "string", Required: true, Desc: "CPU governor (performance, ondemand, powersave, schedutil)."},
	},
	SubjectPowerEPPCmd: {
		{Name: "epp", Type: "string", Required: true, Desc: "Energy Performance Preference (performance, balance_performance, balance_power, power)."},
	},
	SubjectPowerIdleCmd: {
		{Name: "idle_kind", Type: "string", Required: true, Desc: "Idle action (lock, suspend, blank, etc.)."},
		{Name: "idle_sec", Type: "int", Required: true, Desc: "Idle timeout in seconds."},
	},
	SubjectPowerIdleRunningCmd: {
		{Name: "idle_running", Type: "bool", Required: true, Desc: "True to honor idle while media is playing."},
	},
	SubjectPowerButtonCmd: {
		{Name: "button_key", Type: "string", Required: true, Desc: "Button identifier (power, sleep, lid)."},
		{Name: "button_val", Type: "string", Required: true, Desc: "Action (poweroff, suspend, reboot, lock, ignore)."},
	},

	// Privacy
	SubjectPrivacySnapshotCmd: {},
	SubjectPrivacyIdleCmd: {
		{Name: "field", Type: "string", Required: true, Desc: "Privacy idle field name."},
		{Name: "seconds", Type: "int", Required: false, Desc: "Idle timeout in seconds."},
	},
	SubjectPrivacyDNSTLSCmd: {
		{Name: "value", Type: "string", Required: true, Desc: "DNS-over-TLS mode (yes, opportunistic, no)."},
	},
	SubjectPrivacyDNSSECCmd: {
		{Name: "value", Type: "string", Required: true, Desc: "DNSSEC mode (yes, allow-downgrade, no)."},
	},
	SubjectPrivacyFirewallCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable ufw, false to disable."},
	},
	SubjectPrivacySSHCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable sshd, false to disable."},
	},
	SubjectPrivacyClearCmd: {},
	SubjectPrivacyLocationCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable geoclue, false to disable."},
	},
	SubjectPrivacyMACCmd: {
		{Name: "value", Type: "string", Required: true, Desc: "MAC randomization mode (permanent, random, stable)."},
	},
	SubjectPrivacyIndexerCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable the file indexer, false to disable."},
	},
	SubjectPrivacyCoredumpCmd: {
		{Name: "value", Type: "string", Required: true, Desc: "Core dump storage (none, external, journal)."},
	},

	// Users
	SubjectUsersSnapshotCmd: {},
	SubjectUsersAddCmd: {
		{Name: "username", Type: "string", Required: true, Desc: "New account username."},
		{Name: "full_name", Type: "string", Required: false, Desc: "GECOS full name."},
		{Name: "shell", Type: "string", Required: false, Desc: "Login shell (e.g. /bin/bash)."},
		{Name: "admin", Type: "bool", Required: false, Desc: "True to add to the wheel group."},
	},
	SubjectUsersRemoveCmd: {
		{Name: "username", Type: "string", Required: true, Desc: "Username to delete."},
		{Name: "remove_home", Type: "bool", Required: false, Desc: "True to also delete the home directory."},
	},
	SubjectUsersShellCmd: {
		{Name: "username", Type: "string", Required: true, Desc: "Account username."},
		{Name: "shell", Type: "string", Required: true, Desc: "New login shell."},
	},
	SubjectUsersCommentCmd: {
		{Name: "username", Type: "string", Required: true, Desc: "Account username."},
		{Name: "full_name", Type: "string", Required: false, Desc: "New GECOS full name."},
	},
	SubjectUsersLockCmd: {
		{Name: "username", Type: "string", Required: true, Desc: "Account username."},
		{Name: "admin", Type: "bool", Required: true, Desc: "False to lock, true to unlock."},
	},
	SubjectUsersGroupCmd: {
		{Name: "username", Type: "string", Required: true, Desc: "Account username."},
		{Name: "group", Type: "string", Required: true, Desc: "Group name."},
		{Name: "admin", Type: "bool", Required: true, Desc: "True to add to group, false to remove."},
	},
	SubjectUsersAdminCmd: {
		{Name: "username", Type: "string", Required: true, Desc: "Account username."},
		{Name: "admin", Type: "bool", Required: true, Desc: "True to grant wheel membership, false to revoke."},
	},
	SubjectUsersPasswdCmd: {
		{Name: "username", Type: "string", Required: true, Desc: "Account username."},
		{Name: "password", Type: "string", Required: true, Desc: "New password."},
		{Name: "current_pass", Type: "string", Required: false, Desc: "Current password when changing your own account."},
	},
	SubjectUsersElevateCmd: {
		{Name: "password", Type: "string", Required: true, Desc: "Current user's password for privilege elevation."},
	},

	// Appearance
	SubjectAppearanceSnapshotCmd: {},
	SubjectAppearanceGapsInCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Inner gap size in pixels."},
	},
	SubjectAppearanceGapsOutCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Outer gap size in pixels."},
	},
	SubjectAppearanceBorderCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Border width in pixels."},
	},
	SubjectAppearanceRoundingCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Corner radius in pixels."},
	},
	SubjectAppearanceBlurCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable window blur, false to disable."},
	},
	SubjectAppearanceBlurSizeCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Blur size."},
	},
	SubjectAppearanceBlurPassCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Number of blur passes."},
	},
	SubjectAppearanceAnimCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable animations, false to disable."},
	},
	SubjectAppearanceThemeCmd: {
		{Name: "theme", Type: "string", Required: true, Desc: "Omarchy theme name."},
	},
	SubjectAppearanceFontCmd: {
		{Name: "font", Type: "string", Required: true, Desc: "Font family name."},
	},
	SubjectAppearanceFontSizeCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Font size in points."},
	},
	SubjectAppearanceBackgroundCmd: {
		{Name: "background", Type: "string", Required: true, Desc: "Background image path or theme background ID."},
	},

	// Keybindings
	SubjectKeybindSnapshotCmd: {},
	SubjectKeybindAddCmd: {
		{Name: "mods", Type: "string", Required: true, Desc: "Modifier keys (e.g. SUPER, SUPER+SHIFT)."},
		{Name: "key", Type: "string", Required: true, Desc: "Primary key."},
		{Name: "desc", Type: "string", Required: false, Desc: "Human-readable description."},
		{Name: "dispatcher", Type: "string", Required: true, Desc: "Hyprland dispatcher (exec, killactive, movetoworkspace, ...)."},
		{Name: "args", Type: "string", Required: false, Desc: "Dispatcher arguments."},
		{Name: "source", Type: "string", Required: false, Desc: "Source label (user or system)."},
		{Name: "category", Type: "string", Required: false, Desc: "Category label."},
		{Name: "bind_type", Type: "string", Required: false, Desc: "Bind kind (bind, binde, bindl, bindel)."},
	},
	SubjectKeybindUpdateCmd: {
		{Name: "mods", Type: "string", Required: true, Desc: "New modifier keys."},
		{Name: "key", Type: "string", Required: true, Desc: "New primary key."},
		{Name: "desc", Type: "string", Required: false, Desc: "New description."},
		{Name: "dispatcher", Type: "string", Required: true, Desc: "New dispatcher."},
		{Name: "args", Type: "string", Required: false, Desc: "New dispatcher arguments."},
		{Name: "source", Type: "string", Required: false, Desc: "New source label."},
		{Name: "category", Type: "string", Required: false, Desc: "New category label."},
		{Name: "bind_type", Type: "string", Required: false, Desc: "New bind kind."},
		{Name: "old_mods", Type: "string", Required: true, Desc: "Original modifier keys."},
		{Name: "old_key", Type: "string", Required: true, Desc: "Original key."},
		{Name: "old_dispatcher", Type: "string", Required: true, Desc: "Original dispatcher."},
		{Name: "old_args", Type: "string", Required: false, Desc: "Original dispatcher arguments."},
	},
	SubjectKeybindRemoveCmd: {
		{Name: "mods", Type: "string", Required: true, Desc: "Modifier keys of binding to remove."},
		{Name: "key", Type: "string", Required: true, Desc: "Key of binding to remove."},
		{Name: "dispatcher", Type: "string", Required: true, Desc: "Dispatcher of binding to remove."},
		{Name: "args", Type: "string", Required: false, Desc: "Dispatcher arguments (to disambiguate)."},
	},

	// System updates
	SubjectUpdateSnapshotCmd: {},
	SubjectUpdateRunCmd:      {},
	SubjectUpdateChannelCmd: {
		{Name: "channel", Type: "string", Required: true, Desc: "Release channel (stable, rc, edge, dev)."},
	},

	// Firmware
	SubjectFirmwareSnapshotCmd: {},

	// Limine
	SubjectLimineSnapshotCmd: {},
	SubjectLimineCreateCmd: {
		{Name: "description", Type: "string", Required: false, Desc: "Snapshot description."},
	},
	SubjectLimineDeleteCmd: {
		{Name: "number", Type: "int", Required: true, Desc: "Snapshot number to delete."},
	},
	SubjectLimineSyncCmd: {},
	SubjectLimineDefaultEntryCmd: {
		{Name: "entry", Type: "int", Required: true, Desc: "Default boot entry index."},
	},
	SubjectLimineBootConfigCmd: {
		{Name: "key", Type: "string", Required: true, Desc: "Boot config key."},
		{Name: "value", Type: "string", Required: false, Desc: "New value."},
	},
	SubjectLimineSyncConfigCmd: {
		{Name: "key", Type: "string", Required: true, Desc: "Sync config key."},
		{Name: "value", Type: "string", Required: false, Desc: "New value."},
	},
	SubjectLimineOmarchyConfigCmd: {
		{Name: "key", Type: "string", Required: true, Desc: "Omarchy Limine config key."},
		{Name: "value", Type: "string", Required: false, Desc: "New value."},
	},
	SubjectLimineKernelCmdlineCmd: {
		{Name: "lines", Type: "[]string", Required: true, Desc: "Kernel command line tokens."},
	},

	// Screensaver
	SubjectScreensaverSnapshotCmd: {},
	SubjectScreensaverSetEnabledCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable the screensaver, false to disable."},
	},
	SubjectScreensaverSetContentCmd: {
		{Name: "content", Type: "string", Required: true, Desc: "ASCII/ANSI content to display."},
	},
	SubjectScreensaverPreviewCmd: {},

	// Top Bar (waybar)
	SubjectTopBarSnapshotCmd: {},
	SubjectTopBarSetRunningCmd: {
		{Name: "running", Type: "bool", Required: true, Desc: "True to start waybar, false to stop."},
	},
	SubjectTopBarRestartCmd: {},
	SubjectTopBarResetCmd:   {},
	SubjectTopBarSetPositionCmd: {
		{Name: "value", Type: "string", Required: true, Desc: "Position (top, bottom, left, right)."},
	},
	SubjectTopBarSetLayerCmd: {
		{Name: "value", Type: "string", Required: true, Desc: "Layer (overlay, top, bottom)."},
	},
	SubjectTopBarSetHeightCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Bar height in pixels."},
	},
	SubjectTopBarSetSpacingCmd: {
		{Name: "value", Type: "int", Required: true, Desc: "Spacing between items in pixels."},
	},
	SubjectTopBarSetConfigCmd: {
		{Name: "content", Type: "string", Required: true, Desc: "Raw waybar config.jsonc contents."},
	},
	SubjectTopBarSetStyleCmd: {
		{Name: "content", Type: "string", Required: true, Desc: "Raw waybar style.css contents."},
	},

	// Workspaces
	SubjectWorkspacesSnapshotCmd: {},
	SubjectWorkspacesSwitchCmd: {
		{Name: "id", Type: "int", Required: true, Desc: "Workspace ID."},
	},
	SubjectWorkspacesRenameCmd: {
		{Name: "id", Type: "int", Required: true, Desc: "Workspace ID."},
		{Name: "name", Type: "string", Required: true, Desc: "New workspace name."},
	},
	SubjectWorkspacesMoveToMonitorCmd: {
		{Name: "id", Type: "int", Required: true, Desc: "Workspace ID."},
		{Name: "monitor", Type: "string", Required: true, Desc: "Destination monitor name."},
	},
	SubjectWorkspacesSetLayoutCmd: {
		{Name: "id", Type: "int", Required: true, Desc: "Workspace ID."},
		{Name: "layout", Type: "string", Required: true, Desc: "Layout name (dwindle, master)."},
	},
	SubjectWorkspacesSetDefaultLayoutCmd: {
		{Name: "layout", Type: "string", Required: true, Desc: "Default layout name."},
	},
	SubjectWorkspacesSetDwindleOptionCmd: {
		{Name: "key", Type: "string", Required: true, Desc: "Dwindle option key (pseudotile, preserve_split, smart_split, force_split, ...)."},
		{Name: "value", Type: "string", Required: true, Desc: "New value."},
	},
	SubjectWorkspacesSetMasterOptionCmd: {
		{Name: "key", Type: "string", Required: true, Desc: "Master option key (new_status, new_on_top, mfact, orientation, ...)."},
		{Name: "value", Type: "string", Required: true, Desc: "New value."},
	},
	SubjectWorkspacesSetCursorWarpCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to warp the cursor on workspace switches."},
	},
	SubjectWorkspacesSetAnimationsCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to enable workspace animations."},
	},
	SubjectWorkspacesSetHideSpecialCmd: {
		{Name: "enabled", Type: "bool", Required: true, Desc: "True to hide special workspaces when switching."},
	},

	// Dark self-update
	SubjectDarkUpdateSnapshotCmd: {},
	SubjectDarkUpdateCheckCmd:    {},
	SubjectDarkUpdateApplyCmd: {
		{Name: "tag", Type: "string", Required: false, Desc: "Version tag to install; empty uses the latest release."},
	},

	// App Store
	SubjectAppstoreCatalogCmd: {},
	SubjectAppstoreSearchCmd: {
		{Name: "query", Type: "string", Required: false, Desc: "Search query."},
		{Name: "category", Type: "string", Required: false, Desc: "Category ID to filter by."},
		{Name: "include_aur", Type: "bool", Required: false, Desc: "True to include AUR results."},
	},
	SubjectAppstoreDetailCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Package name."},
		{Name: "origin", Type: "string", Required: false, Desc: "Origin (pacman or aur)."},
	},
	SubjectAppstoreRefreshCmd: {},
	SubjectAppstoreInstallCmd: {
		{Name: "names", Type: "[]string", Required: true, Desc: "Package names to install."},
		{Name: "origin", Type: "string", Required: false, Desc: "Origin (pacman or aur)."},
	},
	SubjectAppstoreRemoveCmd: {
		{Name: "names", Type: "[]string", Required: true, Desc: "Package names to remove."},
	},
	SubjectAppstoreUpgradeCmd: {},

	// Scripting (meta)
	SubjectScriptingListCmd:       {},
	SubjectScriptingRegistryCmd:   {},
	SubjectScriptingAPICatalogCmd: {},
	SubjectScriptingReadCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Script file name (basename with .lua)."},
	},
	SubjectScriptingSaveCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Script file name."},
		{Name: "content", Type: "string", Required: true, Desc: "Full file contents."},
	},
	SubjectScriptingDeleteCmd: {
		{Name: "name", Type: "string", Required: true, Desc: "Script file name."},
	},
	SubjectScriptingCallCmd: {
		{Name: "fn", Type: "string", Required: true, Desc: "Global Lua function name to invoke (e.g. volume_up)."},
		{Name: "args", Type: "[]string", Required: false, Desc: "Optional positional arguments; parsed as JSON literals when possible."},
	},
}
