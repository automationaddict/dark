package topbar

import "testing"

// TestPatchRoundTripThroughParser is the integration test: we patch
// a scalar, feed the result back through parseWaybarConfig, and
// confirm the parser sees the new value. Catches bugs where the
// editor produces technically-valid JSON that the stripper or the
// decoder mishandles.
func TestPatchRoundTripThroughParser(t *testing.T) {
	cases := []struct {
		name  string
		kind  string
		value string
		quote bool
		check func(t *testing.T, c parsedConfig)
	}{
		{"position", "position", "bottom", true, func(t *testing.T, c parsedConfig) {
			if c.Position != "bottom" {
				t.Errorf("Position = %q", c.Position)
			}
		}},
		{"layer", "layer", "overlay", true, func(t *testing.T, c parsedConfig) {
			if c.Layer != "overlay" {
				t.Errorf("Layer = %q", c.Layer)
			}
		}},
		{"height", "height", "42", false, func(t *testing.T, c parsedConfig) {
			if c.Height != 42 {
				t.Errorf("Height = %d", c.Height)
			}
		}},
		{"spacing", "spacing", "8", false, func(t *testing.T, c parsedConfig) {
			if c.Spacing != 8 {
				t.Errorf("Spacing = %d", c.Spacing)
			}
		}},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			patched, err := patchScalarLine(sampleConfig, tt.kind, tt.value, tt.quote)
			if err != nil {
				t.Fatal(err)
			}
			got, ok := parseWaybarConfig(patched)
			if !ok {
				t.Fatalf("parser rejected patched config:\n%s", patched)
			}
			tt.check(t, got)
		})
	}
}
