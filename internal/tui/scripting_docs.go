package tui

import (
	"fmt"
	"strings"

	"github.com/automationaddict/dark/internal/core"
)

// This file owns the markdown source builders for the F5 Scripting
// content pane. Each function returns a markdown string that gets
// glamour-rendered through help.RenderMarkdown at display time. Keep
// these strings small and structured — glamour is happy with
// headings, inline code, lists, and fenced code blocks.

func newScriptMarkdown() string {
	return strings.Join([]string{
		"# New Lua script",
		"",
		"Press **Enter** on this row to create a new `.lua` file in " +
			"`~/.config/dark/scripts/`. A name prompt appears, then the " +
			"full-screen editor opens with an empty buffer.",
		"",
		"Save with **Ctrl+S** to write the file. darkd reloads every " +
			"user script after a save so any `dark.on(...)` hooks you " +
			"declare take effect immediately.",
	}, "\n")
}

// mcpMarkdown renders the doc pane for the currently-selected entry
// in the F5 MCP sub-nav. Tools get the same parameter table / call
// example treatment the API tab uses; resources get a read-only view
// showing the URI and the snapshot subject they proxy.
func mcpMarkdown(s *core.State) string {
	if !s.MCPCatalogLoaded {
		return "# MCP\n\nLoading catalog…"
	}
	if s.MCPEntryCount() == 0 {
		return strings.Join([]string{
			"# MCP",
			"",
			"No tools or resources are registered. This usually means " +
				"the daemon didn't ship with any `dark.cmd.*` subjects — " +
				"check the logs for startup errors.",
		}, "\n")
	}
	if tool, ok := s.SelectedMCPTool(); ok {
		return mcpToolMarkdown(tool)
	}
	if res, ok := s.SelectedMCPResource(); ok {
		return mcpResourceMarkdown(res)
	}
	return "# MCP\n\nSelect an entry from the sidebar."
}

func mcpToolMarkdown(t core.MCPToolEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", t.Name)
	b.WriteString("**Kind:** Tool  \n")
	fmt.Fprintf(&b, "**Subject:** `%s`  \n", t.Subject)
	fmt.Fprintf(&b, "**Domain:** `%s`\n\n", t.Domain)
	if t.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", t.Summary)
	}
	writeFieldsTable(&b, t.Fields)
	b.WriteString("## MCP call\n\n")
	fmt.Fprintf(&b, "```json\n%s\n```\n\n", mcpCallExample(t.Name, t.Fields))
	b.WriteString("## Lua equivalent\n\n")
	luaName := "dark.actions." + t.Domain + "." + t.Verb
	fmt.Fprintf(&b, "```lua\n%s\n```\n", luaCallExample(luaName, t.Fields))
	return b.String()
}

func mcpResourceMarkdown(r core.MCPResourceEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", r.Name)
	b.WriteString("**Kind:** Resource  \n")
	fmt.Fprintf(&b, "**URI:** `%s`  \n", r.URI)
	fmt.Fprintf(&b, "**Subject:** `%s`\n\n", r.Subject)
	if r.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", r.Summary)
	}
	b.WriteString("Resources are read-only snapshots. An MCP host can " +
		"call `resources/read` with this URI to fetch the current " +
		"JSON payload.\n\n")
	b.WriteString("## Raw NATS\n\n")
	fmt.Fprintf(&b, "```\nnats req %s ''\n```\n", r.Subject)
	return b.String()
}

// mcpCallExample emits a minimal MCP tools/call JSON body with just
// the required fields populated. The LLM host will normally build
// this for the user, but showing it here helps scripts and operators
// debug with `jq` / stdio directly.
func mcpCallExample(name string, fields []core.CommandField) string {
	args := jsonPayloadExample(fields)
	return "{\n  \"name\": \"" + name + "\",\n  \"arguments\": " + args + "\n}"
}

func luaEntryMarkdown(s *core.State) string {
	if len(s.LuaRegistry) == 0 {
		return "# Lua\n\nLoading registry…"
	}
	idx := s.LuaInnerIdx
	if idx < 0 || idx >= len(s.LuaRegistry) {
		return "# Lua\n\nSelect an entry from the sidebar."
	}
	e := s.LuaRegistry[idx]
	kindLabel := "Function"
	switch e.Kind {
	case "event":
		kindLabel = "Event"
	case "action":
		kindLabel = "Action"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", e.Name)
	fmt.Fprintf(&b, "**Kind:** %s  \n", kindLabel)
	if e.Subject != "" {
		fmt.Fprintf(&b, "**Subject:** `%s`  \n", e.Subject)
	}
	if e.Args != "" {
		fmt.Fprintf(&b, "**Signature:** `%s%s`\n\n", e.Name, e.Args)
	} else {
		b.WriteString("\n")
	}
	if e.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", e.Summary)
	}
	switch e.Kind {
	case "function":
		fmt.Fprintf(&b, "## Example\n\n")
		fmt.Fprintf(&b, "```lua\n%s%s\n```\n", e.Name, argsExample(e.Args))
	case "event":
		fmt.Fprintf(&b, "## Example\n\n")
		fmt.Fprintf(&b, "```lua\ndark.on(%q, function%s\n  dark.log(\"hook fired\")\nend)\n```\n",
			e.Name, e.Args)
	case "action":
		writeFieldsTable(&b, e.Fields)
		fmt.Fprintf(&b, "## Example\n\n")
		fmt.Fprintf(&b, "```lua\n%s\n```\n", luaCallExample(e.Name, e.Fields))
		if e.Subject != "" {
			fmt.Fprintf(&b, "\n## Raw NATS\n\n```\nnats req %s '%s'\n```\n",
				e.Subject, jsonPayloadExample(e.Fields))
		}
	}
	return b.String()
}

