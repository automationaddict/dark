package appearance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// SetTheme runs omarchy-theme-set to switch the active theme. This is the
// canonical way to change themes on omarchy — it handles template
// regeneration, config swapping, and component restarts.
func SetTheme(name string) error {
	return exec.Command("omarchy-theme-set", name).Run()
}

// SetGapsIn writes a gaps_in override to the user's looknfeel.conf.
func SetGapsIn(val int) error {
	return setHyprVar("general", "gaps_in", fmt.Sprintf("%d", val))
}

func SetGapsOut(val int) error {
	return setHyprVar("general", "gaps_out", fmt.Sprintf("%d", val))
}

func SetBorderSize(val int) error {
	return setHyprVar("general", "border_size", fmt.Sprintf("%d", val))
}

func SetRounding(val int) error {
	return setHyprVar("decoration", "rounding", fmt.Sprintf("%d", val))
}

func SetBlurEnabled(enabled bool) error {
	v := "false"
	if enabled {
		v = "true"
	}
	return setHyprVar("decoration.blur", "enabled", v)
}

func SetBlurSize(val int) error {
	return setHyprVar("decoration.blur", "size", fmt.Sprintf("%d", val))
}

func SetBlurPasses(val int) error {
	return setHyprVar("decoration.blur", "passes", fmt.Sprintf("%d", val))
}

func SetAnimEnabled(enabled bool) error {
	v := "no"
	if enabled {
		v = "yes, please :)"
	}
	return setHyprVar("animations", "enabled", v)
}

// setHyprVar patches a value in the user's ~/.config/hypr/looknfeel.conf.
// If the section or key doesn't exist yet, it appends them.
func setHyprVar(section, key, val string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	path := filepath.Join(home, ".config/hypr/looknfeel.conf")

	data, _ := os.ReadFile(path)
	lines := strings.Split(string(data), "\n")

	result, found := patchSection(lines, section, key, val)
	if !found {
		result = appendSection(result, section, key, val)
	}

	return os.WriteFile(path, []byte(strings.Join(result, "\n")), 0o644)
}

// patchSection scans lines for a matching section+key and replaces the value.
func patchSection(lines []string, section, key, val string) ([]string, bool) {
	sectionParts := strings.Split(section, ".")
	depth := 0
	matched := 0
	inTarget := false
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if strings.HasSuffix(trimmed, "{") {
			name := strings.TrimSpace(strings.TrimSuffix(trimmed, "{"))
			depth++
			if matched < len(sectionParts) && name == sectionParts[matched] {
				matched++
				if matched == len(sectionParts) {
					inTarget = true
				}
			}
			continue
		}

		if trimmed == "}" {
			if inTarget && matched == len(sectionParts) {
				inTarget = false
			}
			depth--
			if depth < matched {
				matched = depth
			}
			continue
		}

		if !inTarget {
			continue
		}

		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])

		// Handle commented-out version of the key
		if strings.HasPrefix(k, "# ") {
			k = strings.TrimPrefix(k, "# ")
		}
		if k == key {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + key + " = " + val
			found = true
			break
		}
	}

	return lines, found
}

// appendSection adds a new section+key to the end of the config.
func appendSection(lines []string, section, key, val string) []string {
	parts := strings.Split(section, ".")
	var block []string
	indent := ""
	for _, p := range parts {
		block = append(block, indent+p+" {")
		indent += "    "
	}
	block = append(block, indent+key+" = "+val)
	for i := len(parts) - 1; i >= 0; i-- {
		block = append(block, strings.Repeat("    ", i)+"}")
	}
	lines = append(lines, "")
	lines = append(lines, block...)
	return lines
}
