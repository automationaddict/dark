package bus

import (
	"sort"
	"strings"
)

// APICommandEntry is one enumerated NATS command subject with a
// parsed domain/verb, a short human summary, and the documented
// payload field schema. The F5 Scripting "API" reference tab lists
// these so users can discover the command surface exposed by darkd.
type APICommandEntry struct {
	Subject string         `json:"subject"`
	Domain  string         `json:"domain"`
	Verb    string         `json:"verb"`
	Summary string         `json:"summary,omitempty"`
	Fields  []CommandField `json:"fields,omitempty"`
}

// commandSummaries maps every subject to a short one-line doc
// string. Every entry in subjects.go should have a summary here so
// the F5 reference browser has something readable for every row.
// When you add a new `dark.cmd.*` subject, add its summary too.
var commandSummaries = map[string]string{
	// System info
	SubjectSystemInfoCmd: "Return the current system info snapshot (hostname, uptime, load, memory).",

	// Wi-Fi
	SubjectWifiAdaptersCmd:      "Return the current Wi-Fi snapshot (adapters, networks, known networks).",
	SubjectWifiScanCmd:          "Trigger a Wi-Fi scan on the given adapter.",
	SubjectWifiConnectCmd:       "Connect an adapter to an SSID, optionally with passphrase.",
	SubjectWifiDisconnectCmd:    "Disconnect an adapter from its current network.",
	SubjectWifiForgetCmd:        "Forget a known Wi-Fi network so it no longer auto-connects.",
	SubjectWifiPowerCmd:         "Power an adapter on or off.",
	SubjectWifiAutoconnectCmd:   "Toggle auto-connect for a known SSID.",
	SubjectWifiConnectHiddenCmd: "Connect to a hidden SSID by name.",
	SubjectWifiAPStartCmd:       "Start an access point on the given adapter.",
	SubjectWifiAPStopCmd:        "Stop the access point on the given adapter.",

	// Bluetooth
	SubjectBluetoothAdaptersCmd:            "Return the current Bluetooth snapshot (adapters and their devices).",
	SubjectBluetoothPowerCmd:               "Power a Bluetooth controller on or off.",
	SubjectBluetoothDiscoverOnCmd:          "Start Bluetooth device discovery on an adapter.",
	SubjectBluetoothDiscoverOffCmd:         "Stop Bluetooth device discovery on an adapter.",
	SubjectBluetoothConnectCmd:             "Connect to a Bluetooth device by address.",
	SubjectBluetoothDisconnectCmd:          "Disconnect a Bluetooth device.",
	SubjectBluetoothPairCmd:                "Pair with a Bluetooth device, optionally with a PIN.",
	SubjectBluetoothRemoveCmd:              "Forget a paired Bluetooth device.",
	SubjectBluetoothTrustCmd:               "Mark a Bluetooth device as trusted or untrusted.",
	SubjectBluetoothDiscoverableCmd:        "Toggle whether an adapter advertises itself to other devices.",
	SubjectBluetoothAliasCmd:               "Set or clear a Bluetooth adapter's advertised friendly name.",
	SubjectBluetoothPairableCmd:            "Toggle whether an adapter accepts incoming pairing requests.",
	SubjectBluetoothBlockCmd:               "Block or unblock a Bluetooth device.",
	SubjectBluetoothCancelPairCmd:          "Cancel an in-progress pairing attempt.",
	SubjectBluetoothDiscoverableTimeoutCmd: "Set how long an adapter stays discoverable (seconds, 0 = indefinite).",
	SubjectBluetoothDiscoveryFilterCmd:     "Apply a discovery filter (UUIDs, RSSI, transport) to narrow scan results.",

	// Audio
	SubjectAudioDevicesCmd:           "Return the current audio snapshot (sinks, sources, cards, streams).",
	SubjectAudioSinkVolumeCmd:        "Set a sink's volume (0–150, 100 = 0 dB).",
	SubjectAudioSinkMuteCmd:          "Mute or unmute a sink.",
	SubjectAudioSinkBalanceCmd:       "Set a sink's left/right balance (-100 to 100).",
	SubjectAudioSourceVolumeCmd:      "Set a source's input volume (0–150).",
	SubjectAudioSourceMuteCmd:        "Mute or unmute a source.",
	SubjectAudioSourceBalanceCmd:     "Set a source's left/right balance (-100 to 100).",
	SubjectAudioDefaultSinkCmd:       "Change the system default audio sink by name.",
	SubjectAudioDefaultSourceCmd:     "Change the system default audio source by name.",
	SubjectAudioCardProfileCmd:       "Switch a card to a different profile (e.g. analog-stereo, hdmi-stereo).",
	SubjectAudioSinkPortCmd:          "Select a physical output port on a sink (e.g. headphones vs speakers).",
	SubjectAudioSourcePortCmd:        "Select a physical input port on a source.",
	SubjectAudioSinkInputVolumeCmd:   "Set a per-application stream's output volume.",
	SubjectAudioSinkInputMuteCmd:     "Mute or unmute a per-application output stream.",
	SubjectAudioSinkInputMoveCmd:     "Move a per-application output stream to a different sink.",
	SubjectAudioSourceOutputVolumeCmd: "Set a per-application capture stream's volume.",
	SubjectAudioSourceOutputMuteCmd:  "Mute or unmute a per-application capture stream.",
	SubjectAudioSourceOutputMoveCmd:  "Move a per-application capture stream to a different source.",
	SubjectAudioSinkInputKillCmd:     "Terminate a per-application output stream.",
	SubjectAudioSourceOutputKillCmd:  "Terminate a per-application capture stream.",
	SubjectAudioSuspendSinkCmd:       "Suspend or resume a sink (release hardware when idle).",
	SubjectAudioSuspendSourceCmd:     "Suspend or resume a source (release hardware when idle).",

	// Network
	SubjectNetworkSnapshotCmd:      "Return the current network snapshot (interfaces, routes, DNS).",
	SubjectNetworkReconfigureCmd:   "Rerun DHCP / reconfigure an interface from scratch.",
	SubjectNetworkConfigureIPv4Cmd: "Apply a static or DHCP IPv4 configuration to an interface.",
	SubjectNetworkResetCmd:         "Reset an interface (down/up) to clear transient state.",
	SubjectNetworkAirplaneCmd:      "Toggle airplane mode (disables Wi-Fi and Bluetooth together).",

	// Display
	SubjectDisplayMonitorsCmd:      "Return the current display snapshot (monitors, resolutions, layout).",
	SubjectDisplayResolutionCmd:    "Change a monitor's resolution (and optional refresh rate).",
	SubjectDisplayScaleCmd:         "Change a monitor's scale factor (e.g. 1.0, 1.5, 2.0).",
	SubjectDisplayTransformCmd:     "Rotate/flip a monitor (Hyprland transform code 0–7).",
	SubjectDisplayPositionCmd:      "Set a monitor's X/Y offset in the virtual desktop.",
	SubjectDisplayDpmsCmd:          "Power a monitor on or off via DPMS.",
	SubjectDisplayVrrCmd:           "Set a monitor's variable refresh rate mode.",
	SubjectDisplayMirrorCmd:        "Mirror a monitor onto another, or clear the mirror relationship.",
	SubjectDisplayToggleCmd:        "Enable or disable a monitor entirely.",
	SubjectDisplayIdentifyCmd:      "Briefly flash an identifier on each monitor to help pick one visually.",
	SubjectDisplayBrightnessCmd:    "Set the primary monitor's backlight brightness (0–100%).",
	SubjectDisplayKbdBrightnessCmd: "Set the keyboard backlight brightness (0–100%).",
	SubjectDisplayNightLightCmd:    "Toggle night-light (warm tint) with optional temperature/gamma.",
	SubjectDisplayGammaCmd:         "Set the display gamma percentage.",
	SubjectDisplaySaveProfileCmd:   "Save the current monitor layout as a named profile.",
	SubjectDisplayApplyProfileCmd:  "Apply a previously-saved monitor profile.",
	SubjectDisplayDeleteProfileCmd: "Delete a saved monitor profile.",
	SubjectDisplayGPUModeCmd:       "Switch GPU mode (hybrid, integrated, nvidia) on supported hardware.",

	// Date & Time
	SubjectDateTimeSnapshotCmd: "Return the current date/time snapshot (timezone, NTP, format).",
	SubjectDateTimeTZCmd:       "Set the system timezone.",
	SubjectDateTimeNTPCmd:      "Enable or disable NTP synchronization.",
	SubjectDateTimeFormatCmd:   "Set the system clock display format string.",
	SubjectDateTimeSetTimeCmd:  "Set the system time to a specific RFC3339 timestamp.",
	SubjectDateTimeRTCCmd:      "Switch the hardware clock between local time and UTC.",

	// Input
	SubjectInputSnapshotCmd:      "Return the current input-devices snapshot (keyboards, mice, touchpads).",
	SubjectInputRepeatRateCmd:    "Set the keyboard repeat rate (repeats per second).",
	SubjectInputRepeatDelayCmd:   "Set the keyboard repeat delay (ms before repeat starts).",
	SubjectInputSensitivityCmd:   "Set pointer sensitivity (-1.0 to 1.0).",
	SubjectInputNatScrollCmd:     "Toggle natural scrolling (reverse direction).",
	SubjectInputScrollFactorCmd:  "Set the scroll speed multiplier.",
	SubjectInputKBLayoutCmd:      "Change the keyboard layout (e.g. us, de, us,de).",
	SubjectInputAccelProfileCmd:  "Pick the pointer acceleration profile (flat or adaptive).",
	SubjectInputForceNoAccelCmd:  "Disable mouse acceleration entirely.",
	SubjectInputLeftHandedCmd:    "Swap left/right mouse buttons for left-handed use.",
	SubjectInputDisableTypingCmd: "Disable the touchpad while typing.",
	SubjectInputTapToClickCmd:    "Enable tap-to-click on the touchpad.",
	SubjectInputTapAndDragCmd:    "Enable tap-and-drag on the touchpad.",
	SubjectInputDragLockCmd:      "Enable drag lock on the touchpad.",
	SubjectInputMiddleBtnCmd:     "Enable middle-button emulation via simultaneous click.",
	SubjectInputClickfingerCmd:   "Switch the touchpad between clickfinger and button-area click methods.",

	// Notifications
	SubjectNotifySnapshotCmd:    "Return the current notification config snapshot.",
	SubjectNotifyDNDCmd:         "Toggle do-not-disturb.",
	SubjectNotifyDismissCmd:     "Dismiss all currently displayed notifications.",
	SubjectNotifyAnchorCmd:      "Set the screen anchor for notification popups.",
	SubjectNotifyTimeoutCmd:     "Set the default notification timeout (milliseconds).",
	SubjectNotifyWidthCmd:       "Set the notification popup width (pixels).",
	SubjectNotifyLayerCmd:       "Set the wayland layer the notification daemon renders on.",
	SubjectNotifySoundCmd:       "Set the sound file played for new notifications.",
	SubjectNotifyAddRuleCmd:     "Add a filter rule (e.g. hide notifications from an app).",
	SubjectNotifyRemoveRuleCmd:  "Remove a filter rule by criteria.",

	// Power
	SubjectPowerSnapshotCmd:    "Return the current power snapshot (profile, governor, idle).",
	SubjectPowerProfileCmd:     "Switch the power profile (performance, balanced, power-saver).",
	SubjectPowerGovernorCmd:    "Set the CPU governor (performance, ondemand, powersave, schedutil).",
	SubjectPowerEPPCmd:         "Set the Intel Energy Performance Preference (performance, balance, power).",
	SubjectPowerIdleCmd:        "Configure an idle action (lock, suspend, blank) and its timeout.",
	SubjectPowerIdleRunningCmd: "Toggle whether idle actions fire while media is playing.",
	SubjectPowerButtonCmd:      "Configure what the power/sleep/lid buttons do on press.",

	// Privacy
	SubjectPrivacySnapshotCmd:  "Return the current privacy snapshot.",
	SubjectPrivacyIdleCmd:      "Configure a privacy idle field (activity, location, etc.) and its timeout.",
	SubjectPrivacyDNSTLSCmd:    "Set DNS-over-TLS mode (yes, opportunistic, no).",
	SubjectPrivacyDNSSECCmd:    "Set DNSSEC mode (yes, allow-downgrade, no).",
	SubjectPrivacyFirewallCmd:  "Enable or disable the ufw firewall.",
	SubjectPrivacySSHCmd:       "Enable or disable the sshd service.",
	SubjectPrivacyClearCmd:     "Clear the recently-used files history.",
	SubjectPrivacyLocationCmd:  "Enable or disable geoclue location services.",
	SubjectPrivacyMACCmd:       "Set MAC address randomization mode (permanent, random, stable).",
	SubjectPrivacyIndexerCmd:   "Enable or disable the file indexer.",
	SubjectPrivacyCoredumpCmd:  "Set core dump storage policy (none, external, journal).",

	// Users
	SubjectUsersSnapshotCmd: "Return the current users snapshot (accounts, groups, shells).",
	SubjectUsersAddCmd:      "Create a new user account.",
	SubjectUsersRemoveCmd:   "Delete a user account, optionally removing the home directory.",
	SubjectUsersShellCmd:    "Change a user's login shell.",
	SubjectUsersCommentCmd:  "Change a user's GECOS full-name field.",
	SubjectUsersLockCmd:     "Lock or unlock a user account.",
	SubjectUsersGroupCmd:    "Add a user to or remove a user from a secondary group.",
	SubjectUsersAdminCmd:    "Grant or revoke wheel-group (admin) membership for a user.",
	SubjectUsersPasswdCmd:   "Change a user's password.",
	SubjectUsersElevateCmd:  "Elevate privileges for the current session via password check.",

	// Appearance
	SubjectAppearanceSnapshotCmd:   "Return the current appearance snapshot (gaps, borders, blur, theme, font).",
	SubjectAppearanceGapsInCmd:     "Set the inner gap size between windows (pixels).",
	SubjectAppearanceGapsOutCmd:    "Set the outer gap size around the workspace edge (pixels).",
	SubjectAppearanceBorderCmd:     "Set the window border width (pixels).",
	SubjectAppearanceRoundingCmd:   "Set the window corner rounding radius (pixels).",
	SubjectAppearanceBlurCmd:       "Enable or disable window background blur.",
	SubjectAppearanceBlurSizeCmd:   "Set the blur kernel size.",
	SubjectAppearanceBlurPassCmd:   "Set the number of blur passes.",
	SubjectAppearanceAnimCmd:       "Enable or disable window animations.",
	SubjectAppearanceThemeCmd:      "Switch the Omarchy theme.",
	SubjectAppearanceFontCmd:       "Change the primary UI font family.",
	SubjectAppearanceFontSizeCmd:   "Change the primary UI font size.",
	SubjectAppearanceBackgroundCmd: "Change the desktop background image.",

	// Keybindings
	SubjectKeybindSnapshotCmd: "Return the current keybinding snapshot.",
	SubjectKeybindAddCmd:      "Add a new Hyprland keybinding.",
	SubjectKeybindUpdateCmd:   "Rewrite an existing keybinding by matching its old mods/key/dispatcher.",
	SubjectKeybindRemoveCmd:   "Remove a keybinding by matching its mods/key/dispatcher.",

	// System updates
	SubjectUpdateSnapshotCmd: "Return the current Omarchy update snapshot.",
	SubjectUpdateRunCmd:      "Run the Omarchy update flow.",
	SubjectUpdateChannelCmd:  "Switch the Omarchy update channel (stable, rc, edge, dev).",

	// Firmware
	SubjectFirmwareSnapshotCmd: "Return the current firmware-update snapshot from fwupd.",

	// Limine (snapshots and bootloader config)
	SubjectLimineSnapshotCmd:      "Return the current Limine snapshot list and config.",
	SubjectLimineCreateCmd:        "Create a new Limine snapshot with an optional description.",
	SubjectLimineDeleteCmd:        "Delete a Limine snapshot by number.",
	SubjectLimineSyncCmd:          "Sync Limine bootloader metadata.",
	SubjectLimineDefaultEntryCmd:  "Set the default boot entry index.",
	SubjectLimineBootConfigCmd:    "Set a boot config key/value pair.",
	SubjectLimineSyncConfigCmd:    "Set a sync config key/value pair.",
	SubjectLimineOmarchyConfigCmd: "Set an Omarchy Limine config key/value pair.",
	SubjectLimineKernelCmdlineCmd: "Replace the kernel command line arguments.",

	// Screensaver
	SubjectScreensaverSnapshotCmd:   "Return the current screensaver snapshot.",
	SubjectScreensaverSetEnabledCmd: "Enable or disable the screensaver.",
	SubjectScreensaverSetContentCmd: "Set the ASCII/ANSI content the screensaver displays.",
	SubjectScreensaverPreviewCmd:    "Preview the screensaver without arming it.",

	// Top Bar (waybar)
	SubjectTopBarSnapshotCmd:    "Return the current top-bar snapshot (running state, position, config).",
	SubjectTopBarSetRunningCmd:  "Start or stop waybar.",
	SubjectTopBarRestartCmd:     "Restart waybar.",
	SubjectTopBarResetCmd:       "Reset waybar config and style to their defaults.",
	SubjectTopBarSetPositionCmd: "Set waybar's screen position (top, bottom, left, right).",
	SubjectTopBarSetLayerCmd:    "Set waybar's wayland layer (overlay, top, bottom).",
	SubjectTopBarSetHeightCmd:   "Set waybar's height in pixels.",
	SubjectTopBarSetSpacingCmd:  "Set the spacing between waybar modules in pixels.",
	SubjectTopBarSetConfigCmd:   "Overwrite waybar's config.jsonc with new contents.",
	SubjectTopBarSetStyleCmd:    "Overwrite waybar's style.css with new contents.",

	// Workspaces
	SubjectWorkspacesSnapshotCmd:         "Return the current workspaces snapshot.",
	SubjectWorkspacesSwitchCmd:           "Switch to a workspace by ID.",
	SubjectWorkspacesRenameCmd:           "Rename a workspace.",
	SubjectWorkspacesMoveToMonitorCmd:    "Move a workspace to a different monitor.",
	SubjectWorkspacesSetLayoutCmd:        "Set a workspace's layout (dwindle, master).",
	SubjectWorkspacesSetDefaultLayoutCmd: "Set the default layout for new workspaces.",
	SubjectWorkspacesSetDwindleOptionCmd: "Set a dwindle-layout option (pseudotile, preserve_split, etc.).",
	SubjectWorkspacesSetMasterOptionCmd:  "Set a master-layout option (new_status, orientation, etc.).",
	SubjectWorkspacesSetCursorWarpCmd:    "Toggle cursor warping on workspace switches.",
	SubjectWorkspacesSetAnimationsCmd:    "Toggle workspace animations.",
	SubjectWorkspacesSetHideSpecialCmd:   "Toggle hiding special workspaces when switching.",

	// App Store
	SubjectAppstoreCatalogCmd: "Return the current package catalog snapshot (categories, counts).",
	SubjectAppstoreSearchCmd:  "Search the package catalog.",
	SubjectAppstoreDetailCmd:  "Fetch detail for a single package.",
	SubjectAppstoreRefreshCmd: "Force a refresh of the package catalog.",
	SubjectAppstoreInstallCmd: "Install one or more packages.",
	SubjectAppstoreRemoveCmd:  "Remove one or more packages.",
	SubjectAppstoreUpgradeCmd: "Upgrade all installed packages.",

	// Dark self-update
	SubjectDarkUpdateSnapshotCmd: "Return the current dark self-update snapshot.",
	SubjectDarkUpdateCheckCmd:    "Check GitHub for a new dark release.",
	SubjectDarkUpdateApplyCmd:    "Download and install the latest dark release.",

	// Scripting meta
	SubjectScriptingListCmd:       "List user Lua scripts in ~/.config/dark/scripts/.",
	SubjectScriptingRegistryCmd:   "List Lua host functions and event hook points.",
	SubjectScriptingAPICatalogCmd: "Return the enumerated dark.cmd.* API catalog.",
	SubjectScriptingMCPCatalogCmd: "Return the MCP tool and resource catalog exposed by `dark mcp`.",
	SubjectScriptingReadCmd:       "Read the full contents of a user Lua script by basename.",
	SubjectScriptingSaveCmd:       "Create or overwrite a user Lua script file.",
	SubjectScriptingDeleteCmd:     "Delete a user Lua script file.",
	SubjectScriptingCallCmd:       "Invoke a Lua global function by name with optional JSON arguments.",
	SubjectScriptingReloadCmd:     "Clear every user hook and re-run every user Lua script from disk.",
}

