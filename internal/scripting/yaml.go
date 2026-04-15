package scripting

import (
	"fmt"
	"os"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
	"gopkg.in/yaml.v3"
)

// registerYAML adds the load_yaml(path) function to the Lua VM. The
// function reads a YAML file from the same embedded + user-override
// resolution as scripts, parses it, and returns the top-level value
// as a Lua table/string/number/bool. This lets Lua scripts pull in
// structured data without embedding it in the script itself.
func (e *Engine) registerYAML() {
	e.vm.SetGlobal("load_yaml", e.vm.NewFunction(func(L *lua.LState) int {
		path := L.CheckString(1)
		data, _, err := e.readDataFile(path)
		if err != nil {
			L.ArgError(1, err.Error())
			return 0
		}
		var raw interface{}
		if err := yaml.Unmarshal(data, &raw); err != nil {
			L.ArgError(1, fmt.Sprintf("parse %s: %s", path, err))
			return 0
		}
		L.Push(goToLua(L, raw))
		return 1
	}))
	e.registry.RegisterFunction("load_yaml",
		"(path)",
		"Load a YAML file from the user scripts directory (with embedded fallback) and return it as a Lua table.")
}

// readDataFile reads a data file (YAML, JSON, etc.) from the user
// override directory first, falling back to the embedded default.
// Data files use the same "scripts/" prefix in the embed FS as Lua
// scripts so they can live alongside each other.
func (e *Engine) readDataFile(path string) ([]byte, string, error) {
	if e.userDir != "" {
		userPath := filepath.Join(e.userDir, path)
		if b, err := os.ReadFile(userPath); err == nil {
			return b, userPath, nil
		}
	}
	embeddedPath := "scripts/" + path
	b, err := defaultScripts.ReadFile(embeddedPath)
	if err != nil {
		return nil, "", fmt.Errorf("no data file at %s (checked user dir and embedded)", path)
	}
	return b, "embedded:" + embeddedPath, nil
}

// goToLua recursively converts a Go value (as produced by yaml.Unmarshal
// into interface{}) to the corresponding Lua value. Maps become tables
// with string keys, slices become tables with integer keys, and scalar
// types map to their Lua equivalents.
func goToLua(L *lua.LState, v interface{}) lua.LValue {
	switch val := v.(type) {
	case map[string]interface{}:
		t := L.NewTable()
		for k, v := range val {
			t.RawSetString(k, goToLua(L, v))
		}
		return t
	case map[interface{}]interface{}:
		t := L.NewTable()
		for k, v := range val {
			t.RawSetString(fmt.Sprint(k), goToLua(L, v))
		}
		return t
	case []interface{}:
		t := L.NewTable()
		for _, v := range val {
			t.Append(goToLua(L, v))
		}
		return t
	case string:
		return lua.LString(val)
	case bool:
		return lua.LBool(val)
	case int:
		return lua.LNumber(float64(val))
	case int64:
		return lua.LNumber(float64(val))
	case float64:
		return lua.LNumber(val)
	case nil:
		return lua.LNil
	default:
		return lua.LString(fmt.Sprint(val))
	}
}
