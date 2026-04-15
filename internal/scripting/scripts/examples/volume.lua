-- volume.lua — example script showing how to read audio state and
-- change the default sink's volume through the scripting API.
--
-- darkd loads every .lua file in ~/.config/dark/scripts/ at startup
-- and whenever you save one from the F5 Scripting tab. This file
-- ships as an example; edit it freely or delete it if you don't
-- want the helpers in your namespace.
--
-- After this script loads, three globals are available to any other
-- Lua code running in the same engine:
--
--   volume_up(step?)    — nudge the default sink up by `step`% (5 by default)
--   volume_down(step?)  — nudge the default sink down by `step`% (5 by default)
--   volume_set(pct)     — set the default sink to an absolute 0–150 %
--
-- To trigger a helper from outside the engine, use the dark CLI:
--
--   dark script call volume_up
--   dark script call volume_down
--   dark script call volume_set 75
--
-- Arguments are parsed as JSON literals, so numbers and booleans
-- come through without quoting. Wire these commands to Hyprland
-- keybindings for media-key style control:
--
--   bind = , XF86AudioRaiseVolume, exec, dark script call volume_up
--   bind = , XF86AudioLowerVolume, exec, dark script call volume_down
--   bind = , XF86AudioMute,        exec, dark script call volume_set 0
--
-- The script also installs an `on_audio` hook so it always knows
-- which PipeWire sink is the current default — darkd publishes an
-- audio snapshot whenever volume, mute, default device, or the app
-- stream list changes, and this hook caches the default sink from
-- each update.

local default_sink = nil

dark.on("on_audio", function(snap)
  local sinks = snap.sinks or {}
  for _, sink in ipairs(sinks) do
    if sink.default then
      default_sink = sink
      return
    end
  end
  default_sink = sinks[1]
end)

local function clamp(v, lo, hi)
  if v < lo then return lo end
  if v > hi then return hi end
  return v
end

-- Set the default sink volume to an absolute percentage (0–150).
-- 100 = 0 dB; the kernel mixer allows soft-boost up to 150.
function volume_set(pct)
  if not default_sink then
    dark.log("volume_set: no default sink yet — waiting for audio snapshot")
    return
  end
  local target = clamp(math.floor(pct + 0.5), 0, 150)
  local _, err = dark.actions.audio.sink_volume({
    index = default_sink.index,
    volume = target,
  })
  if err then
    dark.log("volume_set failed: " .. err)
  else
    dark.log("volume_set: default sink → " .. target .. "%")
  end
end

-- Relative bump helpers. Both default to a 5% step so tapping them
-- repeatedly produces an audible change without blasting the sink.
function volume_up(step)
  if not default_sink then
    dark.log("volume_up: no default sink yet")
    return
  end
  volume_set((default_sink.volume or 0) + (step or 5))
end

function volume_down(step)
  if not default_sink then
    dark.log("volume_down: no default sink yet")
    return
  end
  volume_set((default_sink.volume or 0) - (step or 5))
end

dark.log("volume.lua loaded — volume_up() / volume_down() / volume_set() helpers are now global")
