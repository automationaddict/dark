package notifycfg

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func ToggleDND() error {
	return exec.Command("makoctl", "mode", "-t", "do-not-disturb").Run()
}

func DismissAll() error {
	return exec.Command("makoctl", "dismiss", "-a").Run()
}

func SetAnchor(anchor string) error {
	return setGlobalOption("anchor", anchor)
}

func SetTimeout(ms int) error {
	return setGlobalOption("default-timeout", fmt.Sprintf("%d", ms))
}

func SetWidth(px int) error {
	return setGlobalOption("width", fmt.Sprintf("%d", px))
}

func SetBorderSize(px int) error {
	return setGlobalOption("border-size", fmt.Sprintf("%d", px))
}

func SetMaxIcon(px int) error {
	return setGlobalOption("max-icon-size", fmt.Sprintf("%d", px))
}

func AddAppRule(appName string, hide bool) error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "omarchy", "default", "mako", "core.ini")

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	section := fmt.Sprintf("[app-name=%s]", appName)
	if strings.Contains(string(data), section) {
		return fmt.Errorf("rule for %s already exists", appName)
	}

	action := "invisible=0"
	if hide {
		action = "invisible=1"
	}

	content := string(data) + "\n" + section + "\n" + action + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	return reloadMako()
}

func RemoveAppRule(criteria string) error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "omarchy", "default", "mako", "core.ini")

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var out []string
	skip := false
	target := "[" + criteria + "]"
	for _, line := range lines {
		if strings.TrimSpace(line) == target {
			skip = true
			continue
		}
		if skip && strings.HasPrefix(strings.TrimSpace(line), "[") {
			skip = false
		}
		if skip && strings.TrimSpace(line) == "" {
			skip = false
			continue
		}
		if !skip {
			out = append(out, line)
		}
	}

	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return err
	}
	return reloadMako()
}

func setGlobalOption(key, value string) error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "omarchy", "default", "mako", "core.ini")

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			break
		}
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}
	if !found {
		// Insert before first section
		var out []string
		inserted := false
		for _, line := range lines {
			if !inserted && strings.HasPrefix(strings.TrimSpace(line), "[") {
				out = append(out, key+"="+value)
				inserted = true
			}
			out = append(out, line)
		}
		if !inserted {
			out = append(out, key+"="+value)
		}
		lines = out
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	return reloadMako()
}

func SetMaxVisible(n int) error {
	return setGlobalOption("max-visible", fmt.Sprintf("%d", n))
}

func SetLayer(layer string) error {
	return setGlobalOption("layer", layer)
}

func SetNotifySound(soundPath string) error {
	if soundPath == "" {
		return removeGlobalOption("on-notify")
	}
	return setGlobalOption("on-notify", "exec mpv "+soundPath)
}

func SetGroupFormat(format string) error {
	return setGlobalOption("group-format", format)
}

func SetUrgencyTimeout(urgency string, ms int) error {
	return setSectionOption("urgency="+urgency, "default-timeout", fmt.Sprintf("%d", ms))
}

func SetCritLayer(layer string) error {
	return setSectionOption("urgency=critical", "layer", layer)
}

func setSectionOption(section, key, value string) error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "omarchy", "default", "mako", "core.ini")

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	target := "[" + section + "]"
	inSection := false
	found := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == target {
			inSection = true
			continue
		}
		if inSection && strings.HasPrefix(trimmed, "[") {
			if !found {
				// Insert before next section
				insert := key + "=" + value
				newLines := make([]string, 0, len(lines)+1)
				newLines = append(newLines, lines[:i]...)
				newLines = append(newLines, insert)
				newLines = append(newLines, lines[i:]...)
				lines = newLines
				found = true
			}
			break
		}
		if inSection && strings.HasPrefix(trimmed, key+"=") {
			lines[i] = key + "=" + value
			found = true
			break
		}
	}

	if !found && inSection {
		lines = append(lines, key+"="+value)
		found = true
	}

	if !found {
		// Section doesn't exist — create it
		lines = append(lines, "", target, key+"="+value)
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	return reloadMako()
}

func removeGlobalOption(key string) error {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "omarchy", "default", "mako", "core.ini")

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			out = append(out, line)
			continue
		}
		if strings.HasPrefix(trimmed, key+"=") {
			continue
		}
		out = append(out, line)
	}

	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return err
	}
	return reloadMako()
}
