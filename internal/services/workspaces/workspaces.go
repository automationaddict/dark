// Package workspaces wraps Hyprland's workspace state and the
// handful of workspace-adjacent config knobs users typically want
// to tweak (default layout, dwindle and master options, cursor
// warp, workspace animation).
//
// Everything runs through `hyprctl` — live reads for the snapshot,
// `hyprctl keyword …` for setter-style mutations, `hyprctl
// dispatch …` for workspace navigation. None of this needs
// privileges, so no dark-helper is involved.
package workspaces

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Workspace is one Hyprland workspace as reported by
// `hyprctl workspaces -j`. The shape matches Hyprland's JSON
// output; unused fields are omitted so the snapshot stays light.
type Workspace struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	Monitor          string `json:"monitor"`
	MonitorID        int    `json:"monitorID"`
	Windows          int    `json:"windows"`
	HasFullscreen    bool   `json:"hasfullscreen"`
	LastWindow       string `json:"lastwindow,omitempty"`
	LastWindowTitle  string `json:"lastwindowtitle,omitempty"`
	IsPersistent     bool   `json:"ispersistent"`
	TiledLayout      string `json:"tiledLayout,omitempty"`
}

// WorkspaceRule is one persistent-workspace rule from
// `hyprctl workspacerules -j`. These come from `workspace = …`
// lines in hyprland.conf and are separate from live workspaces —
// a persistent rule exists whether or not there's an active
// workspace matching it.
type WorkspaceRule struct {
	WorkspaceString string `json:"workspaceString"`
	Monitor         string `json:"monitor,omitempty"`
	Default         bool   `json:"default,omitempty"`
	Persistent      bool   `json:"persistent,omitempty"`
	Layoutopt       string `json:"layoutopt,omitempty"`
}

// DwindleOptions is the subset of dwindle-layout options dark
// surfaces as editable rows. Hyprland has more; these are the
// ones users tweak.
type DwindleOptions struct {
	Pseudotile     bool `json:"pseudotile"`
	PreserveSplit  bool `json:"preserve_split"`
	ForceSplit     int  `json:"force_split"`
	SmartSplit     bool `json:"smart_split"`
	SmartResizing  bool `json:"smart_resizing"`
}

// MasterOptions mirrors DwindleOptions for the master layout.
type MasterOptions struct {
	NewStatus   string `json:"new_status"`
	Orientation string `json:"orientation"`
}

// Snapshot is the payload darkd publishes on
// dark.workspaces.snapshot. Everything the Workspaces settings
// panel needs to render lives in here — live workspace list,
// persistent rule list, active workspace ID, the global default
// layout, per-layout option blocks, and a handful of behavior
// flags (cursor warp, animation state).
type Snapshot struct {
	Workspaces       []Workspace     `json:"workspaces"`
	Rules            []WorkspaceRule `json:"rules,omitempty"`
	ActiveID         int             `json:"active_id"`
	DefaultLayout    string          `json:"default_layout"` // dwindle / master
	AvailableLayouts []string        `json:"available_layouts"`
	Dwindle          DwindleOptions  `json:"dwindle"`
	Master           MasterOptions   `json:"master"`

	// Behavior flags that affect workspace navigation feel.
	CursorWarp            bool `json:"cursor_warp"`
	AnimationsEnabled     bool `json:"animations_enabled"`
	WorkspaceAnimationOn  bool `json:"workspace_animation_on"`
	HideSpecialOnChange   bool `json:"hide_special_on_change"`
}

// ReadSnapshot assembles a Snapshot from hyprctl reads. Any
// individual read failure degrades gracefully to zero values for
// that field rather than failing the whole snapshot — the panel
// should still show what it can when one of the reads errors.
func ReadSnapshot() Snapshot {
	snap := Snapshot{
		AvailableLayouts: []string{"dwindle", "master"},
	}

	if wss, err := listWorkspaces(); err == nil {
		snap.Workspaces = wss
	}
	if rules, err := listRules(); err == nil {
		snap.Rules = rules
	}
	if id, err := activeWorkspaceID(); err == nil {
		snap.ActiveID = id
	}
	if layout, err := readString("general:layout"); err == nil {
		snap.DefaultLayout = layout
	}

	snap.Dwindle.Pseudotile, _ = readBool("dwindle:pseudotile")
	snap.Dwindle.PreserveSplit, _ = readBool("dwindle:preserve_split")
	snap.Dwindle.ForceSplit, _ = readInt("dwindle:force_split")
	snap.Dwindle.SmartSplit, _ = readBool("dwindle:smart_split")
	snap.Dwindle.SmartResizing, _ = readBool("dwindle:smart_resizing")

	snap.Master.NewStatus, _ = readString("master:new_status")
	snap.Master.Orientation, _ = readString("master:orientation")

	snap.CursorWarp, _ = readBool("cursor:warp_on_change_workspace")
	snap.AnimationsEnabled, _ = readBool("animations:enabled")
	snap.HideSpecialOnChange, _ = readBool("binds:hide_special_on_workspace_change")

	// The workspaces-specific animation line lives under
	// animations and is toggled individually. Hyprland doesn't
	// expose it via getoption for its speed / enabled pair, so
	// we surface a single "enabled" signal based on whether the
	// animations block is on at all — users who want to fine-
	// tune the curve per-animation still edit hyprland.conf.
	snap.WorkspaceAnimationOn = snap.AnimationsEnabled

	return snap
}

