package scripting

import (
	"encoding/json"

	lua "github.com/yuin/gopher-lua"
)

// hooksTable is the Lua global that holds registered event callbacks.
// Layout: _dark_hooks[event_name] = { fn1, fn2, ... }. Scripts never
// touch this directly — they go through dark.on(event, fn).
const hooksTable = "_dark_hooks"

// DispatchEvent invokes every Lua function registered for the named
// event, converting Go arguments to Lua values in order. Errors from
// individual hooks are logged and do not abort the remaining hooks —
// a misbehaving user script must not be able to break a service.
//
// ClearUserHooks wipes every registered event hook so a reload
// doesn't stack duplicate handlers on top of previous runs. Called
// before re-running the user scripts directory after a save/delete.
func (e *Engine) ClearUserHooks() {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vm == nil {
		return
	}
	e.vm.SetGlobal(hooksTable, e.vm.NewTable())
}

// DispatchEvent is safe to call from any goroutine; it serializes on
// the same mutex that guards the VM.
func (e *Engine) DispatchEvent(name string, args ...interface{}) {
	if e == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vm == nil {
		return
	}

	hooks := e.vm.GetGlobal(hooksTable)
	t, ok := hooks.(*lua.LTable)
	if !ok {
		return
	}
	bucket, ok := t.RawGetString(name).(*lua.LTable)
	if !ok {
		return
	}

	luaArgs := make([]lua.LValue, 0, len(args))
	for _, a := range args {
		luaArgs = append(luaArgs, goValueToLua(e.vm, a))
	}

	bucket.ForEach(func(_, fn lua.LValue) {
		if fn.Type() != lua.LTFunction {
			return
		}
		if err := e.vm.CallByParam(lua.P{
			Fn:      fn,
			NRet:    0,
			Protect: true,
		}, luaArgs...); err != nil {
			e.logger.Warn("scripting: hook error",
				"event", name, "error", err.Error())
		}
	})
}

// goValueToLua is the converter used by DispatchEvent. Basic scalar
// types are handled directly; anything else (service snapshot
// structs, nested maps) falls through to a JSON round-trip so the
// receiving Lua hook sees a plain table without the scripting
// package needing to know about every service type.
func goValueToLua(L *lua.LState, v interface{}) lua.LValue {
	switch val := v.(type) {
	case nil:
		return lua.LNil
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
	case []string:
		t := L.NewTable()
		for _, s := range val {
			t.Append(lua.LString(s))
		}
		return t
	case map[string]interface{}:
		return goToLua(L, val)
	case []interface{}:
		return goToLua(L, val)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return lua.LNil
	}
	var raw interface{}
	if err := json.Unmarshal(b, &raw); err != nil {
		return lua.LString(string(b))
	}
	return goToLua(L, raw)
}
