package appearance

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Snapshot captures the full appearance state: theme, colors, Hyprland
// decoration/layout values, available themes, icon themes, cursor themes,
// and font families. Everything is read from config files and sysfs — no
// shell-outs required.
type Snapshot struct {
	Theme           string   `json:"theme"`
	Themes          []string `json:"themes"`
	IconTheme       string   `json:"icon_theme"`
	IconThemes      []string `json:"icon_themes"`
	CursorTheme     string   `json:"cursor_theme"`
	CursorThemes    []string `json:"cursor_themes"`
	CursorSize      int      `json:"cursor_size"`
	KeyboardRGB     string   `json:"keyboard_rgb"`
	Fonts           []string `json:"fonts"`
	Backgrounds     []string `json:"backgrounds"`
	Colors          Colors   `json:"colors"`
	General         General  `json:"general"`
	Decoration      Deco     `json:"decoration"`
	Blur            Blur     `json:"blur"`
	Shadow          Shadow   `json:"shadow"`
	Animations      Anim     `json:"animations"`
	Layout          Layout   `json:"layout"`
	Cursor          Cursor   `json:"cursor"`
	Groupbar        Groupbar `json:"groupbar"`
}

type Colors struct {
	Accent              string `json:"accent"`
	Cursor              string `json:"cursor"`
	Foreground          string `json:"foreground"`
	Background          string `json:"background"`
	SelectionForeground string `json:"selection_foreground"`
	SelectionBackground string `json:"selection_background"`
	Color0              string `json:"color0"`
	Color1              string `json:"color1"`
	Color2              string `json:"color2"`
	Color3              string `json:"color3"`
	Color4              string `json:"color4"`
	Color5              string `json:"color5"`
	Color6              string `json:"color6"`
	Color7              string `json:"color7"`
	Color8              string `json:"color8"`
	Color9              string `json:"color9"`
	Color10             string `json:"color10"`
	Color11             string `json:"color11"`
	Color12             string `json:"color12"`
	Color13             string `json:"color13"`
	Color14             string `json:"color14"`
	Color15             string `json:"color15"`
}

type General struct {
	GapsIn       int    `json:"gaps_in"`
	GapsOut      int    `json:"gaps_out"`
	BorderSize   int    `json:"border_size"`
	ActiveBorder string `json:"active_border"`
	InactiveBorder string `json:"inactive_border"`
	ResizeOnBorder bool   `json:"resize_on_border"`
	LayoutName     string `json:"layout"`
}

type Deco struct {
	Rounding    int     `json:"rounding"`
	DimInactive bool    `json:"dim_inactive"`
	DimStrength float64 `json:"dim_strength"`
}

type Blur struct {
	Enabled    bool    `json:"enabled"`
	Size       int     `json:"size"`
	Passes     int     `json:"passes"`
	Brightness float64 `json:"brightness"`
	Contrast   float64 `json:"contrast"`
	Special    bool    `json:"special"`
}

type Shadow struct {
	Enabled     bool   `json:"enabled"`
	Range       int    `json:"range"`
	RenderPower int    `json:"render_power"`
	Color       string `json:"color"`
}

type Anim struct {
	Enabled bool       `json:"enabled"`
	Rules   []AnimRule `json:"rules"`
}

type AnimRule struct {
	Name   string `json:"name"`
	On     bool   `json:"on"`
	Speed  string `json:"speed"`
	Curve  string `json:"curve"`
	Style  string `json:"style,omitempty"`
}

type Layout struct {
	Pseudotile   bool `json:"pseudotile"`
	PreserveSplit bool `json:"preserve_split"`
	ForceSplit   int  `json:"force_split"`
}

type Cursor struct {
	HideOnKeyPress       bool `json:"hide_on_key_press"`
	WarpOnChangeWorkspace int  `json:"warp_on_change_workspace"`
}

type Groupbar struct {
	FontSize   int    `json:"font_size"`
	FontFamily string `json:"font_family"`
	Height     int    `json:"height"`
	GapsIn     int    `json:"gaps_in"`
	Gradients  bool   `json:"gradients"`
}

