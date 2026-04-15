package tui

import (
	"strings"
	"testing"
)

// escByte is the ANSI CSI introducer. A highlighted line from
// chroma's terminal256 formatter always contains at least one.
const escByte = "\x1b"

func TestHighlightLinesJSON(t *testing.T) {
	content := `{
  "position": "top",
  "height": 26
}`
	lines, ok := highlightLines(content, LangJSON)
	if !ok {
		t.Fatal("highlightLines(json) returned ok=false")
	}
	if got, want := len(lines), 4; got != want {
		t.Errorf("line count = %d, want %d", got, want)
	}
	// At least the line with a string literal must carry an ANSI
	// escape — otherwise chroma isn't actually coloring.
	if !strings.Contains(lines[1], escByte) {
		t.Errorf("expected ANSI escape in line 1 (%q)", lines[1])
	}
}

func TestHighlightLinesJSONC(t *testing.T) {
	// JSONC goes through the json lexer — comments will be mis-
	// tokenized but the content bytes still come through.
	content := `{
  // waybar config
  "layer": "top"
}`
	lines, ok := highlightLines(content, LangJSONC)
	if !ok {
		t.Fatal("highlightLines(jsonc) returned ok=false")
	}
	if len(lines) != 4 {
		t.Errorf("line count = %d, want 4", len(lines))
	}
}

func TestHighlightLinesCSS(t *testing.T) {
	content := `* {
  background-color: #1a1b26;
  font-family: "JetBrains Mono";
}`
	lines, ok := highlightLines(content, LangCSS)
	if !ok {
		t.Fatal("highlightLines(css) returned ok=false")
	}
	if len(lines) != 4 {
		t.Errorf("line count = %d, want 4", len(lines))
	}
	// Keywords / selectors should produce ANSI.
	hasEscape := false
	for _, line := range lines {
		if strings.Contains(line, escByte) {
			hasEscape = true
			break
		}
	}
	if !hasEscape {
		t.Error("no ANSI escapes produced for CSS — chroma not coloring")
	}
}

func TestHighlightLinesUnknownLanguage(t *testing.T) {
	_, ok := highlightLines("hello", "zig")
	if ok {
		t.Error("unknown language should return ok=false so caller falls back to plain")
	}
}

func TestHighlightLinesEmpty(t *testing.T) {
	lines, ok := highlightLines("", LangJSON)
	if !ok {
		t.Fatal("empty content should still highlight")
	}
	if len(lines) != 1 {
		t.Errorf("empty content should produce 1 line, got %d", len(lines))
	}
}

func TestLineNumberWidth(t *testing.T) {
	cases := map[int]int{
		0:    2,
		1:    2,
		9:    2,
		10:   2,
		99:   2,
		100:  3,
		999:  3,
		1000: 4,
	}
	for in, want := range cases {
		if got := lineNumberWidth(in); got != want {
			t.Errorf("lineNumberWidth(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestLineNumberCell(t *testing.T) {
	cases := []struct {
		n, width int
		want     string
	}{
		{1, 3, "  1"},
		{42, 3, " 42"},
		{999, 3, "999"},
		{7, 2, " 7"},
	}
	for _, tt := range cases {
		if got := lineNumberCell(tt.n, tt.width); got != tt.want {
			t.Errorf("lineNumberCell(%d, %d) = %q, want %q", tt.n, tt.width, got, tt.want)
		}
	}
}
