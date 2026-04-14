// Package limine exposes the Limine bootloader and the snapper-driven
// boot snapshots Omarchy maintains for it. The bootloader itself is a
// static binary; the dynamic state dark cares about is the set of
// Btrfs snapshots that limine-snapper-sync has wired into the boot
// menu, plus the configuration files Omarchy uses to drive that sync.
//
// Read access uses the world-readable /boot/limine.conf as the source
// of truth for "what the boot menu actually looks like" — snapper's
// own DBus interface refuses unprivileged callers on default Omarchy
// installs, and /.snapshots/ is locked down to root. Write actions
// (create snapshot, sync to limine, delete snapshot, set default) all
// shell out via pkexec so polkit can prompt the user once per call.
package limine

// Snapshot is the limine domain payload published on the bus.
type Snapshot struct {
	Backend       string         `json:"backend"`
	Available     bool           `json:"available"`
	Error         string         `json:"error,omitempty"`
	Snapshots     []BootSnapshot `json:"snapshots,omitempty"`
	DefaultEntry  int            `json:"default_entry"`
	BootConfig    BootConfig     `json:"boot_config"`
	SyncConfig    SyncConfig     `json:"sync_config"`
	OmarchyConfig OmarchyConfig  `json:"omarchy_config"`
}

// BootSnapshot is one snapshot entry as it appears under the
// "Snapshots" submenu in /boot/limine.conf. The kernel version comes
// straight from the limine entry comment and the subvol number is
// parsed out of the rootflags path.
type BootSnapshot struct {
	// Number is the snapper snapshot id (e.g. 98 from /.snapshots/98/snapshot).
	Number int `json:"number"`
	// Timestamp is the human-friendly label limine-snapper-sync writes
	// (e.g. "2026-04-14 15:00:02"). Kept as a string because it is what
	// the user sees in the boot menu.
	Timestamp string `json:"timestamp"`
	// Type is the snapper snapshot type comment, typically "timeline",
	// "pre", "post", or "single".
	Type string `json:"type,omitempty"`
	// Kernel is the kernel-id comment value (e.g. "linux").
	Kernel string `json:"kernel,omitempty"`
	// Subvol is the rootflags subvol path used by this entry.
	Subvol string `json:"subvol,omitempty"`
}

// BootConfig captures the keys we surface from /boot/limine.conf.
// Limine's config language has many more keys; this is the curated
// subset that the F3 limine view shows. Editing these is left as a
// follow-up — Omarchy ships templates that get regenerated on update.
type BootConfig struct {
	Path             string `json:"path"`
	Timeout          string `json:"timeout,omitempty"`
	DefaultEntry     string `json:"default_entry,omitempty"`
	InterfaceBrand   string `json:"interface_branding,omitempty"`
	HashMismatch     string `json:"hash_mismatch_panic,omitempty"`
	TermBackground   string `json:"term_background,omitempty"`
	Backdrop         string `json:"backdrop,omitempty"`
	TermForeground   string `json:"term_foreground,omitempty"`
	Editor           string `json:"editor_enabled,omitempty"`
	VerboseBoot      string `json:"verbose_boot,omitempty"`
}

// SyncConfig captures the keys we surface from /etc/limine-snapper-sync.conf.
type SyncConfig struct {
	Path                string `json:"path"`
	TargetOSName        string `json:"target_os_name,omitempty"`
	ESPPath             string `json:"esp_path,omitempty"`
	LimitUsagePercent   string `json:"limit_usage_percent,omitempty"`
	MaxSnapshotEntries  string `json:"max_snapshot_entries,omitempty"`
	ExcludeSnapshotTypes string `json:"exclude_snapshot_types,omitempty"`
	EnableNotification  string `json:"enable_notification,omitempty"`
	SnapshotFormat      string `json:"snapshot_format_choice,omitempty"`
}

// OmarchyConfig captures the keys we surface from /etc/default/limine
// (the source-of-truth for Omarchy's UKI build, which limine-mkinitcpio-hook
// reads on every kernel update).
type OmarchyConfig struct {
	Path                string   `json:"path"`
	TargetOSName        string   `json:"target_os_name,omitempty"`
	ESPPath             string   `json:"esp_path,omitempty"`
	EnableUKI           string   `json:"enable_uki,omitempty"`
	CustomUKIName       string   `json:"custom_uki_name,omitempty"`
	EnableLimineFallback string  `json:"enable_limine_fallback,omitempty"`
	FindBootloaders     string   `json:"find_bootloaders,omitempty"`
	BootOrder           string   `json:"boot_order,omitempty"`
	MaxSnapshotEntries  string   `json:"max_snapshot_entries,omitempty"`
	SnapshotFormat      string   `json:"snapshot_format_choice,omitempty"`
	KernelCmdline       []string `json:"kernel_cmdline,omitempty"`
}

// Backend identifiers.
const (
	BackendNone   = "none"
	BackendLimine = "limine"
)

// Service owns the chosen Backend and is the single entry point the
// daemon uses to read or mutate limine state.
type Service struct {
	backend Backend
}

// NewService probes for limine on the host and returns a Service
// wired to the right backend. Returns a noop-backed service when
// limine isn't installed so the rest of the daemon can keep running.
func NewService() (*Service, error) {
	backend, err := newLimineBackend()
	if err != nil {
		return &Service{backend: newNoopBackend()}, err
	}
	return &Service{backend: backend}, nil
}

func (s *Service) Close() {
	if s.backend != nil {
		s.backend.Close()
		s.backend = nil
	}
}

func (s *Service) Snapshot() Snapshot {
	if s.backend == nil {
		return Snapshot{Backend: BackendNone}
	}
	return s.backend.Snapshot()
}

// Detect is a one-shot convenience used by the daemon snapshot reply
// when the long-lived Service couldn't be built.
func Detect() Snapshot {
	svc, err := NewService()
	if err != nil {
		return Snapshot{Backend: BackendNone, Error: err.Error()}
	}
	defer svc.Close()
	return svc.Snapshot()
}

func (s *Service) CreateSnapshot(description string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.CreateSnapshot(description)
}

func (s *Service) DeleteSnapshot(number int) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.DeleteSnapshot(number)
}

func (s *Service) Sync() error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.Sync()
}

func (s *Service) SetDefaultEntry(entry int) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetDefaultEntry(entry)
}

func (s *Service) SetBootConfig(key, value string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetBootConfig(key, value)
}

func (s *Service) SetSyncConfig(key, value string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetSyncConfig(key, value)
}

func (s *Service) SetOmarchyConfig(key, value string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetOmarchyConfig(key, value)
}

func (s *Service) SetOmarchyKernelCmdline(lines []string) error {
	if s.backend == nil {
		return ErrBackendUnsupported
	}
	return s.backend.SetOmarchyKernelCmdline(lines)
}
