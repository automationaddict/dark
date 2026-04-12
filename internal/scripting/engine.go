// Package scripting owns the embedded Lua VM that powers dark's
// configuration and extension layer. The daemon creates one Engine at
// startup and passes it to services that need scriptable behavior.
//
// Scripts live in two places:
//
//  1. Embedded defaults compiled into the binary (internal/scripting/scripts/).
//     These ship the baseline behavior and are always available.
//  2. User overrides at $XDG_CONFIG_HOME/dark/scripts/. A user file at the
//     same relative path as an embedded default replaces it entirely — the
//     engine does not merge them.
//
// The Engine is designed for a long-lived daemon process. The Lua VM stays
// open for the daemon's lifetime so scripts loaded at startup can expose
// functions that are called repeatedly (e.g. on every catalog rebuild).
// Tier 2/3 work will add hook registration, NATS bus bindings, and MCP
// endpoints on top of this same VM.
package scripting

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	lua "github.com/yuin/gopher-lua"
)

//go:embed scripts
var defaultScripts embed.FS

// Engine wraps a GopherLua VM and manages script loading. It is safe to
// call from multiple goroutines — all VM access is serialized through a
// mutex. This is the right tradeoff for a config/extension layer where
// calls are infrequent and short; a pool of VMs would be premature.
type Engine struct {
	logger   *slog.Logger
	vm       *lua.LState
	mu       sync.Mutex
	userDir  string
	loaded   map[string]bool
}

// New creates a scripting engine. The logger should be the daemon's
// structured logger; script load/error events are emitted at info/warn.
// The user override directory is derived from XDG_CONFIG_HOME.
func New(logger *slog.Logger) *Engine {
	if logger == nil {
		logger = slog.Default()
	}
	vm := lua.NewState(lua.Options{
		SkipOpenLibs: false,
	})
	userDir := userScriptDir()
	return &Engine{
		logger:  logger,
		vm:      vm,
		userDir: userDir,
		loaded:  make(map[string]bool),
	}
}

// Close shuts down the Lua VM. Safe to call multiple times.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vm != nil {
		e.vm.Close()
		e.vm = nil
	}
}

// LoadScript loads and executes a script by its relative path (e.g.
// "appstore/categories.lua"). If a user override exists it takes
// precedence over the embedded default. The script's return values
// are left on the Lua stack — callers use the Call* helpers to
// retrieve them. Repeated loads of the same path are no-ops;
// the script runs once and its globals persist in the VM.
func (e *Engine) LoadScript(path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.loaded[path] {
		return nil
	}
	src, source, err := e.readScript(path)
	if err != nil {
		return fmt.Errorf("scripting: load %s: %w", path, err)
	}
	fn, err := e.vm.LoadString(string(src))
	if err != nil {
		return fmt.Errorf("scripting: compile %s: %w", source, err)
	}
	e.vm.Push(fn)
	if err := e.vm.PCall(0, lua.MultRet, nil); err != nil {
		return fmt.Errorf("scripting: exec %s: %w", source, err)
	}
	e.loaded[path] = true
	e.logger.Info("scripting: loaded", "script", source)
	return nil
}

// GetGlobal returns a Lua global by name. The caller must hold no
// expectation about the type — use the lua.LV* type switch or the
// typed helpers below.
func (e *Engine) GetGlobal(name string) lua.LValue {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.vm.GetGlobal(name)
}

// CallFunction calls a global Lua function by name and returns its
// first return value. Arguments are converted to Lua values. This is
// the primary hook point for Tier 2 — services call named functions
// that scripts have registered.
func (e *Engine) CallFunction(name string, args ...lua.LValue) (lua.LValue, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	fn := e.vm.GetGlobal(name)
	if fn == lua.LNil {
		return lua.LNil, fmt.Errorf("scripting: function %q not defined", name)
	}
	if err := e.vm.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, args...); err != nil {
		return lua.LNil, fmt.Errorf("scripting: call %s: %w", name, err)
	}
	ret := e.vm.Get(-1)
	e.vm.Pop(1)
	return ret, nil
}

// TableToStringMap converts a Lua table with string keys and string
// values into a Go map. Non-string keys or values are silently
// skipped. This is the workhorse for reading config tables like the
// XDG category map and the per-package overrides.
func TableToStringMap(t *lua.LTable) map[string]string {
	m := make(map[string]string, t.Len())
	t.ForEach(func(k, v lua.LValue) {
		ks, ok1 := k.(lua.LString)
		vs, ok2 := v.(lua.LString)
		if ok1 && ok2 {
			m[string(ks)] = string(vs)
		}
	})
	return m
}

// TableToStringSlice converts a Lua table used as an array (integer
// keys) into a Go string slice. Non-string values are skipped.
func TableToStringSlice(t *lua.LTable) []string {
	out := make([]string, 0, t.Len())
	t.ForEach(func(k, v lua.LValue) {
		if _, ok := k.(lua.LNumber); ok {
			if s, ok := v.(lua.LString); ok {
				out = append(out, string(s))
			}
		}
	})
	return out
}

// readScript returns the script source and a human-readable origin
// label. User overrides take precedence.
func (e *Engine) readScript(path string) ([]byte, string, error) {
	if e.userDir != "" {
		userPath := filepath.Join(e.userDir, path)
		if b, err := os.ReadFile(userPath); err == nil {
			return b, userPath, nil
		}
	}
	embeddedPath := "scripts/" + path
	b, err := defaultScripts.ReadFile(embeddedPath)
	if err != nil {
		return nil, "", fmt.Errorf("no script at %s (checked user dir %s and embedded)", path, e.userDir)
	}
	return b, "embedded:" + embeddedPath, nil
}

func userScriptDir() string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "dark", "scripts")
}

// ScriptExists reports whether a script exists at the given relative
// path, checking user overrides first, then embedded defaults. Useful
// for optional scripts where the caller wants to skip gracefully.
func (e *Engine) ScriptExists(path string) bool {
	if e.userDir != "" {
		userPath := filepath.Join(e.userDir, path)
		if _, err := os.Stat(userPath); err == nil {
			return true
		}
	}
	embeddedPath := "scripts/" + path
	if _, err := defaultScripts.ReadFile(embeddedPath); err == nil {
		return true
	}
	return false
}

// UserDir returns the user script override directory for logging and
// help documentation. Empty when XDG_CONFIG_HOME couldn't be derived.
func (e *Engine) UserDir() string {
	return e.userDir
}

// SetGlobal exposes a Go value to Lua scripts as a global variable.
// Used by services to inject context before calling script functions
// (e.g. passing the package list to a categorization hook).
func (e *Engine) SetGlobal(name string, value lua.LValue) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.vm.SetGlobal(name, value)
}

// VM returns the raw GopherLua state for advanced use cases. Callers
// MUST hold their own lock via Lock/Unlock if they access the VM
// directly. Prefer the typed helpers above for normal use.
func (e *Engine) VM() *lua.LState {
	return e.vm
}

// Lock acquires the engine mutex. Pair with Unlock. Only needed when
// callers use VM() directly.
func (e *Engine) Lock()   { e.mu.Lock() }
func (e *Engine) Unlock() { e.mu.Unlock() }

// GoString converts a Lua value to a Go string. Returns the fallback
// when the value is nil or not a string. Convenience for pulling
// single values out of tables.
func GoString(v lua.LValue, fallback string) string {
	if s, ok := v.(lua.LString); ok {
		return strings.TrimSpace(string(s))
	}
	return fallback
}
