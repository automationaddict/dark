package topbar

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Editable scalar kind constants used by SetScalar.
const (
	KindPosition = "position"
	KindLayer    = "layer"
	KindHeight   = "height"
	KindSpacing  = "spacing"
)

// Validation for editable scalar values.
var (
	validPositions = map[string]bool{"top": true, "bottom": true, "left": true, "right": true}
	validLayers    = map[string]bool{"top": true, "overlay": true, "bottom": true, "background": true}
)

// SetPosition / SetLayer / SetHeight / SetSpacing are thin wrappers
// around setScalarKey. They validate the input against waybar's
// documented value range before touching the file — a typo at the
// dialog layer should fail loudly, not corrupt config.jsonc.

func SetPosition(value string) error {
	if !validPositions[value] {
		return fmt.Errorf("position must be top/bottom/left/right, got %q", value)
	}
	return setScalarKey(KindPosition, value, true)
}

func SetLayer(value string) error {
	if !validLayers[value] {
		return fmt.Errorf("layer must be top/overlay/bottom/background, got %q", value)
	}
	return setScalarKey(KindLayer, value, true)
}

func SetHeight(value int) error {
	if value < 0 || value > 200 {
		return fmt.Errorf("height must be 0..200, got %d", value)
	}
	return setScalarKey(KindHeight, strconv.Itoa(value), false)
}

func SetSpacing(value int) error {
	if value < 0 || value > 200 {
		return fmt.Errorf("spacing must be 0..200, got %d", value)
	}
	return setScalarKey(KindSpacing, strconv.Itoa(value), false)
}

// setScalarKey reads config.jsonc, line-patches the named key's
// value, and writes back atomically. Line-anchored editing (rather
// than round-tripping through json.Marshal) preserves comments,
// module ordering, and whitespace — the same philosophy hypridle
// and logind edits follow elsewhere in the codebase.
//
// If the key is already present, the value is replaced.  If it's
// missing, a fresh line is inserted after the top-level opening
// brace so waybar picks it up on the next restart.
//
// quote controls JSON quoting: true wraps the value in "..." (for
// string keys like position/layer), false emits the raw value
// (for numeric keys like height/spacing).
func setScalarKey(kind, value string, quote bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home: %w", err)
	}
	path := filepath.Join(home, configRel)
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	patched, err := patchScalarLine(string(data), kind, value, quote)
	if err != nil {
		return err
	}
	return writeAtomic(path, []byte(patched))
}

// patchScalarLine is exported-for-tests-only: takes the full config
// file body plus a (kind, value) pair and returns a new body with
// the matching line replaced or a new line inserted. Exposed so
// edit_test.go can exercise every branch without touching disk.
func patchScalarLine(body, kind, value string, quote bool) (string, error) {
	rendered := renderScalarLine(kind, value, quote)

	// Look for the existing line. The regex is anchored on
	// whitespace + the quoted key + whitespace + colon so only a
	// top-level (or at least outer-scoped) key matches. Nested
	// blocks like "clock": { "format": ... } won't collide because
	// they don't have "position"/"layer"/"height"/"spacing" as
	// immediate children in any Omarchy config.
	re := regexp.MustCompile(`(?m)^(\s*)"` + regexp.QuoteMeta(kind) + `"\s*:\s*[^,\n]*,?\s*$`)
	if re.MatchString(body) {
		replaced := re.ReplaceAllStringFunc(body, func(match string) string {
			// Preserve the existing indent. The trailing comma is
			// re-added if the original line had one so the
			// surrounding JSON still parses.
			indent := leadingWhitespace(match)
			trailing := ""
			if strings.HasSuffix(strings.TrimRight(match, " \t"), ",") {
				trailing = ","
			}
			return indent + `"` + kind + `": ` + quotedOrRaw(value, quote) + trailing
		})
		_ = rendered
		return replaced, nil
	}

	// Key not present — insert after the first top-level `{`.
	idx := strings.Index(body, "{")
	if idx < 0 {
		return "", fmt.Errorf("config has no opening brace")
	}
	// Find the end of the line containing the brace so we insert
	// on the next line with the right indent.
	lineEnd := strings.Index(body[idx:], "\n")
	if lineEnd < 0 {
		return "", fmt.Errorf("config has no newline after opening brace")
	}
	insertAt := idx + lineEnd + 1

	// Indent detection: look at the next non-blank line and copy
	// its leading whitespace, or fall back to two spaces.
	indent := "  "
	if insertAt < len(body) {
		rest := body[insertAt:]
		nl := strings.Index(rest, "\n")
		var firstLine string
		if nl < 0 {
			firstLine = rest
		} else {
			firstLine = rest[:nl]
		}
		if trimmed := strings.TrimLeft(firstLine, " \t"); trimmed != "" {
			indent = firstLine[:len(firstLine)-len(trimmed)]
		}
	}

	inserted := indent + `"` + kind + `": ` + quotedOrRaw(value, quote) + ",\n"
	return body[:insertAt] + inserted + body[insertAt:], nil
}

// renderScalarLine is a debugging aid kept around so the editor can
// render a "what would be written" preview if we ever add one.
func renderScalarLine(kind, value string, quote bool) string {
	return `"` + kind + `": ` + quotedOrRaw(value, quote)
}

func quotedOrRaw(value string, quote bool) string {
	if quote {
		return `"` + value + `"`
	}
	return value
}

func leadingWhitespace(s string) string {
	trimmed := strings.TrimLeft(s, " \t")
	return s[:len(s)-len(trimmed)]
}