// SwitchWorkspace dispatches a switch to the given workspace. id
// may be an integer (navigates to workspace N, creating it if it
// doesn't exist) or a name like "web".
func SwitchWorkspace(id string) error {
	return hyprctl("dispatch", "workspace", id)
}

// RenameWorkspace sets the display name of an existing workspace.
func RenameWorkspace(id int, name string) error {
	return hyprctl("dispatch", "renameworkspace", strconv.Itoa(id), name)
}

// MoveWorkspaceToMonitor moves an existing workspace to a named
// monitor. The monitor name must match `hyprctl monitors` output.
func MoveWorkspaceToMonitor(id int, monitor string) error {
	return hyprctl("dispatch", "moveworkspacetomonitor", strconv.Itoa(id), monitor)
}

// SetWorkspaceLayout changes the tiling layout of a single
// workspace. Accepted values are whatever Hyprland exposes —
// typically "dwindle" or "master". Unknown values fail silently
// from hyprctl's side; dark surfaces any error it returns.
func SetWorkspaceLayout(id int, layout string) error {
	return hyprctl("keyword", "workspace",
		fmt.Sprintf("%d, layout:%s", id, layout))
}

// SetDefaultLayout changes the global default layout. Existing
// workspaces keep their current layout; new workspaces pick up
// the default.
func SetDefaultLayout(layout string) error {
	return hyprctl("keyword", "general:layout", layout)
}

// SetDwindleOption toggles one of the dwindle layout options.
// key is the hyprland config name ("pseudotile", "preserve_split",
// "force_split", "smart_split", "smart_resizing"). value is the
// string form — "true"/"false" for bools, an integer for
// force_split.
func SetDwindleOption(key, value string) error {
	return hyprctl("keyword", "dwindle:"+key, value)
}

// SetMasterOption mirrors SetDwindleOption for the master layout.
func SetMasterOption(key, value string) error {
	return hyprctl("keyword", "master:"+key, value)
}

// SetCursorWarp toggles cursor:warp_on_change_workspace. When
// enabled, switching workspaces moves the cursor to the new
// focus target instead of leaving it in place.
func SetCursorWarp(enabled bool) error {
	return hyprctl("keyword", "cursor:warp_on_change_workspace", boolToInt(enabled))
}

// SetAnimationsEnabled toggles the global animations flag. Gates
// every Hyprland animation, not just workspace transitions —
// there's no per-animation kill switch exposed via getoption.
func SetAnimationsEnabled(enabled bool) error {
	return hyprctl("keyword", "animations:enabled", boolToInt(enabled))
}

// SetHideSpecialOnChange toggles whether the special workspace
// auto-hides when the user switches to a regular workspace.
func SetHideSpecialOnChange(enabled bool) error {
	return hyprctl("keyword", "binds:hide_special_on_workspace_change", boolToInt(enabled))
}

// ─── hyprctl read helpers ─────────────────────────────────────────

func listWorkspaces() ([]Workspace, error) {
	out, err := exec.Command("hyprctl", "workspaces", "-j").Output()
	if err != nil {
		return nil, err
	}
	var wss []Workspace
	if err := json.Unmarshal(out, &wss); err != nil {
		return nil, err
	}
	return wss, nil
}

func listRules() ([]WorkspaceRule, error) {
	out, err := exec.Command("hyprctl", "workspacerules", "-j").Output()
	if err != nil {
		return nil, err
	}
	var rules []WorkspaceRule
	if err := json.Unmarshal(out, &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func activeWorkspaceID() (int, error) {
	out, err := exec.Command("hyprctl", "activeworkspace", "-j").Output()
	if err != nil {
		return 0, err
	}
	var ws struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(out, &ws); err != nil {
		return 0, err
	}
	return ws.ID, nil
}

// readString / readBool / readInt wrap `hyprctl getoption` for
// typed reads of individual config values. hyprctl's getoption
// output is line-based: either "str: <value>\nset: true" or
// "int: <value>\nset: true". We parse the first line.
func readString(key string) (string, error) {
	out, err := exec.Command("hyprctl", "getoption", key).Output()
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "str:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "str:")), nil
		}
	}
	return "", fmt.Errorf("no str value in %s output", key)
}

func readInt(key string) (int, error) {
	out, err := exec.Command("hyprctl", "getoption", key).Output()
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "int:") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "int:"))
			return strconv.Atoi(raw)
		}
	}
	return 0, fmt.Errorf("no int value in %s output", key)
}

func readBool(key string) (bool, error) {
	// Hyprland exposes bools as ints (0 / 1). Some options have
	// both a "bool:" line and an "int:" line — we accept either.
	out, err := exec.Command("hyprctl", "getoption", key).Output()
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "int:") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "int:"))
			n, err := strconv.Atoi(raw)
			if err != nil {
				return false, err
			}
			return n != 0, nil
		}
		if strings.HasPrefix(line, "bool:") {
			raw := strings.TrimSpace(strings.TrimPrefix(line, "bool:"))
			return raw == "true", nil
		}
	}
	return false, fmt.Errorf("no bool value in %s output", key)
}

// ─── hyprctl write helpers ────────────────────────────────────────

// hyprctl runs a single hyprctl command and returns any error
// with the combined stderr attached so callers can surface it to
// the UI without a second read.
func hyprctl(args ...string) error {
	cmd := exec.Command("hyprctl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("hyprctl %s: %w: %s",
			strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func boolToInt(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
