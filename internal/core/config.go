// Package core contains the shared configuration constants for the
// dark ecosystem. Timing values, dimensions, limits, and path
// conventions live here so tuning is a single-file edit instead of
// grepping across 15+ files. Services, daemon, and TUI all import
// from this file.
//
// Nothing in this file is user-configurable at runtime yet — these
// are compile-time defaults. A future config-file or env-var layer
// can read overrides and populate the same constants.
package core

import "time"

// --- Daemon tick intervals ---
// How often darkd publishes periodic snapshots for each service.
// Faster ticks = more responsive UI on changes made by other tools.
// Slower ticks = less CPU on low-power devices.

const (
	TickHeartbeat = 1 * time.Second
	TickSysInfo   = 2 * time.Second
	TickWifi      = 30 * time.Second
	TickBluetooth = 15 * time.Second
	TickAudio     = 30 * time.Second  // safety-net; audio uses event subscription
	TickNetwork   = 10 * time.Second
	TickDisplay   = 10 * time.Second
	TickAppstore  = 60 * time.Second
	TickWorkspaces = 3 * time.Second // workspaces change often; fast poll keeps the live list fresh
)

// --- Client NATS request timeouts ---
// How long the TUI waits for a response from darkd before giving up.
// Grouped by expected operation duration.

const (
	TimeoutFast     = 1 * time.Second   // initial snapshot fetches on startup
	TimeoutNormal   = 10 * time.Second  // typical D-Bus property reads/writes
	TimeoutSlow     = 15 * time.Second  // scans, searches, network reconfigure
	TimeoutConnect  = 20 * time.Second  // wifi connect, bluetooth connect
	TimeoutLong     = 25 * time.Second  // wifi connect with passphrase
	TimeoutPair     = 60 * time.Second  // bluetooth pairing (user interaction)
	TimeoutRefresh  = 30 * time.Second  // appstore catalog refresh
	TimeoutPkexec   = 120 * time.Second // operations behind a polkit dialog
)

// --- Audio ---

const (
	VolumeStepPercent = 5 // +/- per keypress (percentage points)
)

// --- Shutdown ---

const (
	ShutdownTimeout = 5 * time.Second // force-exit if defers hang
)

// --- Daemon-side timing ---

const (
	AudioEventDebounce   = 75 * time.Millisecond  // coalesce pulse events before snapshot
	DisplayEventDebounce = 200 * time.Millisecond
	AudioMeterTickRate  = 50 * time.Millisecond  // 20 Hz VU meter publish
	NotifyDebounce      = 30 * time.Second       // suppress duplicate daemon notifications
	// IWDScanPollInterval and IWDAPTransitionWait live in
	// internal/services/wifi/iwd.go to avoid an import cycle
	// (core → services/wifi → core).
	IWDPowerSettleWait  = 150 * time.Millisecond // wait after toggling iwd radio power
	IWDAPSnapshotWait   = 250 * time.Millisecond // wait for iwd to publish AP state before snapshot
	BlueZPowerWait      = 150 * time.Millisecond // wait after toggling bluez adapter power
)

// --- App Store ---

const (
	AppstoreCategoryLimit = 150 // max results when browsing a category
	AppstoreSearchLimit   = 200 // max results for a text search
)
