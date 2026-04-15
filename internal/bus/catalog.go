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

// commandSummaries maps a subject to a short doc string. Subjects
// without an entry here still appear in the catalog with an empty
// summary — the list is curated, not exhaustive, so backfilling
// is incremental.
var commandSummaries = map[string]string{
	SubjectWifiScanCmd:              "Trigger a Wi-Fi scan on the given adapter.",
	SubjectWifiConnectCmd:           "Connect an adapter to an SSID, optionally with passphrase.",
	SubjectWifiDisconnectCmd:        "Disconnect an adapter from its current network.",
	SubjectWifiForgetCmd:            "Forget a known Wi-Fi network so it no longer auto-connects.",
	SubjectWifiPowerCmd:             "Power an adapter on or off.",
	SubjectWifiAutoconnectCmd:       "Toggle auto-connect for a known SSID.",
	SubjectWifiConnectHiddenCmd:     "Connect to a hidden SSID by name.",
	SubjectWifiAPStartCmd:           "Start an access point on the given adapter.",
	SubjectWifiAPStopCmd:            "Stop the access point on the given adapter.",
	SubjectBluetoothPowerCmd:        "Power a Bluetooth controller on or off.",
	SubjectBluetoothConnectCmd:      "Connect to a Bluetooth device by address.",
	SubjectBluetoothDisconnectCmd:   "Disconnect a Bluetooth device.",
	SubjectBluetoothPairCmd:         "Pair with a Bluetooth device.",
	SubjectBluetoothRemoveCmd:       "Forget a paired Bluetooth device.",
	SubjectAudioSinkVolumeCmd:       "Set a sink's volume (0–150).",
	SubjectAudioSinkMuteCmd:         "Mute or unmute a sink.",
	SubjectAudioSourceVolumeCmd:    "Set a source's input volume.",
	SubjectAudioDefaultSinkCmd:      "Change the system default audio sink.",
	SubjectAudioDefaultSourceCmd:    "Change the system default audio source.",
	SubjectDisplayResolutionCmd:     "Change a monitor's resolution.",
	SubjectDisplayScaleCmd:          "Change a monitor's scale factor.",
	SubjectDisplayBrightnessCmd:     "Set a monitor's backlight brightness.",
	SubjectDisplayNightLightCmd:     "Toggle night-light (warm) tint.",
	SubjectDateTimeTZCmd:            "Set the system timezone.",
	SubjectDateTimeNTPCmd:           "Enable or disable NTP synchronization.",
	SubjectAppstoreSearchCmd:        "Search the package catalog.",
	SubjectAppstoreDetailCmd:        "Fetch detail for a single package.",
	SubjectAppstoreInstallCmd:       "Install one or more packages.",
	SubjectAppstoreRemoveCmd:        "Remove one or more packages.",
	SubjectAppstoreUpgradeCmd:       "Upgrade all installed packages.",
	SubjectUpdateRunCmd:             "Run the Omarchy update flow.",
	SubjectDarkUpdateCheckCmd:       "Check GitHub for a new dark release.",
	SubjectDarkUpdateApplyCmd:       "Download and install the latest dark release.",
	SubjectScriptingListCmd:         "List user Lua scripts in ~/.config/dark/scripts/.",
	SubjectScriptingRegistryCmd:     "List Lua host functions and event hook points.",
	SubjectScriptingAPICatalogCmd:   "Return the enumerated dark.cmd.* API catalog.",
	SubjectScriptingReadCmd:         "Read the full contents of a user Lua script by basename.",
	SubjectScriptingSaveCmd:         "Create or overwrite a user Lua script file.",
	SubjectScriptingDeleteCmd:       "Delete a user Lua script file.",
	SubjectScriptingCallCmd:         "Invoke a Lua global function by name with optional JSON arguments.",
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
	SubjectScriptingReadCmd, SubjectScriptingSaveCmd, SubjectScriptingDeleteCmd,
	SubjectScriptingCallCmd,
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
