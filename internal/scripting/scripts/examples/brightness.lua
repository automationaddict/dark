-- brightness.lua — nudge the backlight with relative steps.
--
-- Companion to volume.lua. Works the same way: define global helpers
-- and expose them to the `dark script call` CLI so Hyprland bindings
-- can drive them.
--
-- Globals exposed:
--
--   brightness_up(step?)    — increase backlight by `step`% (5 default)
--   brightness_down(step?)  — decrease backlight by `step`% (5 default)
--   brightness_set(pct)     — set backlight to an absolute 0–100 %
--
-- Hyprland bindings to try:
--
--   bind = , XF86MonBrightnessUp,   exec, dark script call brightness_up
--   bind = , XF86MonBrightnessDown, exec, dark script call brightness_down
--
-- Unlike volume the display brightness subject takes a flat `pct`
-- field (0–100) and applies to whichever backlight dark controls on
-- the primary output, so there's no per-monitor bookkeeping.

local current_pct = 50 -- conservative default until the first snapshot arrives

dark.on("on_display", function(snap)
  local monitors = snap.monitors or {}
  for _, m in ipairs(monitors) do
    if m.backlight_pct then
      current_pct = m.backlight_pct
      return
    end
  end
end)

local function clamp(v, lo, hi)
  if v < lo then return lo end
  if v > hi then return hi end
  return v
end

function brightness_set(pct)
  local target = clamp(math.floor(pct + 0.5), 0, 100)
  local _, err = dark.actions.display.brightness({ pct = target })
  if err then
    dark.log("brightness_set failed: " .. err)
  else
    current_pct = target
    dark.log("brightness_set: " .. target .. "%")
  end
end

function brightness_up(step)
  brightness_set((current_pct or 50) + (step or 5))
end

function brightness_down(step)
  brightness_set((current_pct or 50) - (step or 5))
end

dark.log("brightness.lua loaded — brightness_up() / brightness_down() / brightness_set() are now global")
