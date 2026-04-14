package limine

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// limineBackend is the real backend. It reads the boot menu state from
// /boot/limine.conf (world-readable) and shells out via pkexec for any
// privileged write action. Polkit handles the prompt in the user's
// session.
type limineBackend struct{}

func newLimineBackend() (*limineBackend, error) {
	if _, err := exec.LookPath("limine"); err != nil {
		return nil, fmt.Errorf("limine binary not found: %w", err)
	}
	return &limineBackend{}, nil
}

func (b *limineBackend) Name() string { return BackendLimine }

func (b *limineBackend) Close() {}

const (
	limineConfPath      = "/boot/limine.conf"
	limineDefaultsPath  = "/etc/default/limine"
	limineSnapperPath   = "/etc/limine-snapper-sync.conf"
)

func (b *limineBackend) Snapshot() Snapshot {
	snap := Snapshot{
		Backend:       BackendLimine,
		Available:     true,
		BootConfig:    BootConfig{Path: limineConfPath},
		SyncConfig:    loadSyncConfig(limineSnapperPath),
		OmarchyConfig: loadOmarchyConfig(limineDefaultsPath),
	}
	cfg, snaps, err := parseLimineConf(limineConfPath)
	if err != nil {
		snap.Error = err.Error()
		return snap
	}
	snap.BootConfig = cfg
	snap.Snapshots = snaps
	if cfg.DefaultEntry != "" {
		if n, perr := strconv.Atoi(cfg.DefaultEntry); perr == nil {
			snap.DefaultEntry = n
		}
	}
	return snap
}

// runPkexec wraps a command that needs root with pkexec so a polkit
// agent can prompt the user. Captures combined output so the caller
// can surface errors back to the TUI.
func runPkexec(name string, args ...string) error {
	if _, err := exec.LookPath("pkexec"); err != nil {
		return fmt.Errorf("pkexec not available: %w", err)
	}
	full := append([]string{name}, args...)
	cmd := exec.Command("pkexec", full...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return err
		}
		return fmt.Errorf("%s: %s", err.Error(), msg)
	}
	return nil
}

func (b *limineBackend) CreateSnapshot(description string) error {
	desc := strings.TrimSpace(description)
	if desc == "" {
		desc = "dark manual snapshot"
	}
	return runPkexec("snapper", "-c", "root", "create",
		"--type", "single",
		"--cleanup-algorithm", "number",
		"--description", desc)
}

func (b *limineBackend) DeleteSnapshot(number int) error {
	if number <= 0 {
		return fmt.Errorf("invalid snapshot number")
	}
	if err := runPkexec("snapper", "-c", "root", "delete", strconv.Itoa(number)); err != nil {
		return err
	}
	return b.Sync()
}

func (b *limineBackend) Sync() error {
	if _, err := exec.LookPath("limine-snapper-sync"); err != nil {
		return fmt.Errorf("limine-snapper-sync not installed: %w", err)
	}
	return runPkexec("limine-snapper-sync")
}

// allowedBootConfigKeys is the whitelist of top-level limine.conf keys
// the TUI is allowed to rewrite. Anything else is rejected so a caller
// can't accidentally clobber a value inside an auto-generated entry
// block (rewriteBootConfKey already protects entries, but belt +
// braces here).
var allowedBootConfigKeys = map[string]bool{
	"timeout":              true,
	"default_entry":        true,
	"interface_branding":   true,
	"hash_mismatch_panic":  true,
	"term_background":      true,
	"backdrop":              true,
	"term_foreground":      true,
	"editor_enabled":       true,
	"verbose":              true,
}

var allowedSyncConfigKeys = map[string]bool{
	"TARGET_OS_NAME":         true,
	"ESP_PATH":               true,
	"LIMIT_USAGE_PERCENT":    true,
	"MAX_SNAPSHOT_ENTRIES":   true,
	"EXCLUDE_SNAPSHOT_TYPES": true,
	"ENABLE_NOTIFICATION":    true,
	"SNAPSHOT_FORMAT_CHOICE": true,
}

var allowedOmarchyConfigKeys = map[string]bool{
	"TARGET_OS_NAME":         true,
	"ESP_PATH":               true,
	"ENABLE_UKI":             true,
	"CUSTOM_UKI_NAME":        true,
	"ENABLE_LIMINE_FALLBACK": true,
	"FIND_BOOTLOADERS":       true,
	"BOOT_ORDER":             true,
	"MAX_SNAPSHOT_ENTRIES":   true,
	"SNAPSHOT_FORMAT_CHOICE": true,
}

func (b *limineBackend) SetDefaultEntry(entry int) error {
	if entry < 0 {
		return fmt.Errorf("default entry must be >= 0")
	}
	return setBootConfKey(limineConfPath, "default_entry", strconv.Itoa(entry))
}

func (b *limineBackend) SetBootConfig(key, value string) error {
	if !allowedBootConfigKeys[key] {
		return fmt.Errorf("boot config key %q is not editable", key)
	}
	return setBootConfKey(limineConfPath, key, value)
}

func (b *limineBackend) SetSyncConfig(key, value string) error {
	if !allowedSyncConfigKeys[key] {
		return fmt.Errorf("sync config key %q is not editable", key)
	}
	return setShellAssignment(limineSnapperPath, key, value)
}

func (b *limineBackend) SetOmarchyConfig(key, value string) error {
	if !allowedOmarchyConfigKeys[key] {
		return fmt.Errorf("omarchy config key %q is not editable", key)
	}
	return setShellAssignment(limineDefaultsPath, key, value)
}

func (b *limineBackend) SetOmarchyKernelCmdline(lines []string) error {
	return setShellCmdline(limineDefaultsPath, lines)
}
