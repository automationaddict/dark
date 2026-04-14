package display

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
)

type Profile struct {
	Name     string          `json:"name"`
	Monitors []MonitorConfig `json:"monitors"`
}

type MonitorConfig struct {
	Name        string  `json:"name"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	RefreshRate float64 `json:"refresh_rate"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	Scale       float64 `json:"scale"`
	Transform   int     `json:"transform"`
	Vrr         bool    `json:"vrr"`
	Enabled     bool    `json:"enabled"`
}

func ProfileDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "dark", "display-profiles")
}

func SaveProfile(name string, snap Snapshot) error {
	dir := ProfileDir()
	if dir == "" {
		return fmt.Errorf("cannot determine profile directory")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create profile directory: %w", err)
	}
	var configs []MonitorConfig
	for _, m := range snap.Monitors {
		configs = append(configs, MonitorConfig{
			Name:        m.Name,
			Width:       m.Width,
			Height:      m.Height,
			RefreshRate: m.RefreshRate,
			X:           m.X,
			Y:           m.Y,
			Scale:       m.Scale,
			Transform:   m.Transform,
			Vrr:         m.Vrr,
			Enabled:     !m.Disabled,
		})
	}
	profile := Profile{Name: name, Monitors: configs}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}
	path := filepath.Join(dir, sanitizeProfileName(name)+".json")
	return os.WriteFile(path, data, 0644)
}

func LoadProfile(name string) (Profile, error) {
	dir := ProfileDir()
	if dir == "" {
		return Profile{}, fmt.Errorf("cannot determine profile directory")
	}
	path := filepath.Join(dir, sanitizeProfileName(name)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, fmt.Errorf("read profile %q: %w", name, err)
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, fmt.Errorf("parse profile %q: %w", name, err)
	}
	return p, nil
}

func ListProfiles() ([]string, error) {
	dir := ProfileDir()
	if dir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".json")
		names = append(names, base)
	}
	return names, nil
}

func DeleteProfile(name string) error {
	dir := ProfileDir()
	if dir == "" {
		return fmt.Errorf("cannot determine profile directory")
	}
	path := filepath.Join(dir, sanitizeProfileName(name)+".json")
	return os.Remove(path)
}

func ApplyProfile(profile Profile) error {
	for _, mc := range profile.Monitors {
		if !mc.Enabled {
			if err := hyprctl("keyword", "monitor", mc.Name+",disabled"); err != nil {
				return fmt.Errorf("disable %s: %w", mc.Name, err)
			}
			continue
		}
		mode := fmt.Sprintf("%dx%d@%.2f", mc.Width, mc.Height, mc.RefreshRate)
		pos := fmt.Sprintf("%dx%d", mc.X, mc.Y)
		scale := fmt.Sprintf("%.2f", mc.Scale)
		arg := mc.Name + "," + mode + "," + pos + "," + scale
		if mc.Transform != 0 {
			arg += fmt.Sprintf(",transform,%d", mc.Transform)
		}
		if err := hyprctl("keyword", "monitor", arg); err != nil {
			return fmt.Errorf("apply %s: %w", mc.Name, err)
		}
		vrrMode := 0
		if mc.Vrr {
			vrrMode = 1
		}
		_ = hyprctl("keyword", "monitor", mc.Name+",vrr,"+fmt.Sprint(vrrMode))
	}
	return nil
}

func sanitizeProfileName(name string) string {
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "..", "_")
	return name
}

func ClosestScaleIndex(options []string, current float64) int {
	best := 0
	bestDist := math.MaxFloat64
	for i, opt := range options {
		var v float64
		if _, err := fmt.Sscanf(opt, "%f", &v); err == nil {
			d := math.Abs(v - current)
			if d < bestDist {
				bestDist = d
				best = i
			}
		}
	}
	return best
}