// allCommandSubjects is the authoritative list of command subjects
// surfaced in the F5 Scripting > API tab. It is maintained by hand
// because Go constants are not reflectable — when a new dark.cmd.*
// subject is added, append it here to make it discoverable.
var allCommandSubjects = []string{
	SubjectSystemInfoCmd,
	SubjectWifiAdaptersCmd, SubjectWifiScanCmd, SubjectWifiConnectCmd,
	SubjectWifiDisconnectCmd, SubjectWifiForgetCmd, SubjectWifiPowerCmd,
	SubjectWifiAutoconnectCmd, SubjectWifiConnectHiddenCmd,
	SubjectWifiAPStartCmd, SubjectWifiAPStopCmd,

	SubjectBluetoothAdaptersCmd, SubjectBluetoothPowerCmd,
	SubjectBluetoothDiscoverOnCmd, SubjectBluetoothDiscoverOffCmd,
	SubjectBluetoothConnectCmd, SubjectBluetoothDisconnectCmd,
	SubjectBluetoothPairCmd, SubjectBluetoothRemoveCmd,
	SubjectBluetoothTrustCmd, SubjectBluetoothDiscoverableCmd,
	SubjectBluetoothAliasCmd, SubjectBluetoothPairableCmd,
	SubjectBluetoothBlockCmd, SubjectBluetoothCancelPairCmd,
	SubjectBluetoothDiscoverableTimeoutCmd, SubjectBluetoothDiscoveryFilterCmd,

	SubjectAudioDevicesCmd, SubjectAudioSinkVolumeCmd, SubjectAudioSinkMuteCmd,
	SubjectAudioSinkBalanceCmd, SubjectAudioSourceVolumeCmd, SubjectAudioSourceMuteCmd,
	SubjectAudioSourceBalanceCmd, SubjectAudioDefaultSinkCmd, SubjectAudioDefaultSourceCmd,
	SubjectAudioCardProfileCmd, SubjectAudioSinkPortCmd, SubjectAudioSourcePortCmd,
	SubjectAudioSinkInputVolumeCmd, SubjectAudioSinkInputMuteCmd, SubjectAudioSinkInputMoveCmd,
	SubjectAudioSourceOutputVolumeCmd, SubjectAudioSourceOutputMuteCmd, SubjectAudioSourceOutputMoveCmd,
	SubjectAudioSinkInputKillCmd, SubjectAudioSourceOutputKillCmd,
	SubjectAudioSuspendSinkCmd, SubjectAudioSuspendSourceCmd,

	SubjectNetworkSnapshotCmd, SubjectNetworkReconfigureCmd,
	SubjectNetworkConfigureIPv4Cmd, SubjectNetworkResetCmd, SubjectNetworkAirplaneCmd,

	SubjectDisplayMonitorsCmd, SubjectDisplayResolutionCmd, SubjectDisplayScaleCmd,
	SubjectDisplayTransformCmd, SubjectDisplayPositionCmd, SubjectDisplayDpmsCmd,
	SubjectDisplayVrrCmd, SubjectDisplayMirrorCmd, SubjectDisplayToggleCmd,
	SubjectDisplayIdentifyCmd, SubjectDisplayBrightnessCmd, SubjectDisplayKbdBrightnessCmd,
	SubjectDisplayNightLightCmd, SubjectDisplayGammaCmd,
	SubjectDisplaySaveProfileCmd, SubjectDisplayApplyProfileCmd, SubjectDisplayDeleteProfileCmd,
	SubjectDisplayGPUModeCmd,

	SubjectDateTimeSnapshotCmd, SubjectDateTimeTZCmd, SubjectDateTimeNTPCmd,
	SubjectDateTimeFormatCmd, SubjectDateTimeSetTimeCmd, SubjectDateTimeRTCCmd,

	SubjectInputSnapshotCmd, SubjectInputRepeatRateCmd, SubjectInputRepeatDelayCmd,
	SubjectInputSensitivityCmd, SubjectInputNatScrollCmd, SubjectInputScrollFactorCmd,
	SubjectInputKBLayoutCmd, SubjectInputAccelProfileCmd, SubjectInputForceNoAccelCmd,
	SubjectInputLeftHandedCmd, SubjectInputDisableTypingCmd, SubjectInputTapToClickCmd,
	SubjectInputTapAndDragCmd, SubjectInputDragLockCmd, SubjectInputMiddleBtnCmd,
	SubjectInputClickfingerCmd,

	SubjectNotifySnapshotCmd, SubjectNotifyDNDCmd, SubjectNotifyDismissCmd,
	SubjectNotifyAnchorCmd, SubjectNotifyTimeoutCmd, SubjectNotifyWidthCmd,
	SubjectNotifyLayerCmd, SubjectNotifySoundCmd, SubjectNotifyAddRuleCmd,
	SubjectNotifyRemoveRuleCmd,

	SubjectPowerSnapshotCmd, SubjectPowerProfileCmd, SubjectPowerGovernorCmd,
	SubjectPowerEPPCmd, SubjectPowerIdleCmd, SubjectPowerIdleRunningCmd,
	SubjectPowerButtonCmd,

	SubjectAppstoreCatalogCmd, SubjectAppstoreSearchCmd, SubjectAppstoreDetailCmd,
	SubjectAppstoreRefreshCmd, SubjectAppstoreInstallCmd, SubjectAppstoreRemoveCmd,
	SubjectAppstoreUpgradeCmd,

	SubjectPrivacySnapshotCmd, SubjectPrivacyIdleCmd, SubjectPrivacyDNSTLSCmd,
	SubjectPrivacyDNSSECCmd, SubjectPrivacyFirewallCmd, SubjectPrivacySSHCmd,
	SubjectPrivacyClearCmd, SubjectPrivacyLocationCmd, SubjectPrivacyMACCmd,
	SubjectPrivacyIndexerCmd, SubjectPrivacyCoredumpCmd,

	SubjectUsersSnapshotCmd, SubjectUsersAddCmd, SubjectUsersRemoveCmd,
	SubjectUsersShellCmd, SubjectUsersCommentCmd, SubjectUsersLockCmd,
	SubjectUsersGroupCmd, SubjectUsersAdminCmd, SubjectUsersPasswdCmd,
	SubjectUsersElevateCmd,

	SubjectAppearanceSnapshotCmd, SubjectAppearanceGapsInCmd, SubjectAppearanceGapsOutCmd,
	SubjectAppearanceBorderCmd, SubjectAppearanceRoundingCmd, SubjectAppearanceBlurCmd,
	SubjectAppearanceBlurSizeCmd, SubjectAppearanceBlurPassCmd, SubjectAppearanceAnimCmd,
	SubjectAppearanceThemeCmd, SubjectAppearanceFontCmd, SubjectAppearanceFontSizeCmd,
	SubjectAppearanceBackgroundCmd,

	SubjectKeybindSnapshotCmd, SubjectKeybindAddCmd, SubjectKeybindUpdateCmd,
	SubjectKeybindRemoveCmd,

	SubjectUpdateSnapshotCmd, SubjectUpdateRunCmd, SubjectUpdateChannelCmd,
	SubjectFirmwareSnapshotCmd,

	SubjectLimineSnapshotCmd, SubjectLimineCreateCmd, SubjectLimineDeleteCmd,
	SubjectLimineSyncCmd, SubjectLimineDefaultEntryCmd, SubjectLimineBootConfigCmd,
	SubjectLimineSyncConfigCmd, SubjectLimineOmarchyConfigCmd, SubjectLimineKernelCmdlineCmd,

	SubjectScreensaverSnapshotCmd, SubjectScreensaverSetEnabledCmd,
	SubjectScreensaverSetContentCmd, SubjectScreensaverPreviewCmd,

	SubjectTopBarSnapshotCmd, SubjectTopBarSetRunningCmd, SubjectTopBarRestartCmd,
	SubjectTopBarResetCmd, SubjectTopBarSetPositionCmd, SubjectTopBarSetLayerCmd,
	SubjectTopBarSetHeightCmd, SubjectTopBarSetSpacingCmd, SubjectTopBarSetConfigCmd,
	SubjectTopBarSetStyleCmd,

	SubjectWorkspacesSnapshotCmd, SubjectWorkspacesSwitchCmd, SubjectWorkspacesRenameCmd,
	SubjectWorkspacesMoveToMonitorCmd, SubjectWorkspacesSetLayoutCmd,
	SubjectWorkspacesSetDefaultLayoutCmd, SubjectWorkspacesSetDwindleOptionCmd,
	SubjectWorkspacesSetMasterOptionCmd, SubjectWorkspacesSetCursorWarpCmd,
	SubjectWorkspacesSetAnimationsCmd, SubjectWorkspacesSetHideSpecialCmd,

	SubjectDarkUpdateSnapshotCmd, SubjectDarkUpdateCheckCmd, SubjectDarkUpdateApplyCmd,

	SubjectScriptingListCmd, SubjectScriptingRegistryCmd, SubjectScriptingAPICatalogCmd,
	SubjectScriptingMCPCatalogCmd,
	SubjectScriptingReadCmd, SubjectScriptingSaveCmd, SubjectScriptingDeleteCmd,
	SubjectScriptingCallCmd, SubjectScriptingReloadCmd,
}

