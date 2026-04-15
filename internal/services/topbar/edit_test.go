package topbar

import (
	"strings"
	"testing"
)

const sampleConfig = `{
  "reload_style_on_change": true,
  "layer": "top",
  "position": "top",
  "spacing": 0,
  "height": 26,
  "modules-left": ["custom/omarchy"],
  "modules-center": ["clock"],
  "modules-right": ["battery"],
  "clock": {
    "format": "{:%H:%M}"
  }
}`

func TestPatchScalarLineReplaceString(t *testing.T) {
	out, err := patchScalarLine(sampleConfig, "position", "bottom", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"position": "bottom"`) {
		t.Errorf("expected new position line, got:\n%s", out)
	}
	if strings.Contains(out, `"position": "top"`) {
		t.Errorf("old position line should be gone:\n%s", out)
	}
	// Make sure we didn't break anything else.
	if !strings.Contains(out, `"layer": "top"`) {
		t.Errorf("layer line lost: %s", out)
	}
	if !strings.Contains(out, `"clock": {`) {
		t.Errorf("clock block lost: %s", out)
	}
}

func TestPatchScalarLineReplaceInt(t *testing.T) {
	out, err := patchScalarLine(sampleConfig, "height", "40", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"height": 40`) {
		t.Errorf("expected new height line, got:\n%s", out)
	}
	if strings.Contains(out, `"height": 26`) {
		t.Errorf("old height line should be gone:\n%s", out)
	}
}

func TestPatchScalarLinePreservesTrailingComma(t *testing.T) {
	// All four patched keys have trailing commas in sampleConfig.
	// The replacement must keep that comma or the JSON breaks.
	out, err := patchScalarLine(sampleConfig, "spacing", "4", false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"spacing": 4,`) {
		t.Errorf("trailing comma lost:\n%s", out)
	}
}

func TestPatchScalarLineInsertsWhenMissing(t *testing.T) {
	configWithoutLayer := `{
  "position": "top",
  "height": 26,
  "modules-left": ["clock"]
}`
	out, err := patchScalarLine(configWithoutLayer, "layer", "overlay", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"layer": "overlay"`) {
		t.Errorf("missing key not inserted:\n%s", out)
	}
	// The inserted line should appear before the position line
	// because we insert right after the opening brace.
	layerIdx := strings.Index(out, `"layer"`)
	posIdx := strings.Index(out, `"position"`)
	if layerIdx < 0 || posIdx < 0 || layerIdx >= posIdx {
		t.Errorf("layer should be inserted before position; got:\n%s", out)
	}
}

func TestPatchScalarLineIsIdempotent(t *testing.T) {
	// Two passes should produce the same result as one.
	once, err := patchScalarLine(sampleConfig, "position", "left", true)
	if err != nil {
		t.Fatal(err)
	}
	twice, err := patchScalarLine(once, "position", "left", true)
	if err != nil {
		t.Fatal(err)
	}
	if once != twice {
		t.Errorf("patching twice should be a no-op.\nonce:\n%s\ntwice:\n%s", once, twice)
	}
}

func TestPatchScalarLinePreservesComments(t *testing.T) {
	in := `{
  // bar at the top
  "position": "top",
  "height": 26
}`
	out, err := patchScalarLine(in, "position", "bottom", true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "// bar at the top") {
		t.Errorf("comment lost:\n%s", out)
	}
}

func TestSetPositionValidates(t *testing.T) {
	if err := SetPosition("diagonal"); err == nil {
		t.Error("expected error for invalid position")
	}
}

func TestSetLayerValidates(t *testing.T) {
	if err := SetLayer("float"); err == nil {
		t.Error("expected error for invalid layer")
	}
}

func TestSetHeightRange(t *testing.T) {
	if err := SetHeight(-1); err == nil {
		t.Error("negative height should fail")
	}
	if err := SetHeight(500); err == nil {
		t.Error("height > 200 should fail")
	}
}

func TestSetSpacingRange(t *testing.T) {
	if err := SetSpacing(-5); err == nil {
		t.Error("negative spacing should fail")
	}
}
