-- dnd_toggle.lua — flip do-not-disturb and report the new state.
--
-- Demonstrates the stateful toggle pattern: cache the current value
-- from a snapshot event, flip it on demand, then call the action
-- with the inverted state. Different from volume/brightness, which
-- are purely relative bumps.
--
-- Globals exposed:
--
--   dnd_toggle()   — flip DND on/off, returns the new state (bool)
--   dnd_set(on)    — force DND to the given state
--
-- Hyprland binding idea:
--
--   bind = SUPER, N, exec, dark script call dnd_toggle

local dnd_enabled = false

dark.on("on_notify", function(snap)
  if snap.dnd ~= nil then
    dnd_enabled = snap.dnd
  end
end)

function dnd_set(on)
  local _, err = dark.actions.notify.dnd({ enabled = on })
  if err then
    dark.log("dnd_set failed: " .. err)
    return nil
  end
  dnd_enabled = on
  dark.log("dnd_set: " .. tostring(on))
  return on
end

function dnd_toggle()
  return dnd_set(not dnd_enabled)
end

dark.log("dnd_toggle.lua loaded — dnd_toggle() / dnd_set() are now global")
