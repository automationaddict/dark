package scripting

import (
	"encoding/json"
	"fmt"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// registerDarkModule installs the `dark` global Lua table and
// populates it with the host functions user scripts call: on() to
// register an event hook and log() to emit a line into the daemon
// log. The _dark_hooks shadow table is created here so DispatchEvent
// can assume it exists.
//
// Call from Engine.New while the mutex is owned by the constructor
// path. All functions invoked here access the VM directly and must
// not re-enter the engine mutex.
func (e *Engine) registerDarkModule() {
	L := e.vm

	// Shadow table for hook storage. Scripts never touch this.
	L.SetGlobal(hooksTable, L.NewTable())

	darkTbl := L.NewTable()
	L.SetFuncs(darkTbl, map[string]lua.LGFunction{
		"on":  e.luaOn,
		"log": e.luaLog,
		"cmd": e.luaCmd,
	})
	L.SetGlobal("dark", darkTbl)

	e.registry.RegisterFunction("dark.on",
		"(event, fn)",
		"Register a Lua function to run when the named event fires.")
	e.registry.RegisterFunction("dark.log",
		"(message)",
		"Write a line to the dark daemon log at info level.")
	e.registry.RegisterFunction("dark.cmd",
		"(subject, payload)",
		"Issue a NATS request against a dark.cmd.* subject. Returns (reply_table, nil) on success or (nil, error_string) on failure.")
}

// luaOn implements dark.on(event, fn). Appends fn to the event's
// bucket inside _dark_hooks, creating the bucket on first use.
// Errors surface through L.ArgError so misuse fails loudly in Lua.
func (e *Engine) luaOn(L *lua.LState) int {
	event := L.CheckString(1)
	fn := L.CheckFunction(2)

	hooks, ok := L.GetGlobal(hooksTable).(*lua.LTable)
	if !ok {
		L.RaiseError("scripting: hook table missing")
		return 0
	}
	bucket, ok := hooks.RawGetString(event).(*lua.LTable)
	if !ok {
		bucket = L.NewTable()
		hooks.RawSetString(event, bucket)
	}
	bucket.Append(fn)
	return 0
}

// luaLog implements dark.log(message). Emits the message through
// the engine logger at info level tagged with "script".
func (e *Engine) luaLog(L *lua.LState) int {
	msg := L.CheckString(1)
	e.logger.Info("script: "+msg, "source", "lua")
	return 0
}

// luaCmd implements dark.cmd(subject, payload). Payload is optional;
// when present it must be a table that JSON-marshals cleanly. The
// request is dispatched through the engine's installed RequesterFunc
// (set by darkd at startup). The reply is JSON-decoded into a Lua
// table and returned; on any failure the function returns (nil, err)
// so Lua callers can destructure with `local reply, err = ...`.
func (e *Engine) luaCmd(L *lua.LState) int {
	subject := L.CheckString(1)
	return e.dispatchRequest(L, subject, 2, "dark.cmd")
}

// dispatchRequest is the shared worker behind both dark.cmd and every
// auto-generated dark.actions.<domain>.<verb> wrapper. payloadArg is
// the 1-based Lua stack index of the optional payload table; label is
// the caller name used in error messages so the user can tell whether
// they hit dark.cmd or a specific action wrapper.
func (e *Engine) dispatchRequest(L *lua.LState, subject string, payloadArg int, label string) int {
	if e.requester == nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(label + ": no bus requester installed"))
		return 2
	}

	var payload []byte
	if L.GetTop() >= payloadArg && L.Get(payloadArg) != lua.LNil {
		tbl, ok := L.Get(payloadArg).(*lua.LTable)
		if !ok {
			L.Push(lua.LNil)
			L.Push(lua.LString(label + ": payload must be a table"))
			return 2
		}
		raw := luaTableToGo(tbl)
		b, err := json.Marshal(raw)
		if err != nil {
			L.Push(lua.LNil)
			L.Push(lua.LString(fmt.Sprintf("%s: marshal payload: %s", label, err)))
			return 2
		}
		payload = b
	}

	reply, err := e.requester(subject, payload)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}
	if len(reply) == 0 {
		L.Push(L.NewTable())
		L.Push(lua.LNil)
		return 2
	}
	var decoded interface{}
	if err := json.Unmarshal(reply, &decoded); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("%s: decode reply: %s", label, err)))
		return 2
	}
	L.Push(goToLua(L, decoded))
	L.Push(lua.LNil)
	return 2
}

