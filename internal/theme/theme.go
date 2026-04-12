package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const themeColorsPath = ".config/omarchy/current/theme/colors.toml"

// Palette holds the colors dark draws from. Everything is sourced from
// Omarchy's current theme (~/.config/omarchy/current/theme/colors.toml);
// hardcoded values are only used as a last-resort fallback.
type Palette struct {
	Background     string // overall app bg
	HelpBackground string // derived, slightly darker than Background
	Foreground     string // body text
	Dim            string // readable-but-subdued text (labels, hints)
	Accent         string // highlights, active states, heading text
	Muted          string // borders, separators — too dim for body text
	Green          string // success, connected, healthy states
	Gold           string // inline code, warnings
	Red            string // errors, disconnected, broken states
	Cursor         string // selection / input markers
}

// Load reads the current Omarchy theme palette. On any error it returns the
// fallback palette without failing startup.
func Load() Palette {
	path, err := omarchyColorsPath()
	if err != nil {
		return defaults()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return defaults()
	}

	kv := parseKV(data)
	p := defaults()

	if v := kv["background"]; v != "" {
		p.Background = v
	}
	if v := kv["foreground"]; v != "" {
		p.Foreground = v
	}
	if v := kv["accent"]; v != "" {
		p.Accent = v
	}
	if v := kv["cursor"]; v != "" {
		p.Cursor = v
	}
	// color7 is the regular "white" ANSI slot, which in most dark themes
	// is a dimmed-but-readable gray used for labels and secondary text.
	if v := kv["color7"]; v != "" {
		p.Dim = v
	}
	// color8 is bright-black, the much darker shade that themes use for
	// borders, inactive separators, and anything that should barely be
	// visible.
	if v := kv["color8"]; v != "" {
		p.Muted = v
	}
	if v := kv["color2"]; v != "" {
		p.Green = v
	}
	if v := kv["color3"]; v != "" {
		p.Gold = v
	}
	if v := kv["color1"]; v != "" {
		p.Red = v
	}

	p.HelpBackground = darken(p.Background, 0x0a)
	return p
}

func omarchyColorsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, themeColorsPath), nil
}

// parseKV parses the flat `key = "value"` lines Omarchy uses in colors.toml.
// We don't pull in a full TOML dependency because the file is intentionally
// simple and the format is stable.
func parseKV(data []byte) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		val = strings.Trim(val, "\"'")
		out[key] = val
	}
	return out
}

// darken returns hex shifted darker by `amount` per RGB channel, clamped to
// zero. Input is expected to be "#rrggbb"; malformed input returns the
// original string unchanged.
func darken(hex string, amount int) string {
	if len(hex) != 7 || hex[0] != '#' {
		return hex
	}
	r, err1 := strconv.ParseUint(hex[1:3], 16, 8)
	g, err2 := strconv.ParseUint(hex[3:5], 16, 8)
	b, err3 := strconv.ParseUint(hex[5:7], 16, 8)
	if err1 != nil || err2 != nil || err3 != nil {
		return hex
	}
	return fmt.Sprintf("#%02x%02x%02x", clamp(int(r)-amount), clamp(int(g)-amount), clamp(int(b)-amount))
}

func clamp(v int) int {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return v
}

// BgEscape returns the truecolor SGR background escape for a hex color.
// Used by the help layer to reapply backgrounds after ANSI resets.
func BgEscape(hex string) string {
	if len(hex) != 7 || hex[0] != '#' {
		return "\x1b[49m"
	}
	r, _ := strconv.ParseUint(hex[1:3], 16, 8)
	g, _ := strconv.ParseUint(hex[3:5], 16, 8)
	b, _ := strconv.ParseUint(hex[5:7], 16, 8)
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

func defaults() Palette {
	bg := "#1a1a1a"
	return Palette{
		Background:     bg,
		HelpBackground: darken(bg, 0x0a),
		Foreground:     "#e6e6e6",
		Dim:            "#9aa0b3",
		Accent:         "#7aa2f7",
		Muted:          "#444b6a",
		Green:          "#9ece6a",
		Gold:           "#e0af68",
		Red:            "#f7768e",
		Cursor:         "#c0caf5",
	}
}
