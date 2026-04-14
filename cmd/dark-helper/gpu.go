package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// setGPUMode switches between Hybrid and Integrated GPU modes via
// supergfxd config, mirroring omarchy-toggle-hybrid-gpu.
func setGPUMode(mode string) error {
	confPath := "/etc/supergfxd.conf"
	omarchyPath := os.Getenv("OMARCHY_PATH")
	if omarchyPath == "" {
		home, _ := os.UserHomeDir()
		omarchyPath = filepath.Join(home, ".local", "share", "omarchy")
	}

	switch mode {
	case "hybrid":
		if err := sedJSON(confPath, "mode", "Hybrid"); err != nil {
			return fmt.Errorf("set mode: %w", err)
		}
		// Remove sleep hook and startup delay
		os.Remove("/usr/lib/systemd/system-sleep/force-igpu")
		os.Remove("/etc/systemd/system/supergfxd.service.d/delay-start.conf")
		return nil

	case "integrated":
		if err := sedJSON(confPath, "mode", "Integrated"); err != nil {
			return fmt.Errorf("set mode: %w", err)
		}
		sedJSON(confPath, "vfio_enable", "true")
		// Install sleep hook
		src := filepath.Join(omarchyPath, "default/systemd/system-sleep/force-igpu")
		dst := "/usr/lib/systemd/system-sleep/force-igpu"
		if data, err := os.ReadFile(src); err == nil {
			os.MkdirAll(filepath.Dir(dst), 0o755)
			os.WriteFile(dst, data, 0o755)
		}
		// Install startup delay
		delaySrc := filepath.Join(omarchyPath, "default/systemd/system/supergfxd.service.d/delay-start.conf")
		delayDst := "/etc/systemd/system/supergfxd.service.d/delay-start.conf"
		if data, err := os.ReadFile(delaySrc); err == nil {
			os.MkdirAll(filepath.Dir(delayDst), 0o755)
			os.WriteFile(delayDst, data, 0o644)
		}
		return nil

	default:
		return fmt.Errorf("mode must be hybrid or integrated, got %q", mode)
	}
}

// sedJSON does a simple string replacement in a JSON config file for a
// key's value. This is intentionally not a full JSON parse-and-rewrite
// to preserve formatting and comments.
func sedJSON(path, key, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		prefix := `"` + key + `"`
		if strings.HasPrefix(trimmed, prefix) {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			if value == "true" || value == "false" {
				lines[i] = indent + `"` + key + `": ` + value + ","
			} else {
				lines[i] = indent + `"` + key + `": "` + value + `",`
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("key %q not found in %s", key, path)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
}