// RegisterAction installs a first-class Lua wrapper around a bus
// command subject. The wrapper appears at dark.actions.<domain>.<verb>
// and is simultaneously registered in the Lua symbol registry so it
// shows up in the F5 reference browser. The Lua side is nothing more
// than a typed alias for dark.cmd(subject, payload) — it exists so
// scripts can discover commands by tab-completing into dark.actions
// instead of having to remember every subject constant.
//
// Subjects that don't match the `dark.cmd.<domain>.<verb>` shape are
// silently skipped so new bus naming patterns don't blow up the
// daemon at startup. Fields carries the payload schema so the F5
// Scripting doc can render a real parameter table.
func (e *Engine) RegisterAction(subject, summary string, fields []RegistryField) {
	parts := strings.Split(subject, ".")
	if len(parts) != 4 || parts[0] != "dark" || parts[1] != "cmd" {
		return
	}
	domain, verb := parts[2], parts[3]
	name := "dark.actions." + domain + "." + verb
	e.registry.RegisterAction(name, subject, summary, fields)

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vm == nil {
		return
	}
	L := e.vm
	dark, ok := L.GetGlobal("dark").(*lua.LTable)
	if !ok {
		return
	}
	actions, ok := dark.RawGetString("actions").(*lua.LTable)
	if !ok {
		actions = L.NewTable()
		dark.RawSetString("actions", actions)
	}
	domainTbl, ok := actions.RawGetString(domain).(*lua.LTable)
	if !ok {
		domainTbl = L.NewTable()
		actions.RawSetString(domain, domainTbl)
	}
	captured := subject
	domainTbl.RawSetString(verb, L.NewFunction(func(ls *lua.LState) int {
		return e.dispatchRequest(ls, captured, 1, name)
	}))
}

// CallUserFunction invokes a Lua global by name with JSON-friendly
// arguments and returns the result decoded back into Go. Used by the
// dark.cmd.scripting.call bus handler so external callers (a CLI
// subcommand, a Hyprland keybinding, another service) can trigger
// helper functions scripts have defined at top level. Returns an
// error when the name isn't a function or when the invocation panics
// on the Lua side.
func (e *Engine) CallUserFunction(name string, args ...interface{}) (interface{}, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.vm == nil {
		return nil, fmt.Errorf("scripting: engine closed")
	}
	L := e.vm
	fn := L.GetGlobal(name)
	if fn == lua.LNil {
		return nil, fmt.Errorf("scripting: function %q not defined", name)
	}
	if fn.Type() != lua.LTFunction {
		return nil, fmt.Errorf("scripting: %q is not a function (got %s)", name, fn.Type())
	}
	luaArgs := make([]lua.LValue, 0, len(args))
	for _, a := range args {
		luaArgs = append(luaArgs, goValueToLua(L, a))
	}
	if err := L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, luaArgs...); err != nil {
		return nil, fmt.Errorf("scripting: call %s: %w", name, err)
	}
	ret := L.Get(-1)
	L.Pop(1)
	return luaValueToGo(ret), nil
}

// luaTableToGo converts a Lua table to its nearest Go equivalent —
// an []interface{} when every key is an integer 1..n (a Lua array),
// otherwise a map[string]interface{}. Nested tables recurse.
func luaTableToGo(t *lua.LTable) interface{} {
	if t == nil {
		return nil
	}
	// Detect array vs map: if every key is a positive integer from
	// 1..n with no gaps, treat as array.
	n := t.Len()
	isArray := n > 0
	count := 0
	t.ForEach(func(k, _ lua.LValue) {
		count++
		if _, ok := k.(lua.LNumber); !ok {
			isArray = false
		}
	})
	if isArray && count == n {
		out := make([]interface{}, 0, n)
		for i := 1; i <= n; i++ {
			out = append(out, luaValueToGo(t.RawGetInt(i)))
		}
		return out
	}
	m := make(map[string]interface{})
	t.ForEach(func(k, v lua.LValue) {
		m[k.String()] = luaValueToGo(v)
	})
	return m
}

func luaValueToGo(v lua.LValue) interface{} {
	if v == nil || v == lua.LNil {
		return nil
	}
	switch val := v.(type) {
	case lua.LBool:
		return bool(val)
	case lua.LNumber:
		return float64(val)
	case lua.LString:
		return string(val)
	case *lua.LTable:
		return luaTableToGo(val)
	}
	return nil
}
