package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mcpgo "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
)

// defaultRequestTimeout bounds how long a tool / resource call waits
// for darkd to reply over NATS. Tools like wifi_scan or update_run
// can take a while on real hardware, so the timeout sits on the
// generous side of normal.
const defaultRequestTimeout = 30 * time.Second

// Serve connects the provided NATS client to an MCP server that
// proxies every dark.cmd.* subject as a tool and every snapshot
// subject as a resource, then runs the server on stdio until the
// transport closes. Used by the `dark mcp` subcommand.
func Serve(nc *nats.Conn, version string) error {
	s := NewServer(nc, version)
	return server.ServeStdio(s)
}

// NewServer builds a fully-populated MCP server without starting the
// stdio loop. Split out so tests can drive the server directly.
func NewServer(nc *nats.Conn, version string) *server.MCPServer {
	s := server.NewMCPServer(
		"dark",
		version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)
	registerTools(s, nc)
	registerResources(s, nc)
	return s
}

// registerTools installs one MCP tool per bus command subject. The
// handler is a thin shim that reads the LLM's arguments back into a
// JSON payload and forwards the request over NATS.
func registerTools(s *server.MCPServer, nc *nats.Conn) {
	for _, t := range Tools() {
		tool := mcpgo.NewTool(t.Name, toolOptions(t.Summary, t.Fields)...)
		s.AddTool(tool, makeToolHandler(nc, t.Subject, t.Fields))
	}
}

// registerResources installs one MCP resource per curated snapshot
// subject. Resource handlers issue the zero-payload command and
// return the raw JSON reply as text content.
func registerResources(s *server.MCPServer, nc *nats.Conn) {
	for _, r := range Resources() {
		res := mcpgo.NewResource(
			r.URI,
			r.Name,
			mcpgo.WithResourceDescription(r.Summary),
			mcpgo.WithMIMEType("application/json"),
		)
		s.AddResource(res, makeResourceHandler(nc, r.Subject))
	}
}

// toolOptions builds the mcp-go tool option slice from a bus command
// schema. Every CommandField maps onto one of mcp-go's typed property
// builders so the LLM host gets a real JSON Schema to validate
// against before it calls the tool.
func toolOptions(summary string, fields []bus.CommandField) []mcpgo.ToolOption {
	opts := []mcpgo.ToolOption{mcpgo.WithDescription(summary)}
	for _, f := range fields {
		var props []mcpgo.PropertyOption
		if f.Required {
			props = append(props, mcpgo.Required())
		}
		if f.Desc != "" {
			props = append(props, mcpgo.Description(f.Desc))
		}
		switch f.Type {
		case "bool":
			opts = append(opts, mcpgo.WithBoolean(f.Name, props...))
		case "int", "float":
			opts = append(opts, mcpgo.WithNumber(f.Name, props...))
		case "[]string":
			opts = append(opts, mcpgo.WithArray(f.Name, append(props, mcpgo.WithStringItems())...))
		case "table":
			opts = append(opts, mcpgo.WithObject(f.Name, props...))
		default: // "string" and anything else
			opts = append(opts, mcpgo.WithString(f.Name, props...))
		}
	}
	return opts
}

// makeToolHandler returns the ToolHandlerFunc for one bus subject.
// The returned closure reads arguments from the MCP request, rebuilds
// the JSON payload darkd expects, issues a NATS request, and returns
// the reply as MCP text content. Errors are reported as MCP tool
// errors rather than Go errors so the LLM sees a structured failure.
func makeToolHandler(nc *nats.Conn, subject string, fields []bus.CommandField) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcpgo.CallToolRequest) (*mcpgo.CallToolResult, error) {
		args := req.GetArguments()
		payload := map[string]any{}
		for _, f := range fields {
			v, ok := args[f.Name]
			if !ok || v == nil {
				if f.Required {
					return mcpgo.NewToolResultError(
						fmt.Sprintf("missing required field %q", f.Name)), nil
				}
				continue
			}
			// MCP "number" values arrive as float64; coerce to int
			// when the bus schema wants an integer so darkd's
			// typed decoder accepts the payload.
			if f.Type == "int" {
				if fv, ok := v.(float64); ok {
					payload[f.Name] = int(fv)
					continue
				}
			}
			payload[f.Name] = v
		}

		var data []byte
		if len(payload) > 0 {
			b, err := json.Marshal(payload)
			if err != nil {
				return mcpgo.NewToolResultError(
					fmt.Sprintf("marshal payload: %s", err)), nil
			}
			data = b
		}

		reply, err := nc.Request(subject, data, defaultRequestTimeout)
		if err != nil {
			return mcpgo.NewToolResultError(err.Error()), nil
		}
		text := string(reply.Data)
		if text == "" {
			text = "{}"
		}
		return mcpgo.NewToolResultText(text), nil
	}
}

// makeResourceHandler returns the ResourceHandlerFunc for one
// snapshot subject. Resources are read-only, so the handler always
// issues a zero-payload request and returns the JSON reply verbatim.
func makeResourceHandler(nc *nats.Conn, subject string) server.ResourceHandlerFunc {
	return func(ctx context.Context, req mcpgo.ReadResourceRequest) ([]mcpgo.ResourceContents, error) {
		reply, err := nc.Request(subject, nil, defaultRequestTimeout)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", subject, err)
		}
		text := string(reply.Data)
		if text == "" {
			text = "{}"
		}
		return []mcpgo.ResourceContents{
			mcpgo.TextResourceContents{
				URI:      req.Params.URI,
				MIMEType: "application/json",
				Text:     text,
			},
		}, nil
	}
}
