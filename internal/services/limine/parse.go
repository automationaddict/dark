package limine

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// parseLimineConf walks /boot/limine.conf (the user-facing entry list
// produced by limine-entry-tool) and returns a curated BootConfig and
// the list of boot snapshots discovered under the auto-generated
// "Snapshots" submenu. The grammar uses leading slashes for menu
// nesting depth: /entry is depth 1, //child is depth 2, etc.
func parseLimineConf(path string) (BootConfig, []BootSnapshot, error) {
	cfg := BootConfig{Path: path}
	f, err := os.Open(path)
	if err != nil {
		return cfg, nil, err
	}
	defer f.Close()

	var snaps []BootSnapshot
	var current *BootSnapshot
	var nextLabel, nextType, nextKernel string
	// snapshotsDepth is the slash-depth of the //Snapshots entry once
	// we've entered it, or 0 when we are not inside it. Any line at
	// depth <= snapshotsDepth that is not under the Snapshots subtree
	// pops us out.
	snapshotsDepth := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Menu entry headers start with one or more slashes followed
		// by a label. Determine the depth by counting leading slashes.
		if strings.HasPrefix(line, "/") {
			depth := 0
			for depth < len(line) && line[depth] == '/' {
				depth++
			}
			label := strings.TrimSpace(line[depth:])

			if snapshotsDepth > 0 && depth <= snapshotsDepth && label != "Snapshots" {
				// Left the Snapshots subtree.
				snapshotsDepth = 0
				current = nil
			}

			if label == "Snapshots" {
				snapshotsDepth = depth
				current = nil
				continue
			}

			if snapshotsDepth == 0 {
				current = nil
				continue
			}

			// Inside the Snapshots subtree.
			if depth == snapshotsDepth+1 {
				// Per-snapshot timestamp label.
				nextLabel = label
				nextType = ""
				nextKernel = ""
				current = nil
				continue
			}
			if depth == snapshotsDepth+2 {
				// Per-kernel entry inside the snapshot.
				snaps = append(snaps, BootSnapshot{
					Timestamp: nextLabel,
					Type:      nextType,
					Kernel:    nextKernel,
				})
				current = &snaps[len(snaps)-1]
				continue
			}
			continue
		}

		// Top-level "key: value" config keys appear before any /entry.
		if snapshotsDepth == 0 && current == nil {
			if k, v, ok := splitColonKV(line); ok {
				switch k {
				case "comment", "protocol", "path", "cmdline":
					// These belong to a menu entry above; ignore.
				default:
					assignBootConfig(&cfg, k, v)
				}
			}
			continue
		}

		if strings.HasPrefix(line, "comment:") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "comment:"))
			if current == nil {
				if val == "timeline" || val == "pre" || val == "post" || val == "single" {
					nextType = val
				} else if strings.HasPrefix(val, "Kernel version:") {
					nextKernel = strings.TrimSpace(strings.TrimPrefix(val, "Kernel version:"))
				} else if strings.HasPrefix(val, "kernel-id=") && nextKernel == "" {
					nextKernel = strings.TrimPrefix(val, "kernel-id=")
				}
				continue
			}
			if strings.HasPrefix(val, "Kernel version:") {
				current.Kernel = strings.TrimSpace(strings.TrimPrefix(val, "Kernel version:"))
			}
			continue
		}

		if strings.HasPrefix(line, "cmdline:") && current != nil {
			subvol, num := extractSubvol(line)
			current.Subvol = subvol
			if num > 0 {
				current.Number = num
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return cfg, snaps, err
	}
	return cfg, snaps, nil
}

func splitColonKV(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}
	k := strings.TrimSpace(line[:idx])
	v := strings.TrimSpace(line[idx+1:])
	return k, v, true
}

func assignBootConfig(cfg *BootConfig, key, value string) {
	switch key {
	case "timeout":
		cfg.Timeout = value
	case "default_entry":
		cfg.DefaultEntry = value
	case "interface_branding":
		cfg.InterfaceBrand = value
	case "hash_mismatch_panic":
		cfg.HashMismatch = value
	case "term_background":
		cfg.TermBackground = value
	case "backdrop":
		cfg.Backdrop = value
	case "term_foreground":
		cfg.TermForeground = value
	case "editor_enabled":
		cfg.Editor = value
	case "verbose":
		cfg.VerboseBoot = value
	}
}

var subvolRe = regexp.MustCompile(`rootflags=subvol=(/@/\.snapshots/(\d+)/snapshot)`)

func extractSubvol(line string) (string, int) {
	m := subvolRe.FindStringSubmatch(line)
	if m == nil {
		return "", 0
	}
	num, _ := strconv.Atoi(m[2])
	return m[1], num
}

// parseShellAssignments parses a shell-style key=value file (like
// /etc/default/limine and /etc/limine-snapper-sync.conf). It strips
// quotes, ignores comments, and concatenates KERNEL_CMDLINE[default]+=
// values into a slice. This is not a real shell parser — it's just
// enough to surface the keys dark renders.
func parseShellAssignments(path string) (map[string]string, []string, error) {
	out := map[string]string{}
	var cmdline []string
	f, err := os.Open(path)
	if err != nil {
		return out, cmdline, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "KERNEL_CMDLINE[default]+=") {
			val := strings.TrimSpace(strings.TrimPrefix(line, "KERNEL_CMDLINE[default]+="))
			cmdline = append(cmdline, dequote(val))
			continue
		}
		idx := strings.Index(line, "=")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := dequote(strings.TrimSpace(line[idx+1:]))
		out[key] = val
	}
	return out, cmdline, scanner.Err()
}

func dequote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func loadOmarchyConfig(path string) OmarchyConfig {
	cfg := OmarchyConfig{Path: path}
	kv, cmdline, err := parseShellAssignments(path)
	if err != nil {
		return cfg
	}
	cfg.TargetOSName = kv["TARGET_OS_NAME"]
	cfg.ESPPath = kv["ESP_PATH"]
	cfg.EnableUKI = kv["ENABLE_UKI"]
	cfg.CustomUKIName = kv["CUSTOM_UKI_NAME"]
	cfg.EnableLimineFallback = kv["ENABLE_LIMINE_FALLBACK"]
	cfg.FindBootloaders = kv["FIND_BOOTLOADERS"]
	cfg.BootOrder = kv["BOOT_ORDER"]
	cfg.MaxSnapshotEntries = kv["MAX_SNAPSHOT_ENTRIES"]
	cfg.SnapshotFormat = kv["SNAPSHOT_FORMAT_CHOICE"]
	cfg.KernelCmdline = cmdline
	return cfg
}

func loadSyncConfig(path string) SyncConfig {
	cfg := SyncConfig{Path: path}
	kv, _, err := parseShellAssignments(path)
	if err != nil {
		return cfg
	}
	cfg.TargetOSName = kv["TARGET_OS_NAME"]
	cfg.ESPPath = kv["ESP_PATH"]
	cfg.LimitUsagePercent = kv["LIMIT_USAGE_PERCENT"]
	cfg.MaxSnapshotEntries = kv["MAX_SNAPSHOT_ENTRIES"]
	cfg.ExcludeSnapshotTypes = kv["EXCLUDE_SNAPSHOT_TYPES"]
	cfg.EnableNotification = kv["ENABLE_NOTIFICATION"]
	cfg.SnapshotFormat = kv["SNAPSHOT_FORMAT_CHOICE"]
	return cfg
}