// writeFieldsTable emits a markdown parameters section when the
// action has a documented schema. Required fields are bolded so the
// reader can tell at a glance what must be supplied.
func writeFieldsTable(b *strings.Builder, fields []core.CommandField) {
	if len(fields) == 0 {
		b.WriteString("## Parameters\n\n_This action takes no payload — call with `{}` or no arguments._\n\n")
		return
	}
	b.WriteString("## Parameters\n\n")
	for _, f := range fields {
		req := "optional"
		if f.Required {
			req = "**required**"
		}
		fmt.Fprintf(b, "- `%s` *(%s, %s)* — %s\n", f.Name, f.Type, req, f.Desc)
	}
	b.WriteString("\n")
}

// luaCallExample produces a realistic Lua call snippet for an action,
// filling every required field with a typed placeholder. Optional
// fields are shown as commented-out lines so scripts can uncomment
// what they need.
func luaCallExample(name string, fields []core.CommandField) string {
	if len(fields) == 0 {
		return "local reply, err = " + name + "()\n" +
			"if err then dark.log(\"" + name + ": \" .. err) end"
	}
	var b strings.Builder
	b.WriteString("local reply, err = " + name + "({\n")
	for _, f := range fields {
		if f.Required {
			fmt.Fprintf(&b, "  %s = %s,\n", f.Name, luaPlaceholder(f.Type, f.Name))
		}
	}
	hasOptional := false
	for _, f := range fields {
		if !f.Required {
			if !hasOptional {
				b.WriteString("  -- optional:\n")
				hasOptional = true
			}
			fmt.Fprintf(&b, "  -- %s = %s,\n", f.Name, luaPlaceholder(f.Type, f.Name))
		}
	}
	b.WriteString("})\nif err then dark.log(\"" + name + ": \" .. err) end")
	return b.String()
}

// jsonPayloadExample builds a minimal JSON object containing just the
// required fields so `nats req` copy-paste works without editing.
func jsonPayloadExample(fields []core.CommandField) string {
	if len(fields) == 0 {
		return "{}"
	}
	var parts []string
	for _, f := range fields {
		if !f.Required {
			continue
		}
		parts = append(parts, fmt.Sprintf("%q:%s", f.Name, jsonPlaceholder(f.Type)))
	}
	if len(parts) == 0 {
		return "{}"
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// luaPlaceholder returns a syntactically-valid Lua literal whose
// shape hints at the expected type. Strings pull from the field
// name for context ("adapter" → "\"adapter\"").
func luaPlaceholder(typ, name string) string {
	switch typ {
	case "bool":
		return "true"
	case "int":
		return "0"
	case "float":
		return "1.0"
	case "[]string":
		return "{ \"...\" }"
	case "table":
		return "{}"
	default:
		return fmt.Sprintf("%q", name)
	}
}

func jsonPlaceholder(typ string) string {
	switch typ {
	case "bool":
		return "true"
	case "int", "float":
		return "0"
	case "[]string":
		return "[\"\"]"
	case "table":
		return "{}"
	default:
		return "\"\""
	}
}

// argsExample turns "(path)" into "(\"...\")" so copy-pasting a
// signature yields a syntactically valid call even before the user
// replaces the placeholder.
func argsExample(args string) string {
	if args == "" {
		return "()"
	}
	if strings.Contains(args, ",") || strings.Contains(args, "(") {
		return strings.ReplaceAll(strings.ReplaceAll(args, "(", "(\""), ")", "\")")
	}
	return "(\"...\")"
}

func apiEntryMarkdown(s *core.State) string {
	if len(s.APICommands) == 0 {
		return "# API\n\nLoading catalog…"
	}
	idx := s.APIInnerIdx
	if idx < 0 || idx >= len(s.APICommands) {
		return "# API\n\nSelect a command from the sidebar."
	}
	c := s.APICommands[idx]
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", c.Subject)
	fmt.Fprintf(&b, "**Domain:** `%s`  \n", c.Domain)
	fmt.Fprintf(&b, "**Verb:** `%s`\n\n", c.Verb)
	if c.Summary != "" {
		fmt.Fprintf(&b, "%s\n\n", c.Summary)
	}
	writeFieldsTable(&b, c.Fields)
	fmt.Fprintf(&b, "## NATS round-trip\n\n")
	fmt.Fprintf(&b, "```\nnats req %s '%s'\n```\n\n", c.Subject, jsonPayloadExample(c.Fields))
	fmt.Fprintf(&b, "## Lua equivalent\n\n")
	luaName := "dark.actions." + c.Domain + "." + c.Verb
	fmt.Fprintf(&b, "```lua\n%s\n```\n", luaCallExample(luaName, c.Fields))
	return b.String()
}
