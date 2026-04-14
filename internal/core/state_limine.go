package core

import "github.com/johnnelson/dark/internal/services/limine"

// LimineSection describes a sub-section inside the "Limine" omarchy
// section. The F3 limine area has a snapshots table plus three
// config panes, one per file the service touches.
type LimineSection struct {
	ID    string
	Icon  string
	Label string
}

func LimineSections() []LimineSection {
	return []LimineSection{
		{"snapshots", "󰨤", "Snapshots"},
		{"boot", "󰒓", "Boot Config"},
		{"sync", "󰓦", "Sync Config"},
		{"omarchy", "󰣛", "Omarchy Defaults"},
	}
}

func (s *State) ActiveLimineSection() LimineSection {
	secs := LimineSections()
	if s.LimineSubIdx >= len(secs) {
		return secs[0]
	}
	return secs[s.LimineSubIdx]
}

func (s *State) MoveLimineSection(delta int) {
	n := len(LimineSections())
	if n == 0 {
		return
	}
	s.LimineSubIdx = (s.LimineSubIdx + delta + n) % n
}

func (s *State) SetLimine(snap limine.Snapshot) {
	s.Limine = snap
	s.LimineLoaded = true
	if s.LimineSnapshotIdx >= len(snap.Snapshots) {
		s.LimineSnapshotIdx = 0
	}
}

func (s *State) MoveLimineSnapshot(delta int) {
	n := len(s.Limine.Snapshots)
	if n == 0 {
		return
	}
	s.LimineSnapshotIdx = (s.LimineSnapshotIdx + delta + n) % n
}

func (s *State) SelectedLimineSnapshot() (limine.BootSnapshot, bool) {
	if len(s.Limine.Snapshots) == 0 {
		return limine.BootSnapshot{}, false
	}
	if s.LimineSnapshotIdx >= len(s.Limine.Snapshots) {
		s.LimineSnapshotIdx = 0
	}
	return s.Limine.Snapshots[s.LimineSnapshotIdx], true
}

// LimineConfigRow describes one editable row in a limine config
// sub-section. Label is the human-readable form, Key is the on-disk
// key name, and Value reads from whichever snapshot field holds the
// current value. Kept as a flat list (no pointer into the snapshot)
// so both the view and the action layer can walk the same order.
type LimineConfigRow struct {
	Label string
	Key   string
	Value string
}

// LimineBootConfigRows returns the editable rows for the Boot Config
// sub-section in the order they render. Any row whose value hasn't
// been parsed from /boot/limine.conf is still shown so the user can
// append a new key.
func (s *State) LimineBootConfigRows() []LimineConfigRow {
	c := s.Limine.BootConfig
	return []LimineConfigRow{
		{"timeout", "timeout", c.Timeout},
		{"default_entry", "default_entry", c.DefaultEntry},
		{"interface_branding", "interface_branding", c.InterfaceBrand},
		{"hash_mismatch_panic", "hash_mismatch_panic", c.HashMismatch},
		{"term_background", "term_background", c.TermBackground},
		{"backdrop", "backdrop", c.Backdrop},
		{"term_foreground", "term_foreground", c.TermForeground},
		{"editor_enabled", "editor_enabled", c.Editor},
		{"verbose", "verbose", c.VerboseBoot},
	}
}

func (s *State) LimineSyncConfigRows() []LimineConfigRow {
	c := s.Limine.SyncConfig
	return []LimineConfigRow{
		{"target_os_name", "TARGET_OS_NAME", c.TargetOSName},
		{"esp_path", "ESP_PATH", c.ESPPath},
		{"limit_usage_percent", "LIMIT_USAGE_PERCENT", c.LimitUsagePercent},
		{"max_snapshot_entries", "MAX_SNAPSHOT_ENTRIES", c.MaxSnapshotEntries},
		{"exclude_snapshot_types", "EXCLUDE_SNAPSHOT_TYPES", c.ExcludeSnapshotTypes},
		{"enable_notification", "ENABLE_NOTIFICATION", c.EnableNotification},
		{"snapshot_format_choice", "SNAPSHOT_FORMAT_CHOICE", c.SnapshotFormat},
	}
}

