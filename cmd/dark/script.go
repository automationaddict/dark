package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
)

// runScriptSubcommand handles the `dark script ...` one-shot CLI.
// It skips the TUI and the process lock so it can run alongside a
// long-lived dark session — exactly the shape you want for wiring
// Hyprland keybindings to Lua helpers.
//
// Supported forms:
//
//	dark script call <fn> [arg1 arg2 ...]
//
// Positional arguments are parsed as JSON literals when possible
// (numbers, booleans, null, arrays, objects) and fall back to plain
// strings otherwise, so `dark script call volume_set 75` sends the
// number 75 and `dark script call set_name alice` sends the string
// "alice" — no quoting ceremony required for the common cases.
func runScriptSubcommand(args []string) int {
	if len(args) == 0 {
		scriptUsage()
		return 2
	}
	switch args[0] {
	case "call":
		return scriptCall(args[1:])
	case "-h", "--help", "help":
		scriptUsage()
		return 0
	}
	fmt.Fprintln(os.Stderr, "dark script: unknown subcommand:", args[0])
	scriptUsage()
	return 2
}

func scriptUsage() {
	fmt.Println("usage: dark script call <fn> [arg1 arg2 ...]")
	fmt.Println()
	fmt.Println("Invoke a Lua global function defined by a user script.")
	fmt.Println("Arguments are parsed as JSON literals when possible.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  dark script call volume_up")
	fmt.Println("  dark script call volume_set 75")
	fmt.Println("  dark script call notify '\"hello\"'")
}

func scriptCall(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "dark script call: missing function name")
		return 2
	}
	fn := args[0]
	parsed := make([]interface{}, 0, len(args)-1)
	for _, raw := range args[1:] {
		var v interface{}
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			parsed = append(parsed, v)
		} else {
			parsed = append(parsed, raw)
		}
	}
	payload, _ := json.Marshal(map[string]interface{}{
		"fn":   fn,
		"args": parsed,
	})

	nc, err := bus.ConnectClient("dark-script", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark script:", err)
		return 1
	}
	defer nc.Drain()

	reply, err := nc.Request(bus.SubjectScriptingCallCmd, payload, core.TimeoutPkexec)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark script:", err)
		return 1
	}
	var resp struct {
		Fn     string      `json:"fn"`
		Result interface{} `json:"result,omitempty"`
		Error  string      `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		fmt.Fprintln(os.Stderr, "dark script: decode reply:", err)
		return 1
	}
	if resp.Error != "" {
		fmt.Fprintln(os.Stderr, "dark script:", resp.Error)
		return 1
	}
	if resp.Result != nil {
		out, _ := json.Marshal(resp.Result)
		fmt.Println(string(out))
	}
	return 0
}