func ReadSnapshot() Snapshot {
	home, _ := os.UserHomeDir()

	s := Snapshot{
		// defaults matching omarchy
		General: General{
			GapsIn: 5, GapsOut: 10, BorderSize: 2, LayoutName: "dwindle",
		},
		Decoration: Deco{DimStrength: 0.15},
		Blur:       Blur{Enabled: true, Size: 2, Passes: 2, Brightness: 0.60, Contrast: 0.75, Special: true},
		Shadow:     Shadow{Enabled: true, Range: 2, RenderPower: 3},
		Animations: Anim{Enabled: true},
		Layout:     Layout{Pseudotile: true, PreserveSplit: true, ForceSplit: 2},
		Cursor:     Cursor{HideOnKeyPress: true, WarpOnChangeWorkspace: 1},
		Groupbar:   Groupbar{FontSize: 12, FontFamily: "monospace", Height: 22, GapsIn: 5, Gradients: true},
		CursorSize: 24,
	}

	s.Theme = readThemeName(home)
	s.Themes = readAvailableThemes(home)
	s.Colors = readColors(home)
	s.IconTheme = readIconTheme(home)
	s.IconThemes = readIconThemes()
	s.CursorTheme, s.CursorThemes = readCursorThemes()
	s.KeyboardRGB = readKeyboardRGB(home)
	s.Fonts = readFontFamilies()
	s.Backgrounds = readBackgrounds(home)

	// Parse omarchy defaults first, then user overrides on top.
	defaultConf := filepath.Join(home, ".local/share/omarchy/default/hypr/looknfeel.conf")
	userConf := filepath.Join(home, ".config/hypr/looknfeel.conf")
	parseHyprlandConfig(&s, defaultConf)
	parseHyprlandConfig(&s, userConf)

	return s
}

func readThemeName(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".config/omarchy/current/theme.name"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readAvailableThemes(home string) []string {
	var themes []string
	seen := map[string]bool{}

	for _, dir := range []string{
		filepath.Join(home, ".local/share/omarchy/themes"),
		filepath.Join(home, ".config/omarchy/themes"),
	} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() && !seen[e.Name()] {
				themes = append(themes, e.Name())
				seen[e.Name()] = true
			}
		}
	}

	sort.Strings(themes)
	return themes
}

func readColors(home string) Colors {
	data, err := os.ReadFile(filepath.Join(home, ".config/omarchy/current/theme/colors.toml"))
	if err != nil {
		return Colors{}
	}
	kv := parseKV(data)
	return Colors{
		Accent:              kv["accent"],
		Cursor:              kv["cursor"],
		Foreground:          kv["foreground"],
		Background:          kv["background"],
		SelectionForeground: kv["selection_foreground"],
		SelectionBackground: kv["selection_background"],
		Color0:              kv["color0"],
		Color1:              kv["color1"],
		Color2:              kv["color2"],
		Color3:              kv["color3"],
		Color4:              kv["color4"],
		Color5:              kv["color5"],
		Color6:              kv["color6"],
		Color7:              kv["color7"],
		Color8:              kv["color8"],
		Color9:              kv["color9"],
		Color10:             kv["color10"],
		Color11:             kv["color11"],
		Color12:             kv["color12"],
		Color13:             kv["color13"],
		Color14:             kv["color14"],
		Color15:             kv["color15"],
	}
}

func readIconTheme(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".config/omarchy/current/theme/icons.theme"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readIconThemes() []string {
	entries, _ := os.ReadDir("/usr/share/icons")
	var themes []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "default" || name == "hicolor" || strings.HasPrefix(name, ".") {
			continue
		}
		themes = append(themes, name)
	}
	sort.Strings(themes)
	return themes
}

func readCursorThemes() (string, []string) {
	var themes []string
	entries, _ := os.ReadDir("/usr/share/icons")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		cursorDir := filepath.Join("/usr/share/icons", e.Name(), "cursors")
		if info, err := os.Stat(cursorDir); err == nil && info.IsDir() {
			themes = append(themes, e.Name())
		}
	}
	sort.Strings(themes)

	// Try to detect current cursor theme from hyprland env or XCURSOR_THEME
	current := os.Getenv("XCURSOR_THEME")
	if current == "" && len(themes) > 0 {
		current = themes[0]
	}
	return current, themes
}

