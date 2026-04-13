package input

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// --- Hyprland keyword mutations ---

func SetRepeatRate(rate int) error {
	return hyprctl("keyword", "input:repeat_rate", strconv.Itoa(rate))
}

func SetRepeatDelay(delay int) error {
	return hyprctl("keyword", "input:repeat_delay", strconv.Itoa(delay))
}

func SetSensitivity(sens float64) error {
	return hyprctl("keyword", "input:sensitivity", fmt.Sprintf("%.2f", sens))
}

func SetNaturalScroll(enabled bool) error {
	v := "false"
	if enabled {
		v = "true"
	}
	return hyprctl("keyword", "input:touchpad:natural_scroll", v)
}

func SetScrollFactor(factor float64) error {
	return hyprctl("keyword", "input:touchpad:scroll_factor", fmt.Sprintf("%.2f", factor))
}

func SetKBLayout(layout string) error {
	return hyprctl("keyword", "input:kb_layout", layout)
}

func SetAccelProfile(profile string) error {
	return hyprctl("keyword", "input:accel_profile", profile)
}

func SetForceNoAccel(enabled bool) error {
	return hyprctl("keyword", "input:force_no_accel", boolStr(enabled))
}

func SetLeftHanded(enabled bool) error {
	return hyprctl("keyword", "input:left_handed", boolStr(enabled))
}

func SetFollowMouse(mode int) error {
	return hyprctl("keyword", "input:follow_mouse", strconv.Itoa(mode))
}

func SetDisableWhileTyping(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:disable_while_typing", boolStr(enabled))
}

func SetTapToClick(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:tap-to-click", boolStr(enabled))
}

func SetTapAndDrag(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:tap-and-drag", boolStr(enabled))
}

func SetDragLock(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:drag_lock", boolStr(enabled))
}

func SetMiddleButtonEmu(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:middle_button_emulation", boolStr(enabled))
}

func SetClickfingerBehavior(enabled bool) error {
	return hyprctl("keyword", "input:touchpad:clickfinger_behavior", boolStr(enabled))
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func hyprctl(args ...string) error {
	cmd := exec.Command("hyprctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hyprctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
