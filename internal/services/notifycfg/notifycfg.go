package notifycfg

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Snapshot struct {
	Running     bool          `json:"running"`
	Daemon      string        `json:"daemon"`
	Anchor      string        `json:"anchor"`
	TimeoutMS   int           `json:"timeout_ms"`
	Width       int           `json:"width"`
	Height      int           `json:"height"`
	Padding     string        `json:"padding"`
	BorderSize  int           `json:"border_size"`
	BorderRadius int          `json:"border_radius"`
	Font        string        `json:"font"`
	MaxIcon     int           `json:"max_icon"`
	MaxHistory  int           `json:"max_history"`
	Icons       bool          `json:"icons"`
	Markup      bool          `json:"markup"`
	Actions     bool          `json:"actions"`
	TextColor   string        `json:"text_color"`
	BorderColor string        `json:"border_color"`
	BgColor     string        `json:"bg_color"`
	Layer       string        `json:"layer"`
	DNDActive   bool          `json:"dnd_active"`
	Rules       []AppRule     `json:"rules"`
	History     []HistoryItem `json:"history"`
	Sounds      []string      `json:"sounds"`
}

type AppRule struct {
	Criteria string `json:"criteria"`
	Action   string `json:"action"`
}

type HistoryItem struct {
	AppName string `json:"app_name"`
	Summary string `json:"summary"`
	Body    string `json:"body"`
	Urgency string `json:"urgency"`
}

func ReadSnapshot() Snapshot {
	s := Snapshot{
		Daemon:     "mako",
		TimeoutMS:  5000,
		Width:      300,
		Height:     100,
		BorderSize: 2,
		MaxIcon:    64,
		MaxHistory: 5,
		Icons:      true,
		Markup:     true,
		Actions:    true,
		Anchor:     "top-right",
		Layer:      "top",
	}

	s.Running = isDaemonRunning()
	s.DNDActive = isDNDActive()

	core, theme := readMakoConfigs()
	for k, v := range core {
		s.applyGlobal(k, v)
	}
	for k, v := range theme {
		s.applyGlobal(k, v)
	}

	s.Rules = parseRules()
	s.History = readHistory()
	s.Sounds = listSounds()

	return s
}

// --- Mutations ---

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

func reloadMako() error {
	return exec.Command("makoctl", "reload").Run()
}

// --- Snapshot helpers ---

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
