package tui

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// Syntax language identifiers the Editor overlay accepts. Anything
// outside this set routes through the plain-text path.
const (
	LangNone = ""
	LangJSON = "json"
	LangJSONC = "jsonc"
	LangCSS  = "css"
)

// highlightTheme picks the chroma style that maps best to dark's
// Omarchy-derived palette. monokai has good contrast on dark
// backgrounds and punctuation that doesn't fade out. Users on
// light themes (which Omarchy doesn't ship but someone might use
// via a custom theme) get the same colors — chroma styles aren't
// theme-derived yet.
const highlightTheme = "monokai"

// highlightLines tokenizes content with the chroma lexer for lang
// and returns one pre-formatted ANSI string per line. Length of
// the returned slice always matches strings.Split(content, "\n")
// even when a line would otherwise be empty, so callers can use
// line index from the editor directly.
//
// Returns (nil, false) when the lexer isn't available or chroma
// errors out. Callers fall back to plain text in that case.
func highlightLines(content, lang string) ([]string, bool) {
	lexer := lexerFor(lang)
	if lexer == nil {
		return nil, false
	}

	style := styles.Get(highlightTheme)
	if style == nil {
		style = styles.Fallback
	}

	// Use the 256-color formatter for broadest terminal compat.
	// terminal256 emits ESC[38;5;... sequences that every modern
	// emulator supports; chroma's truecolor formatter would also
	// work but offers no visible benefit at syntax-highlight
	// granularity.
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iter, err := lexer.Tokenise(nil, content)
	if err != nil {
		return nil, false
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iter); err != nil {
		return nil, false
	}

	highlighted := buf.String()
	// Split the ANSI-annotated output the same way we'd split the
	// original content. Chroma preserves newlines in its output,
	// so line counts line up.
	lines := strings.Split(highlighted, "\n")

	// Align slice length to the original content's line count in
	// case chroma collapses a trailing newline (it sometimes does
	// for files that don't end in a newline).
	expected := strings.Count(content, "\n") + 1
	for len(lines) < expected {
		lines = append(lines, "")
	}
	if len(lines) > expected {
		lines = lines[:expected]
	}
	return lines, true
}

// lexerFor maps dark's language identifiers to chroma lexers.
// jsonc (JSON with comments) falls back to plain json because
// chroma doesn't ship a dedicated jsonc lexer; the waybar config
// renders almost-correctly this way because chroma colors tokens
// and silently ignores its own comment-parsing failures.
func lexerFor(lang string) chroma.Lexer {
	switch lang {
	case LangJSON, LangJSONC:
		return lexers.Get("json")
	case LangCSS:
		return lexers.Get("css")
	}
	return nil
}
