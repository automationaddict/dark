package tui

import (
	"bytes"
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/johnnelson/dark/internal/theme"
)

// Syntax language identifiers the Editor overlay accepts. Anything
// outside this set routes through the plain-text path.
const (
	LangNone  = ""
	LangJSON  = "json"
	LangJSONC = "jsonc"
	LangCSS   = "css"
)

// syntaxStyle is the chroma.Style derived from the active Omarchy
// palette. Built once at startup by setHighlightPalette (invoked
// from ApplyPalette) and reused for every highlight call. Protected
// by a mutex so a future live-reload path can swap it atomically.
var (
	syntaxStyleMu sync.RWMutex
	syntaxStyle   *chroma.Style
)

// setHighlightPalette rebuilds syntaxStyle from the loaded theme
// palette. Called from ApplyPalette so editor highlighting tracks
// whatever Omarchy theme is active. A chroma built-in is used as
// a last-resort fallback when the palette can't be turned into a
// valid chroma style (shouldn't happen in practice).
func setHighlightPalette(p theme.Palette) {
	entries := paletteToStyleEntries(p)
	s, err := chroma.NewStyle("dark-omarchy", entries)
	syntaxStyleMu.Lock()
	defer syntaxStyleMu.Unlock()
	if err != nil || s == nil {
		syntaxStyle = styles.Get("monokai")
		return
	}
	syntaxStyle = s
}

// paletteToStyleEntries maps Omarchy palette slots to chroma token
// types. The mapping is deliberately conservative: background and
// default foreground mirror the app's existing colors so the editor
// looks like a natural extension of the TUI, and tokens lean on
// Accent / Green / Gold / Muted to carry type distinctions.
//
//   - Comments → Muted (dim, unobtrusive)
//   - Strings → Green (text value)
//   - Numbers → Gold (number value)
//   - Keywords / constants / tag names / attribute names → Accent
//   - Operators / punctuation → Foreground (neutral)
//   - Errors → Red
//
// Every color is emitted as a chroma StyleEntry string which uses
// the #rrggbb syntax. Omarchy palette values already come in that
// form so no conversion is needed.
func paletteToStyleEntries(p theme.Palette) chroma.StyleEntries {
	// Safe fallbacks if the palette is somehow incomplete (e.g.
	// the defaults path). Keeping the output chroma-parseable is
	// more important than matching the user's theme exactly.
	fallback := func(v, def string) string {
		if v == "" {
			return def
		}
		return v
	}

	fg := fallback(p.Foreground, "#c0c0c0")
	bg := fallback(p.Background, "#000000")
	accent := fallback(p.Accent, "#87afff")
	muted := fallback(p.Muted, "#444444")
	green := fallback(p.Green, "#8fbf8f")
	gold := fallback(p.Gold, "#e6c07b")
	red := fallback(p.Red, "#ff5555")

	return chroma.StyleEntries{
		// Background + default text. "Background" is the
		// style-level canvas; Text is the per-token default.
		chroma.Background: fg + " bg:" + bg,
		chroma.Text:       fg,

		// Comments — JSONC // syntax slips through as Error with
		// the json lexer, but block /* */ comments in CSS land
		// here and render dim.
		chroma.Comment:          muted,
		chroma.CommentSingle:    muted,
		chroma.CommentMultiline: muted,
		chroma.CommentPreproc:   muted,

		// Keywords and constants: true / false / null in JSON,
		// !important / @media / @keyframes in CSS.
		chroma.Keyword:         accent + " bold",
		chroma.KeywordConstant: accent + " bold",
		chroma.KeywordReserved: accent + " bold",
		chroma.KeywordType:     accent,

		// Names — JSON keys, CSS tag / class / property names.
		chroma.Name:          fg,
		chroma.NameTag:       accent,
		chroma.NameClass:     accent,
		chroma.NameAttribute: accent,
		chroma.NameFunction:  accent,
		chroma.NameBuiltin:   accent,
		chroma.NameVariable:  fg,

		// Literals — strings and numbers, the main visual split
		// in JSON configs.
		chroma.LiteralString:         green,
		chroma.LiteralStringDouble:   green,
		chroma.LiteralStringSingle:   green,
		chroma.LiteralStringEscape:   green + " bold",
		chroma.LiteralStringInterpol: green + " bold",
		chroma.LiteralNumber:         gold,
		chroma.LiteralNumberInteger:  gold,
		chroma.LiteralNumberFloat:    gold,
		chroma.LiteralNumberHex:      gold,

		// Punctuation / operators — neutral so brace/colon
		// density doesn't flood the view with color noise.
		chroma.Punctuation: fg,
		chroma.Operator:    fg,

		// Errors — chroma uses this for syntax the lexer couldn't
		// resolve. JSONC // comments land here, which reads okay
		// because they're still dim-ish when the palette's Red is
		// muted enough, and visibly different from real tokens.
		chroma.Error: red,
	}
}

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

	style := currentSyntaxStyle()
	if style == nil {
		// Pre-ApplyPalette code path (tests in particular) — fall
		// back to a chroma built-in so highlightLines is still
		// callable without wiring ApplyPalette first.
		style = styles.Get("monokai")
	}

	// terminal16m is chroma's truecolor formatter — it emits the
	// full #rrggbb values from the palette without quantizing to
	// 256 colors. Every terminal Omarchy ships with (ghostty,
	// alacritty, kitty) supports truecolor, so there's no reason
	// to fall back to terminal256 for visual fidelity. We do
	// still fall back if the formatter name ever disappears.
	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Get("terminal256")
	}
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

// currentSyntaxStyle returns the palette-derived chroma.Style
// under a read lock. nil before setHighlightPalette has run.
func currentSyntaxStyle() *chroma.Style {
	syntaxStyleMu.RLock()
	defer syntaxStyleMu.RUnlock()
	return syntaxStyle
}

// lexerFor maps dark's language identifiers to chroma lexers.
// jsonc (JSON with comments) falls back to plain json because
// chroma doesn't ship a dedicated jsonc lexer; the waybar config
// renders almost-correctly this way because chroma colors tokens
// and ignores its own comment-parsing failures (they land as
// chroma.Error which we color via palette.Red).
func lexerFor(lang string) chroma.Lexer {
	switch lang {
	case LangJSON, LangJSONC:
		return lexers.Get("json")
	case LangCSS:
		return lexers.Get("css")
	}
	return nil
}
