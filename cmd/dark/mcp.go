package main

import (
	"fmt"
	"os"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/mcp"
	"github.com/automationaddict/dark/internal/services/sysinfo"
)

// runMcpSubcommand handles the `dark mcp` one-shot that turns the
// running daemon into a Model Context Protocol server speaking JSON-
// RPC over stdio. MCP hosts like Claude Desktop spawn this binary per
// session and pipe messages in/out; when the host closes the pipes,
// ServeStdio returns and we exit.
//
// Skips the TUI and the process lock so it runs alongside an active
// dark session the same way `dark script` does. No arguments are
// supported today — every tool and resource is derived automatically
// from bus.APICommandCatalog.
func runMcpSubcommand(args []string) int {
	if len(args) > 0 {
		switch args[0] {
		case "-h", "--help":
			mcpUsage()
			return 0
		default:
			fmt.Fprintf(os.Stderr, "dark mcp: unknown argument %q\n", args[0])
			mcpUsage()
			return 2
		}
	}

	// Use a short client name so the daemon log distinguishes MCP
	// sessions from the main TUI. ConnectClient writes a helpful
	// "not running" error when the daemon is down, which stdio
	// hosts surface to the user as a startup failure.
	nc, err := bus.ConnectClient("dark-mcp", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark mcp:", err)
		return 1
	}
	defer nc.Drain()

	if err := mcp.Serve(nc, sysinfo.DarkVersion); err != nil {
		fmt.Fprintln(os.Stderr, "dark mcp:", err)
		return 1
	}
	return 0
}

func mcpUsage() {
	fmt.Fprintln(os.Stderr, "Usage: dark mcp")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Run an MCP JSON-RPC server over stdio that proxies dark's")
	fmt.Fprintln(os.Stderr, "command catalog as MCP tools and snapshot subjects as MCP")
	fmt.Fprintln(os.Stderr, "resources. Intended to be launched by an MCP host such as")
	fmt.Fprintln(os.Stderr, "Claude Desktop — not used interactively.")
}