// APICommandCatalog returns every known dark.cmd.* subject sorted by
// domain then verb, with curated summaries where available. The F5
// Scripting > API tab renders the returned slice directly.
func APICommandCatalog() []APICommandEntry {
	out := make([]APICommandEntry, 0, len(allCommandSubjects))
	seen := map[string]bool{}
	for _, subj := range allCommandSubjects {
		if seen[subj] {
			continue
		}
		seen[subj] = true
		domain, verb := parseCommandSubject(subj)
		out = append(out, APICommandEntry{
			Subject: subj,
			Domain:  domain,
			Verb:    verb,
			Summary: commandSummaries[subj],
			Fields:  commandSchemas[subj],
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Domain != out[j].Domain {
			return out[i].Domain < out[j].Domain
		}
		return out[i].Verb < out[j].Verb
	})
	return out
}

// parseCommandSubject extracts the domain and verb from a standard
// `dark.cmd.<domain>.<verb>` subject. Anything outside that shape
// returns empty strings, which simply renders the subject unsorted.
func parseCommandSubject(subject string) (string, string) {
	parts := strings.Split(subject, ".")
	if len(parts) < 4 || parts[0] != "dark" || parts[1] != "cmd" {
		return "", ""
	}
	domain := parts[2]
	verb := strings.Join(parts[3:], ".")
	return domain, verb
}
