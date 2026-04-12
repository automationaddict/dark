package display

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type hyprlandBackend struct{}

func newHyprlandBackend() (*hyprlandBackend, error) {
	if os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") == "" {
		return nil, fmt.Errorf("not running under Hyprland")
	}
	if _, err := exec.LookPath("hyprctl"); err != nil {
		return nil, fmt.Errorf("hyprctl not found: %w", err)
	}
	return &hyprlandBackend{}, nil
}

func (b *hyprlandBackend) Name() string { return "hyprland" }

func (b *hyprlandBackend) Snapshot() Snapshot {
	out, err := exec.Command("hyprctl", "monitors", "-j").Output()
	if err != nil {
		return Snapshot{}
	}
	var monitors []Monitor
	if err := json.Unmarshal(out, &monitors); err != nil {
		return Snapshot{}
	}
	return Snapshot{Monitors: monitors}
}

func (b *hyprlandBackend) Close() {}

func (b *hyprlandBackend) SetResolution(name string, width, height int, refreshRate float64) error {
	mode := fmt.Sprintf("%dx%d@%.2f", width, height, refreshRate)
	return hyprctl("keyword", "monitor", name+","+mode+",auto,auto")
}

func (b *hyprlandBackend) SetScale(name string, scale float64) error {
	snap := b.Snapshot()
	for _, m := range snap.Monitors {
		if m.Name == name {
			mode := fmt.Sprintf("%dx%d@%.2f", m.Width, m.Height, m.RefreshRate)
			pos := fmt.Sprintf("%dx%d", m.X, m.Y)
			scaleStr := fmt.Sprintf("%.2f", scale)
			return hyprctl("keyword", "monitor", name+","+mode+","+pos+","+scaleStr)
		}
	}
	return fmt.Errorf("monitor %s not found", name)
}

func (b *hyprlandBackend) SetTransform(name string, transform int) error {
	return hyprctl("keyword", "monitor", name+",transform,"+fmt.Sprint(transform))
}

func (b *hyprlandBackend) SetPosition(name string, x, y int) error {
	snap := b.Snapshot()
	for _, m := range snap.Monitors {
		if m.Name == name {
			mode := fmt.Sprintf("%dx%d@%.2f", m.Width, m.Height, m.RefreshRate)
			pos := fmt.Sprintf("%dx%d", x, y)
			scaleStr := fmt.Sprintf("%.2f", m.Scale)
			return hyprctl("keyword", "monitor", name+","+mode+","+pos+","+scaleStr)
		}
	}
	return fmt.Errorf("monitor %s not found", name)
}

func (b *hyprlandBackend) SetDpms(name string, on bool) error {
	flag := "off"
	if on {
		flag = "on"
	}
	return hyprctl("dispatch", "dpms", flag, name)
}

func (b *hyprlandBackend) SetVrr(name string, mode int) error {
	return hyprctl("keyword", "monitor", name+",vrr,"+fmt.Sprint(mode))
}

func (b *hyprlandBackend) SetMirror(name, mirrorOf string) error {
	return hyprctl("keyword", "monitor", name+",preferred,auto,1,mirror,"+mirrorOf)
}

func (b *hyprlandBackend) ToggleEnabled(name string) error {
	snap := b.Snapshot()
	for _, m := range snap.Monitors {
		if m.Name == name {
			if m.Disabled {
				return hyprctl("keyword", "monitor", name+",preferred,auto,auto")
			}
			return hyprctl("keyword", "monitor", name+",disabled")
		}
	}
	return fmt.Errorf("monitor %s not found", name)
}

func (b *hyprlandBackend) Identify() error {
	snap := b.Snapshot()
	if len(snap.Monitors) == 0 {
		return fmt.Errorf("no monitors to identify")
	}

	var focused string
	for _, m := range snap.Monitors {
		if m.Focused {
			focused = m.Name
			break
		}
	}

	for i, m := range snap.Monitors {
		_ = hyprctl("dispatch", "focusmonitor", m.Name)
		label := fmt.Sprintf("  %d: %s  ", i+1, m.Name)
		_ = hyprctl("notify", "1", "3000", "0", label)
	}

	if focused != "" {
		_ = hyprctl("dispatch", "focusmonitor", focused)
	}
	return nil
}

func hyprctl(args ...string) error {
	cmd := exec.Command("hyprctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hyprctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