// LimineOmarchyConfigRows returns the scalar editable rows in
// /etc/default/limine. KERNEL_CMDLINE lives as a virtual row at the
// end of the list and uses its own edit flow because it's a multi-
// line block, not a single key.
func (s *State) LimineOmarchyConfigRows() []LimineConfigRow {
	c := s.Limine.OmarchyConfig
	return []LimineConfigRow{
		{"target_os_name", "TARGET_OS_NAME", c.TargetOSName},
		{"esp_path", "ESP_PATH", c.ESPPath},
		{"enable_uki", "ENABLE_UKI", c.EnableUKI},
		{"custom_uki_name", "CUSTOM_UKI_NAME", c.CustomUKIName},
		{"enable_limine_fallback", "ENABLE_LIMINE_FALLBACK", c.EnableLimineFallback},
		{"find_bootloaders", "FIND_BOOTLOADERS", c.FindBootloaders},
		{"boot_order", "BOOT_ORDER", c.BootOrder},
		{"max_snapshot_entries", "MAX_SNAPSHOT_ENTRIES", c.MaxSnapshotEntries},
		{"snapshot_format_choice", "SNAPSHOT_FORMAT_CHOICE", c.SnapshotFormat},
	}
}

// SelectedLimineConfigRow returns the currently-focused config row
// for the active sub-section. The second return value is a kind
// string ("boot" / "sync" / "omarchy" / "omarchy_cmdline") the
// action layer uses to pick the right NATS subject.
func (s *State) SelectedLimineConfigRow() (LimineConfigRow, string, bool) {
	switch s.ActiveLimineSection().ID {
	case "boot":
		rows := s.LimineBootConfigRows()
		if len(rows) == 0 {
			return LimineConfigRow{}, "", false
		}
		if s.LimineBootCfgIdx >= len(rows) {
			s.LimineBootCfgIdx = 0
		}
		return rows[s.LimineBootCfgIdx], "boot", true
	case "sync":
		rows := s.LimineSyncConfigRows()
		if len(rows) == 0 {
			return LimineConfigRow{}, "", false
		}
		if s.LimineSyncCfgIdx >= len(rows) {
			s.LimineSyncCfgIdx = 0
		}
		return rows[s.LimineSyncCfgIdx], "sync", true
	case "omarchy":
		rows := s.LimineOmarchyConfigRows()
		if s.LimineOmarchyCfgIdx > len(rows) {
			s.LimineOmarchyCfgIdx = 0
		}
		if s.LimineOmarchyCfgIdx == len(rows) {
			return LimineConfigRow{Label: "kernel_cmdline", Key: "KERNEL_CMDLINE"}, "omarchy_cmdline", true
		}
		return rows[s.LimineOmarchyCfgIdx], "omarchy", true
	}
	return LimineConfigRow{}, "", false
}

func (s *State) moveLimineFocus(delta int) {
	if !s.LimineContentFocused {
		s.MoveLimineSection(delta)
		return
	}
	switch s.ActiveLimineSection().ID {
	case "snapshots":
		s.MoveLimineSnapshot(delta)
	case "boot":
		moveRowIdx(&s.LimineBootCfgIdx, len(s.LimineBootConfigRows()), delta)
	case "sync":
		moveRowIdx(&s.LimineSyncCfgIdx, len(s.LimineSyncConfigRows()), delta)
	case "omarchy":
		// +1 for the virtual kernel_cmdline row.
		moveRowIdx(&s.LimineOmarchyCfgIdx, len(s.LimineOmarchyConfigRows())+1, delta)
	}
}

func moveRowIdx(idx *int, n, delta int) {
	if n <= 0 {
		return
	}
	*idx = (*idx + delta + n) % n
}
