package display

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

// ReadGPUInfo detects GPUs and hybrid GPU state.
func ReadGPUInfo() GPUInfo {
	gpus := detectGPUs()
	info := GPUInfo{
		GPUs:            gpus,
		HybridSupported: len(gpus) >= 2,
	}
	if info.HybridSupported {
		info.Mode = readSupergfxMode()
	}
	return info
}

// detectGPUs reads /proc/bus/pci or lspci output to find GPU devices.
func detectGPUs() []string {
	out, err := exec.Command("lspci").Output()
	if err != nil {
		return nil
	}
	var gpus []string
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		// VGA compatible controller, 3D controller, Display controller
		if strings.Contains(line, "VGA compatible controller") ||
			strings.Contains(line, "3D controller") ||
			strings.Contains(line, "Display controller") {
			// Extract the device name after the colon
			if idx := strings.Index(line, ": "); idx >= 0 {
				gpus = append(gpus, strings.TrimSpace(line[idx+2:]))
			}
		}
	}
	return gpus
}

// readSupergfxMode reads the current GPU mode from supergfxctl or its
// config file. Returns "" if supergfxctl is not installed.
func readSupergfxMode() string {
	// Try the daemon first
	out, err := exec.Command("supergfxctl", "-g").Output()
	if err == nil {
		mode := strings.TrimSpace(string(out))
		if mode != "" {
			return mode
		}
	}
	// Fall back to reading the config file
	data, err := os.ReadFile("/etc/supergfxd.conf")
	if err != nil {
		return ""
	}
	var conf struct {
		Mode string `json:"mode"`
	}
	if err := json.Unmarshal(data, &conf); err != nil {
		return ""
	}
	return conf.Mode
}
