package display

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const monitorsConfRelPath = ".config/hypr/monitors.conf"

func monitorsConfPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, monitorsConfRelPath)
}

// PersistMonitor writes or updates a monitor line in monitors.conf for
// the given monitor state. Preserves comments and other content in the
// file. If a line for this monitor name already exists, it's replaced;
// otherwise a new line is appended.
func PersistMonitor(mon Monitor) error {
	path := monitorsConfPath()
	if path == "" {
		return fmt.Errorf("cannot determine home directory")
	}

	line := formatMonitorLine(mon)

	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var result []string
	found := false
	prefix := monitorLinePrefix(mon.Name)

	if len(existing) > 0 {
		scanner := bufio.NewScanner(strings.NewReader(string(existing)))
		for scanner.Scan() {
			text := scanner.Text()
			trimmed := strings.TrimSpace(text)
			if !strings.HasPrefix(trimmed, "#") && matchesMonitorLine(trimmed, prefix) {
				result = append(result, line)
				found = true
			} else {
				result = append(result, text)
			}
		}
	}

	if !found {
		if len(result) > 0 && result[len(result)-1] != "" {
			result = append(result, "")
		}
		result = append(result, line)
	}

	content := strings.Join(result, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}

	return os.WriteFile(path, []byte(content), 0644)
}

func formatMonitorLine(mon Monitor) string {
	res := fmt.Sprintf("%dx%d@%.2f", mon.Width, mon.Height, mon.RefreshRate)
	pos := fmt.Sprintf("%dx%d", mon.X, mon.Y)
	scale := fmt.Sprintf("%.2f", mon.Scale)

	line := fmt.Sprintf("monitor = %s, %s, %s, %s", mon.Name, res, pos, scale)

	if mon.Transform != 0 {
		line += fmt.Sprintf(", transform, %d", mon.Transform)
	}
	if mon.Vrr {
		line += ", vrr, 1"
	}

	return line
}

func monitorLinePrefix(name string) string {
	return "monitor=" + name + ","
}

func matchesMonitorLine(line, prefix string) bool {
	normalized := strings.ReplaceAll(line, " ", "")
	normalizedPrefix := strings.ReplaceAll(prefix, " ", "")
	return strings.HasPrefix(normalized, normalizedPrefix)
}
