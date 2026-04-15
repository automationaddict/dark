package topbar

import (
	"reflect"
	"strings"
	"testing"
)

func TestStripJSONCComments(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "line comment",
			in:   `{ "a": 1, // trailing` + "\n" + `"b": 2 }`,
			want: `{ "a": 1, ` + "\n" + `"b": 2 }`,
		},
		{
			name: "block comment",
			in:   `{ /* header */ "a": 1 }`,
			want: `{  "a": 1 }`,
		},
		{
			name: "comment-like in string",
			in:   `{ "x": "// not a comment" }`,
			want: `{ "x": "// not a comment" }`,
		},
		{
			name: "escaped quote inside string",
			in:   `{ "x": "he said \"hi\"" }`,
			want: `{ "x": "he said \"hi\"" }`,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			got := stripJSONCComments(tt.in)
			if got != tt.want {
				t.Errorf("stripJSONCComments:\nwant: %q\ngot:  %q", tt.want, got)
			}
		})
	}
}

func TestParseWaybarConfigHappy(t *testing.T) {
	in := `{
  // top bar at the top of the screen
  "layer": "top",
  "position": "top",
  "spacing": 0,
  "height": 26,
  "modules-left": ["custom/omarchy", "hyprland/workspaces"],
  "modules-center": ["clock"],
  "modules-right": ["battery"]
}`
	got, ok := parseWaybarConfig(in)
	if !ok {
		t.Fatal("parse failed")
	}
	want := parsedConfig{
		Position:      "top",
		Layer:         "top",
		Height:        26,
		Spacing:       0,
		ModulesLeft:   []string{"custom/omarchy", "hyprland/workspaces"},
		ModulesCenter: []string{"clock"},
		ModulesRight:  []string{"battery"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("parsed mismatch:\nwant: %+v\ngot:  %+v", want, got)
	}
}

func TestParseWaybarConfigMalformedReturnsZero(t *testing.T) {
	_, ok := parseWaybarConfig(`{ this is not json `)
	if ok {
		t.Error("expected parse to fail on garbage input")
	}
}

func TestParseWaybarConfigIgnoresExtraKeys(t *testing.T) {
	// Real waybar configs have per-module blocks dark doesn't care
	// about. parseWaybarConfig should ignore them without erroring.
	in := `{
  "position": "bottom",
  "height": 30,
  "modules-left": ["custom/omarchy"],
  "custom/omarchy": {
    "format": "icon",
    "on-click": "foo"
  },
  "clock": { "format": "{:%H:%M}" }
}`
	got, ok := parseWaybarConfig(in)
	if !ok {
		t.Fatal("parse failed on config with nested blocks")
	}
	if got.Position != "bottom" || got.Height != 30 {
		t.Errorf("unexpected parse: %+v", got)
	}
	if !strings.HasPrefix(got.ModulesLeft[0], "custom/omarchy") {
		t.Errorf("modules-left mismatch: %+v", got.ModulesLeft)
	}
}
