package workspaces

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestWorkspaceUnmarshal verifies the Workspace struct round-trips
// hyprctl's actual JSON shape. The sample payload below is a real
// capture from a ghostty + firefox session.
func TestWorkspaceUnmarshal(t *testing.T) {
	raw := `[{
    "id": 1,
    "name": "1",
    "monitor": "eDP-1",
    "monitorID": 0,
    "windows": 3,
    "hasfullscreen": false,
    "lastwindow": "0x55634c182ac0",
    "lastwindowtitle": "Claude Code",
    "ispersistent": false,
    "tiledLayout": "dwindle"
}]`
	var got []Workspace
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	want := []Workspace{{
		ID:              1,
		Name:            "1",
		Monitor:         "eDP-1",
		MonitorID:       0,
		Windows:         3,
		HasFullscreen:   false,
		LastWindow:      "0x55634c182ac0",
		LastWindowTitle: "Claude Code",
		IsPersistent:    false,
		TiledLayout:     "dwindle",
	}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("workspace mismatch:\nwant: %+v\ngot:  %+v", want, got)
	}
}

// TestWorkspaceRuleUnmarshal covers the persistent-workspace rules
// endpoint. Minimal rule (just a workspace string) should decode
// without erroring.
func TestWorkspaceRuleUnmarshal(t *testing.T) {
	raw := `[{"workspaceString": "1"}, {"workspaceString": "2", "monitor": "eDP-1", "persistent": true}]`
	var got []WorkspaceRule
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(got))
	}
	if got[0].WorkspaceString != "1" {
		t.Errorf("rule 0 workspace = %q", got[0].WorkspaceString)
	}
	if !got[1].Persistent {
		t.Errorf("rule 1 should be persistent")
	}
	if got[1].Monitor != "eDP-1" {
		t.Errorf("rule 1 monitor = %q", got[1].Monitor)
	}
}

// TestBoolToInt checks the tiny helper used for hyprctl keyword
// writes. Hyprland expects "0" or "1" for bool options, never
// "true"/"false".
func TestBoolToInt(t *testing.T) {
	if got := boolToInt(true); got != "1" {
		t.Errorf("boolToInt(true) = %q, want 1", got)
	}
	if got := boolToInt(false); got != "0" {
		t.Errorf("boolToInt(false) = %q, want 0", got)
	}
}

// TestSnapshotZeroReadable is a smoke test: a zero-value Snapshot
// shouldn't crash the JSON encoder and should produce an output
// that carries all the expected top-level fields.
func TestSnapshotZeroJSON(t *testing.T) {
	data, err := json.Marshal(Snapshot{})
	if err != nil {
		t.Fatal(err)
	}
	// Spot check that the animations flag is present by name.
	if !jsonHasKey(data, "animations_enabled") {
		t.Errorf("animations_enabled missing from JSON: %s", data)
	}
	if !jsonHasKey(data, "default_layout") {
		t.Errorf("default_layout missing from JSON: %s", data)
	}
}

func jsonHasKey(data []byte, key string) bool {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return false
	}
	_, ok := m[key]
	return ok
}
