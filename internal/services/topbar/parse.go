package topbar

import (
	"encoding/json"
	"strings"
)

// parsedConfig is the subset of waybar config.jsonc we surface on
// the snapshot. Anything not listed here is left untouched by dark's
// editors and is invisible to the UI — the full-screen editor is
// the escape hatch for everything else.
type parsedConfig struct {
	Position      string
	Layer         string
	Height        int
	Spacing       int
	ModulesLeft   []string
	ModulesCenter []string
	ModulesRight  []string
}

// parseWaybarConfig decodes the scalar top-level keys dark cares
// about. The config file is JSON-with-comments, which encoding/json
// rejects, so we strip line and block comments first. On any decode
// failure we return (zero, false) — the caller treats that as "no
// parsed data" and the UI shows "—" for every row.
func parseWaybarConfig(content string) (parsedConfig, bool) {
	cleaned := stripJSONCComments(content)

	var raw struct {
		Position      string          `json:"position"`
		Layer         string          `json:"layer"`
		Height        int             `json:"height"`
		Spacing       int             `json:"spacing"`
		ModulesLeft   []string        `json:"modules-left"`
		ModulesCenter []string        `json:"modules-center"`
		ModulesRight  []string        `json:"modules-right"`
		Extra         map[string]json.RawMessage `json:"-"`
	}
	if err := json.Unmarshal([]byte(cleaned), &raw); err != nil {
		return parsedConfig{}, false
	}
	return parsedConfig{
		Position:      raw.Position,
		Layer:         raw.Layer,
		Height:        raw.Height,
		Spacing:       raw.Spacing,
		ModulesLeft:   raw.ModulesLeft,
		ModulesCenter: raw.ModulesCenter,
		ModulesRight:  raw.ModulesRight,
	}, true
}

// stripJSONCComments removes // line comments and /* */ block
// comments while leaving string literals untouched. encoding/json
// will then accept the result. This is a minimal JSONC stripper —
// no escape-handling for backslash-quotes inside strings, because
// waybar's config uses nothing fancier than plain double-quoted
// strings. If that ever changes we'll swap to tailscale/hujson.
func stripJSONCComments(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	i := 0
	for i < len(s) {
		c := s[i]
		if inString {
			b.WriteByte(c)
			if c == '\\' && i+1 < len(s) {
				// Copy the escaped byte verbatim so a \" doesn't
				// close the string prematurely.
				b.WriteByte(s[i+1])
				i += 2
				continue
			}
			if c == '"' {
				inString = false
			}
			i++
			continue
		}
		if c == '"' {
			inString = true
			b.WriteByte(c)
			i++
			continue
		}
		if c == '/' && i+1 < len(s) {
			next := s[i+1]
			if next == '/' {
				// Skip to end of line but keep the newline so
				// line numbers in downstream errors stay stable.
				end := strings.IndexByte(s[i:], '\n')
				if end < 0 {
					return b.String()
				}
				i += end
				continue
			}
			if next == '*' {
				end := strings.Index(s[i+2:], "*/")
				if end < 0 {
					return b.String()
				}
				i += end + 4
				continue
			}
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}
