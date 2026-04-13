package input

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// --- Hyprland input config parsing ---

func readHyprlandConfig() InputConfig {
	cfg := InputConfig{
		KBLayout:           "us",
		RepeatRate:         25,
		RepeatDelay:        600,
		FollowMouse:        1,
		DisableWhileTyping: true,
		TapToClick:         true,
		TapAndDrag:         true,
	}

	cfg.KBLayout = hyprOptionStr("input:kb_layout", cfg.KBLayout)
	cfg.KBVariant = hyprOptionStr("input:kb_variant", "")
	cfg.KBModel = hyprOptionStr("input:kb_model", "")
	cfg.KBOptions = hyprOptionStr("input:kb_options", "")
	cfg.RepeatRate = hyprOptionInt("input:repeat_rate", cfg.RepeatRate)
	cfg.RepeatDelay = hyprOptionInt("input:repeat_delay", cfg.RepeatDelay)
	cfg.NumlockDefault = hyprOptionBool("input:numlock_by_default", false)

	cfg.Sensitivity = hyprOptionFloat("input:sensitivity", 0)
	cfg.AccelProfile = hyprOptionStr("input:accel_profile", "")
	cfg.ForceNoAccel = hyprOptionBool("input:force_no_accel", false)
	cfg.LeftHanded = hyprOptionBool("input:left_handed", false)
	cfg.ScrollMethod = hyprOptionStr("input:scroll_method", "")
	cfg.FollowMouse = hyprOptionInt("input:follow_mouse", 1)

	cfg.NaturalScroll = hyprOptionBool("input:touchpad:natural_scroll", false)
	cfg.ScrollFactor = hyprOptionFloat("input:touchpad:scroll_factor", 1.0)
	cfg.DisableWhileTyping = hyprOptionBool("input:touchpad:disable_while_typing", true)
	cfg.TapToClick = hyprOptionBool("input:touchpad:tap-to-click", true)
	cfg.TapAndDrag = hyprOptionBool("input:touchpad:tap-and-drag", true)
	cfg.DragLock = hyprOptionBool("input:touchpad:drag_lock", false)
	cfg.MiddleButtonEmu = hyprOptionBool("input:touchpad:middle_button_emulation", false)
	cfg.ClickfingerBehavior = hyprOptionBool("input:touchpad:clickfinger_behavior", false)

	return cfg
}

func hyprOptionStr(key, fallback string) string {
	out, err := exec.Command("hyprctl", "getoption", key, "-j").Output()
	if err != nil {
		return fallback
	}
	// Parse JSON: {"option":"...","str":"value","int":0,"float":0.0,"set":true}
	var result struct {
		Str string `json:"str"`
	}
	if err := parseJSON(out, &result); err != nil {
		return fallback
	}
	if result.Str == "[EMPTY]" || result.Str == "" {
		return fallback
	}
	return result.Str
}

func hyprOptionInt(key string, fallback int) int {
	out, err := exec.Command("hyprctl", "getoption", key, "-j").Output()
	if err != nil {
		return fallback
	}
	var result struct {
		Int int `json:"int"`
	}
	if err := parseJSON(out, &result); err != nil {
		return fallback
	}
	return result.Int
}

func hyprOptionFloat(key string, fallback float64) float64 {
	out, err := exec.Command("hyprctl", "getoption", key, "-j").Output()
	if err != nil {
		return fallback
	}
	var result struct {
		Float float64 `json:"float"`
	}
	if err := parseJSON(out, &result); err != nil {
		return fallback
	}
	return result.Float
}

func hyprOptionBool(key string, fallback bool) bool {
	v := hyprOptionInt(key, -1)
	if v == -1 {
		return fallback
	}
	return v == 1
}

func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func readSysStr(dir, name string) string {
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readSysInt(dir, name string) int {
	s := readSysStr(dir, name)
	v, _ := strconv.Atoi(s)
	return v
}
