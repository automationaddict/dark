package appearance

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// readFontFamilies uses fc-list to discover monospace fonts, matching
// the omarchy-font-list script. Falls back to directory scanning if
// fc-list is unavailable.
func readFontFamilies() []string {
	out, err := exec.Command("fc-list", ":spacing=100", "-f", "%{family[0]}\n").Output()
	if err != nil {
		return readFontFallback()
	}
	seen := map[string]bool{}
	var families []string
	for _, line := range strings.Split(string(out), "\n") {
		name := strings.TrimSpace(line)
		if name == "" || seen[name] {
			continue
		}
		lower := strings.ToLower(name)
		if strings.Contains(lower, "emoji") ||
			strings.Contains(lower, "signwriting") ||
			strings.Contains(lower, "omarchy") {
			continue
		}
		seen[name] = true
		families = append(families, name)
	}
	sort.Strings(families)
	return families
}

func readFontFallback() []string {
	seen := map[string]bool{}
	var families []string
	dirs := []string{"/usr/share/fonts", "/usr/local/share/fonts"}
	home, _ := os.UserHomeDir()
	if home != "" {
		dirs = append(dirs, filepath.Join(home, ".local/share/fonts"))
	}
	for _, base := range dirs {
		entries, _ := os.ReadDir(base)
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			subEntries, _ := os.ReadDir(filepath.Join(base, e.Name()))
			for _, sub := range subEntries {
				if sub.IsDir() && !seen[sub.Name()] {
					families = append(families, sub.Name())
					seen[sub.Name()] = true
				}
			}
		}
	}
	sort.Strings(families)
	return families
}

// readCurrentFont extracts the active monospace font from waybar's CSS.
func readCurrentFont(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".config/waybar/style.css"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.Contains(line, "font-family") {
			continue
		}
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		val := strings.TrimSpace(line[idx+1:])
		val = strings.TrimSuffix(val, ";")
		val = strings.Trim(val, "\"' ")
		if val != "" {
			return val
		}
	}
	return ""
}

// readCurrentFontSize extracts the terminal font size from ghostty,
// alacritty, or kitty config (first found wins).
func readCurrentFontSize(home string) int {
	configs := []struct {
		path   string
		prefix string
	}{
		{filepath.Join(home, ".config/ghostty/config"), "font-size"},
		{filepath.Join(home, ".config/alacritty/alacritty.toml"), "size"},
		{filepath.Join(home, ".config/kitty/kitty.conf"), "font_size"},
	}
	for _, c := range configs {
		data, err := os.ReadFile(c.path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, c.prefix) {
				continue
			}
			rest := strings.TrimPrefix(trimmed, c.prefix)
			rest = strings.TrimLeft(rest, " =")
			rest = strings.TrimSpace(rest)
			if v := atoi(rest); v > 0 {
				return v
			}
			if f := atof(rest); f > 0 {
				return int(f)
			}
		}
	}
	return 0
}

// SetFontSize updates the font size across terminal configs.
func SetFontSize(size int) error {
	if size < 4 || size > 72 {
		return fmt.Errorf("font size %d out of range (4-72)", size)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sizeStr := fmt.Sprintf("%d", size)
	sizeFloat := fmt.Sprintf("%.1f", float64(size))

	sedFile(filepath.Join(home, ".config/ghostty/config"),
		"font-size = ", "\n", sizeStr)
	sedFileKV(filepath.Join(home, ".config/ghostty/config"),
		"font-size = ", sizeStr)
	signalProcess("ghostty", "-SIGUSR2")

	sedFileKV(filepath.Join(home, ".config/alacritty/alacritty.toml"),
		"size = ", sizeStr)

	sedFilePrefix(filepath.Join(home, ".config/kitty/kitty.conf"),
		"font_size ", sizeFloat)
	signalProcess("kitty", "-USR1")

	return nil
}

// SetFont updates the monospace font across all terminal and UI configs,
// mirroring the omarchy-font-set script.
func SetFont(name string) error {
	if name == "" {
		return fmt.Errorf("empty font name")
	}
	out, err := exec.Command("fc-list", name).Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return fmt.Errorf("font %q not found", name)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	sedFile(filepath.Join(home, ".config/alacritty/alacritty.toml"),
		`family = "`, `"`, name)

	if sedFilePrefix(filepath.Join(home, ".config/kitty/kitty.conf"),
		"font_family ", name) {
		signalProcess("kitty", "-USR1")
	}

	if sedFile(filepath.Join(home, ".config/ghostty/config"),
		`font-family = "`, `"`, name) {
		signalProcess("ghostty", "-SIGUSR2")
	}

	sedFileKV(filepath.Join(home, ".config/hypr/hyprlock.conf"),
		"font_family = ", name)

	sedFileCSS(filepath.Join(home, ".config/waybar/style.css"),
		"font-family:", name)

	sedFileCSS(filepath.Join(home, ".config/swayosd/style.css"),
		"font-family:", name)

	setFontconfigMonospace(filepath.Join(home, ".config/fontconfig/fonts.conf"), name)

	exec.Command("omarchy-restart-waybar").Run()
	exec.Command("omarchy-restart-swayosd").Run()
	exec.Command("omarchy-hook", "font-set", name).Run()

	return nil
}

// sedFile replaces a value between prefix and suffix in a file.
func sedFile(path, prefix, suffix, value string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		idx := strings.Index(line, prefix)
		if idx < 0 {
			continue
		}
		before := line[:idx+len(prefix)]
		rest := line[idx+len(prefix):]
		end := strings.Index(rest, suffix)
		if end < 0 {
			continue
		}
		after := rest[end:]
		lines[i] = before + value + after
		changed = true
	}
	if changed {
		os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
	}
	return changed
}

// sedFilePrefix replaces lines starting with prefix.
func sedFilePrefix(path, prefix, value string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + prefix + value
			changed = true
		}
	}
	if changed {
		os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
	}
	return changed
}

// sedFileKV replaces key = value lines.
func sedFileKV(path, prefix, value string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prefix) {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + prefix + value
			changed = true
		}
	}
	if changed {
		os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
	}
}

// sedFileCSS replaces CSS font-family declarations.
func sedFileCSS(path, prop, value string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	changed := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, prop) {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			lines[i] = indent + prop + " '" + value + "';"
			changed = true
		}
	}
	if changed {
		os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644)
	}
}

// setFontconfigMonospace updates the monospace alias in fonts.conf.
func setFontconfigMonospace(path, value string) {
	exec.Command("xmlstarlet", "ed", "-L",
		"-u", `//match[@target="pattern"][test/string="monospace"]/edit[@name="family"]/string`,
		"-v", value, path).Run()
}

func signalProcess(name, signal string) {
	exec.Command("pkill", signal, name).Run()
}