func readKeyboardRGB(home string) string {
	data, err := os.ReadFile(filepath.Join(home, ".config/omarchy/current/theme/keyboard.rgb"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readFontFamilies() []string {
	f, err := os.Open("/etc/fonts/fonts.conf")
	if err != nil {
		return readFontFallback()
	}
	f.Close()
	return readFontFallback()
}

// readFontFallback scans /usr/share/fonts for family directories.
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

func readBackgrounds(home string) []string {
	dir := filepath.Join(home, ".config/omarchy/current/theme/backgrounds")
	entries, _ := os.ReadDir(dir)
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// parseHyprlandConfig reads a Hyprland config file and applies values to
// the snapshot. It handles the nested section syntax (general { ... }).
func parseHyprlandConfig(s *Snapshot, path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var section []string // stack of open sections

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// strip inline comments
		if idx := strings.Index(line, "#"); idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}

		if strings.HasSuffix(line, "{") {
			name := strings.TrimSpace(strings.TrimSuffix(line, "{"))
			section = append(section, name)
			continue
		}
		if line == "}" {
			if len(section) > 0 {
				section = section[:len(section)-1]
			}
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		prefix := strings.Join(section, ".")
		applyHyprValue(s, prefix, key, val)
	}
}

func applyHyprValue(s *Snapshot, section, key, val string) {
	switch section {
	case "general":
		switch key {
		case "gaps_in":
			s.General.GapsIn = atoi(val)
		case "gaps_out":
			s.General.GapsOut = atoi(val)
		case "border_size":
			s.General.BorderSize = atoi(val)
		case "col.active_border":
			s.General.ActiveBorder = val
		case "col.inactive_border":
			s.General.InactiveBorder = val
		case "resize_on_border":
			s.General.ResizeOnBorder = parseBool(val)
		case "layout":
			s.General.LayoutName = val
		}
	case "decoration":
		switch key {
		case "rounding":
			s.Decoration.Rounding = atoi(val)
		case "dim_inactive":
			s.Decoration.DimInactive = parseBool(val)
		case "dim_strength":
			s.Decoration.DimStrength = atof(val)
		}
	case "decoration.blur":
		switch key {
		case "enabled":
			s.Blur.Enabled = parseBool(val)
		case "size":
			s.Blur.Size = atoi(val)
		case "passes":
			s.Blur.Passes = atoi(val)
		case "brightness":
			s.Blur.Brightness = atof(val)
		case "contrast":
			s.Blur.Contrast = atof(val)
		case "special":
			s.Blur.Special = parseBool(val)
		}
	case "decoration.shadow":
		switch key {
		case "enabled":
			s.Shadow.Enabled = parseBool(val)
		case "range":
			s.Shadow.Range = atoi(val)
		case "render_power":
			s.Shadow.RenderPower = atoi(val)
		case "color":
			s.Shadow.Color = val
		}
	case "animations":
		switch key {
		case "enabled":
			s.Animations.Enabled = parseBool(val)
		case "animation":
			s.Animations.Rules = append(s.Animations.Rules, parseAnimRule(val))
		}
	case "dwindle":
		switch key {
		case "pseudotile":
			s.Layout.Pseudotile = parseBool(val)
		case "preserve_split":
			s.Layout.PreserveSplit = parseBool(val)
		case "force_split":
			s.Layout.ForceSplit = atoi(val)
		}
	case "cursor":
		switch key {
		case "hide_on_key_press":
			s.Cursor.HideOnKeyPress = parseBool(val)
		case "warp_on_change_workspace":
			s.Cursor.WarpOnChangeWorkspace = atoi(val)
		}
	case "group.groupbar":
		switch key {
		case "font_size":
			s.Groupbar.FontSize = atoi(val)
		case "font_family":
			s.Groupbar.FontFamily = val
		case "height":
			s.Groupbar.Height = atoi(val)
		case "gaps_in":
			s.Groupbar.GapsIn = atoi(val)
		case "gradients":
			s.Groupbar.Gradients = parseBool(val)
		}
	}
}

func parseAnimRule(val string) AnimRule {
	// animation = name, on, speed, curve[, style]
	parts := strings.Split(val, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	r := AnimRule{}
	if len(parts) >= 1 {
		r.Name = parts[0]
	}
	if len(parts) >= 2 {
		r.On = parts[1] == "1"
	}
	if len(parts) >= 3 {
		r.Speed = parts[2]
	}
	if len(parts) >= 4 {
		r.Curve = parts[3]
	}
	if len(parts) >= 5 {
		r.Style = parts[4]
	}
	return r
}

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

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "true" || s == "yes" || s == "1" || strings.HasPrefix(s, "yes")
}

func atoi(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func atof(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
