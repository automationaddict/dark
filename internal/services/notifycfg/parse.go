package notifycfg

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func parseUrgencyAndNotify(s *Snapshot) {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "omarchy", "default", "mako", "core.ini")
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	var currentSection string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch currentSection {
		case "urgency=low":
			if key == "default-timeout" {
				s.LowTimeout = atoi(val, -1)
			}
		case "urgency=critical":
			switch key {
			case "default-timeout":
				s.CritTimeout = atoi(val, -1)
			case "layer":
				s.CritLayer = val
			}
		}
	}
}

func (s *Snapshot) applyGlobal(key, val string) {
	switch key {
	case "anchor":
		s.Anchor = val
	case "default-timeout":
		s.TimeoutMS = atoi(val, s.TimeoutMS)
	case "width":
		s.Width = atoi(val, s.Width)
	case "height":
		s.Height = atoi(val, s.Height)
	case "padding":
		s.Padding = val
	case "border-size":
		s.BorderSize = atoi(val, s.BorderSize)
	case "border-radius":
		s.BorderRadius = atoi(val, s.BorderRadius)
	case "font":
		s.Font = val
	case "max-icon-size":
		s.MaxIcon = atoi(val, s.MaxIcon)
	case "max-history":
		s.MaxHistory = atoi(val, s.MaxHistory)
	case "icons":
		s.Icons = val == "1"
	case "markup":
		s.Markup = val == "1"
	case "actions":
		s.Actions = val == "1"
	case "text-color":
		s.TextColor = val
	case "border-color":
		s.BorderColor = val
	case "background-color":
		s.BgColor = val
	case "layer":
		s.Layer = val
	case "max-visible":
		s.MaxVisible = atoi(val, s.MaxVisible)
	case "on-notify":
		s.NotifySound = val
	case "group-format":
		s.GroupFormat = val
	}
}

func atoi(s string, fallback int) int {
	v := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			v = v*10 + int(c-'0')
		} else {
			break
		}
	}
	if v == 0 && s != "0" {
		return fallback
	}
	return v
}

func isDaemonRunning() bool {
	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		comm, err := os.ReadFile(filepath.Join("/proc", e.Name(), "comm"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(comm)) == "mako" {
			return true
		}
	}
	return false
}

func isDNDActive() bool {
	out, err := exec.Command("makoctl", "mode").Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "do-not-disturb" {
			return true
		}
	}
	return false
}

func readMakoConfigs() (core map[string]string, theme map[string]string) {
	core = make(map[string]string)
	theme = make(map[string]string)

	home := os.Getenv("HOME")

	corePath := filepath.Join(home, ".local", "share", "omarchy", "default", "mako", "core.ini")
	parseINIGlobals(corePath, core)

	themePath := filepath.Join(home, ".config", "omarchy", "current", "theme", "mako.ini")
	parseINIGlobals(themePath, theme)

	return
}

func parseINIGlobals(path string, out map[string]string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			break
		}
		if strings.HasPrefix(line, "include=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			out[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
}

func parseRules() []AppRule {
	home := os.Getenv("HOME")
	path := filepath.Join(home, ".local", "share", "omarchy", "default", "mako", "core.ini")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var rules []AppRule
	var currentSection string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = line[1 : len(line)-1]
			continue
		}
		if currentSection == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		action := ""
		switch key {
		case "invisible":
			if val == "1" || val == "true" {
				action = "hidden"
			} else {
				action = "visible"
			}
		case "default-timeout":
			action = "timeout " + val + "ms"
		case "layer":
			action = "layer: " + val
		case "on-button-left":
			action = "click action"
		case "max-icon-size":
			action = "icon: " + val + "px"
		case "format":
			action = "custom format"
		default:
			action = key + "=" + val
		}

		if action != "" {
			existing := false
			for i, r := range rules {
				if r.Criteria == currentSection {
					rules[i].Action += ", " + action
					existing = true
					break
				}
			}
			if !existing {
				rules = append(rules, AppRule{Criteria: currentSection, Action: action})
			}
		}
	}
	return rules
}

func readHistory() []HistoryItem {
	out, err := exec.Command("makoctl", "history").Output()
	if err != nil {
		return nil
	}

	var data struct {
		Type string `json:"type"`
		Data [][]struct {
			AppName struct{ Data string } `json:"app-name"`
			Summary struct{ Data string } `json:"summary"`
			Body    struct{ Data string } `json:"body"`
			Urgency struct{ Data int }    `json:"urgency"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil
	}

	var items []HistoryItem
	for _, group := range data.Data {
		for _, n := range group {
			urgency := "normal"
			switch n.Urgency.Data {
			case 0:
				urgency = "low"
			case 2:
				urgency = "critical"
			}
			items = append(items, HistoryItem{
				AppName: n.AppName.Data,
				Summary: n.Summary.Data,
				Body:    n.Body.Data,
				Urgency: urgency,
			})
		}
	}
	return items
}

func listSounds() []string {
	dir := "/usr/share/sounds/freedesktop/stereo"
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var sounds []string
	for _, e := range entries {
		name := e.Name()
		name = strings.TrimSuffix(name, filepath.Ext(name))
		sounds = append(sounds, name)
	}
	return sounds
}
