package bus

// Subject constants for the dark message bus. The naming convention is:
//
//	dark.<domain>.<resource>         events, server -> clients
//	dark.cmd.<domain>.<verb>         commands, client -> server (request/reply)
//	dark.daemon.<event>              daemon lifecycle
//
// Clients should use wildcard subscriptions where it makes sense
// (e.g. dark.> for everything, dark.wifi.> for all Wi-Fi events).
const (
	SubjectDaemonHeartbeat = "dark.daemon.heartbeat"

	SubjectSystemInfo    = "dark.system.info"
	SubjectSystemInfoCmd = "dark.cmd.system.info"

	SubjectWifiAdapters      = "dark.wifi.adapters"
	SubjectWifiAdaptersCmd   = "dark.cmd.wifi.adapters"
	SubjectWifiScanCmd       = "dark.cmd.wifi.scan"
	SubjectWifiConnectCmd    = "dark.cmd.wifi.connect"
	SubjectWifiDisconnectCmd = "dark.cmd.wifi.disconnect"
	SubjectWifiForgetCmd     = "dark.cmd.wifi.forget"
	SubjectWifiPowerCmd         = "dark.cmd.wifi.power"
	SubjectWifiAutoconnectCmd   = "dark.cmd.wifi.autoconnect"
	SubjectWifiConnectHiddenCmd = "dark.cmd.wifi.connect_hidden"
	SubjectWifiAPStartCmd       = "dark.cmd.wifi.ap_start"
	SubjectWifiAPStopCmd        = "dark.cmd.wifi.ap_stop"

	SubjectBluetoothAdapters       = "dark.bluetooth.adapters"
	SubjectBluetoothAdaptersCmd    = "dark.cmd.bluetooth.adapters"
	SubjectBluetoothPowerCmd       = "dark.cmd.bluetooth.power"
	SubjectBluetoothDiscoverOnCmd  = "dark.cmd.bluetooth.discovery_start"
	SubjectBluetoothDiscoverOffCmd = "dark.cmd.bluetooth.discovery_stop"
	SubjectBluetoothConnectCmd     = "dark.cmd.bluetooth.connect"
	SubjectBluetoothDisconnectCmd  = "dark.cmd.bluetooth.disconnect"
	SubjectBluetoothPairCmd        = "dark.cmd.bluetooth.pair"
	SubjectBluetoothRemoveCmd      = "dark.cmd.bluetooth.remove"
	SubjectBluetoothTrustCmd        = "dark.cmd.bluetooth.trust"
	SubjectBluetoothDiscoverableCmd = "dark.cmd.bluetooth.discoverable"
	SubjectBluetoothAliasCmd        = "dark.cmd.bluetooth.alias"
	SubjectBluetoothPairableCmd            = "dark.cmd.bluetooth.pairable"
	SubjectBluetoothBlockCmd               = "dark.cmd.bluetooth.block"
	SubjectBluetoothCancelPairCmd          = "dark.cmd.bluetooth.cancel_pair"
	SubjectBluetoothDiscoverableTimeoutCmd = "dark.cmd.bluetooth.discoverable_timeout"
	SubjectBluetoothDiscoveryFilterCmd     = "dark.cmd.bluetooth.discovery_filter"

	SubjectAudioDevices         = "dark.audio.devices"
	SubjectAudioDevicesCmd      = "dark.cmd.audio.devices"
	SubjectAudioSinkVolumeCmd   = "dark.cmd.audio.sink_volume"
	SubjectAudioSinkMuteCmd     = "dark.cmd.audio.sink_mute"
	SubjectAudioSourceVolumeCmd = "dark.cmd.audio.source_volume"
	SubjectAudioSourceMuteCmd   = "dark.cmd.audio.source_mute"
	SubjectAudioDefaultSinkCmd   = "dark.cmd.audio.default_sink"
	SubjectAudioDefaultSourceCmd = "dark.cmd.audio.default_source"
	SubjectAudioCardProfileCmd       = "dark.cmd.audio.card_profile"
	SubjectAudioSinkPortCmd          = "dark.cmd.audio.sink_port"
	SubjectAudioSourcePortCmd        = "dark.cmd.audio.source_port"
	SubjectAudioSinkInputVolumeCmd   = "dark.cmd.audio.sink_input_volume"
	SubjectAudioSinkInputMuteCmd     = "dark.cmd.audio.sink_input_mute"
	SubjectAudioSinkInputMoveCmd     = "dark.cmd.audio.sink_input_move"
	SubjectAudioSourceOutputVolumeCmd = "dark.cmd.audio.source_output_volume"
	SubjectAudioSourceOutputMuteCmd  = "dark.cmd.audio.source_output_mute"
	SubjectAudioSourceOutputMoveCmd  = "dark.cmd.audio.source_output_move"
	SubjectAudioSinkInputKillCmd     = "dark.cmd.audio.sink_input_kill"
	SubjectAudioSourceOutputKillCmd  = "dark.cmd.audio.source_output_kill"
	SubjectAudioSuspendSinkCmd       = "dark.cmd.audio.suspend_sink"
	SubjectAudioSuspendSourceCmd     = "dark.cmd.audio.suspend_source"
	SubjectAudioLevels               = "dark.audio.levels"

	SubjectNetworkSnapshot         = "dark.network.snapshot"
	SubjectNetworkSnapshotCmd      = "dark.cmd.network.snapshot"
	SubjectNetworkReconfigureCmd   = "dark.cmd.network.reconfigure"
	SubjectNetworkConfigureIPv4Cmd = "dark.cmd.network.configure_ipv4"
	SubjectNetworkResetCmd         = "dark.cmd.network.reset"

	SubjectAppstoreCatalog    = "dark.appstore.catalog"
	SubjectAppstoreCatalogCmd = "dark.cmd.appstore.catalog"
	SubjectAppstoreSearchCmd  = "dark.cmd.appstore.search"
	SubjectAppstoreDetailCmd  = "dark.cmd.appstore.detail"
	SubjectAppstoreRefreshCmd = "dark.cmd.appstore.refresh"
	SubjectAppstoreInstallCmd = "dark.cmd.appstore.install"
	SubjectAppstoreRemoveCmd  = "dark.cmd.appstore.remove"
	SubjectAppstoreUpgradeCmd = "dark.cmd.appstore.upgrade"
)
