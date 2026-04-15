-- wifi_watch.lua — pure event observer, no CLI surface.
--
-- This example is the opposite end of the spectrum from volume.lua:
-- it doesn't expose any callable functions and doesn't need the
-- `dark script call` CLI at all. It just listens for wifi snapshots
-- and logs whenever the connected SSID changes, which is useful for
-- surfacing roaming events in the darkd journal.
--
-- Tail it with:
--
--   journalctl --user -u darkd -f | grep wifi_watch
--
-- Extending this is straightforward — inside the hook you can call
-- any `dark.actions.*` function, which means you could auto-set
-- display brightness based on SSID, remount a VPN on connect, etc.

local last_ssid = nil

dark.on("on_wifi", function(snap)
  local ssid = nil
  for _, adapter in ipairs(snap.adapters or {}) do
    for _, net in ipairs(adapter.networks or {}) do
      if net.connected then
        ssid = net.ssid
        break
      end
    end
    if ssid then break end
  end

  if ssid ~= last_ssid then
    if ssid then
      dark.log("wifi_watch: connected to " .. ssid)
    elseif last_ssid then
      dark.log("wifi_watch: disconnected from " .. last_ssid)
    end
    last_ssid = ssid
  end
end)

dark.log("wifi_watch.lua loaded — watching for SSID changes")
