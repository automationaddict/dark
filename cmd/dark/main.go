package main

import (
	"encoding/json"
	"fmt"
	"os"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"

	"github.com/johnnelson/dark/internal/bus"
	"github.com/johnnelson/dark/internal/core"
	"github.com/johnnelson/dark/internal/help"
	"github.com/johnnelson/dark/internal/lock"
	appstoresvc "github.com/johnnelson/dark/internal/services/appstore"
	audiosvc "github.com/johnnelson/dark/internal/services/audio"
	btsvc "github.com/johnnelson/dark/internal/services/bluetooth"
	displaysvc "github.com/johnnelson/dark/internal/services/display"
	netsvc "github.com/johnnelson/dark/internal/services/network"
	"github.com/johnnelson/dark/internal/services/notify"
	"github.com/johnnelson/dark/internal/services/sysinfo"
	"github.com/johnnelson/dark/internal/services/wifi"
	"github.com/johnnelson/dark/internal/theme"
	"github.com/johnnelson/dark/internal/tui"
)

func main() {
	lk, err := lock.Acquire("dark")
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark:", err)
		os.Exit(1)
	}
	defer lk.Release()

	binPath, err := os.Executable()
	if err != nil {
		binPath = os.Args[0]
	}

	palette := theme.Load()
	tui.ApplyPalette(palette)
	help.SetPalette(palette)

	startTab := core.ParseStartTab(os.Args[1:])

	nc, err := bus.ConnectClient("dark", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark:", err)
		os.Exit(1)
	}
	defer nc.Drain()

	state := core.NewState(startTab, binPath)

	wifiActions := tui.WifiActions{
		Scan: func(adapter string) tea.Cmd {
			return func() tea.Msg {
				snap, err := wifiRequest(nc, bus.SubjectWifiScanCmd, adapter, "")
				if err != nil {
					return tui.WifiScanResultMsg{Err: err.Error()}
				}
				return tui.WifiScanResultMsg{Snapshot: snap}
			}
		},
		Connect: func(adapter, ssid, passphrase string) tea.Cmd {
			return func() tea.Msg {
				return wifiConnectRequest(nc, bus.SubjectWifiConnectCmd, adapter, ssid, passphrase)
			}
		},
		ConnectHidden: func(adapter, ssid, passphrase string) tea.Cmd {
			return func() tea.Msg {
				return wifiConnectRequest(nc, bus.SubjectWifiConnectHiddenCmd, adapter, ssid, passphrase)
			}
		},
		Disconnect: func(adapter string) tea.Cmd {
			return func() tea.Msg {
				snap, err := wifiRequest(nc, bus.SubjectWifiDisconnectCmd, adapter, "")
				if err != nil {
					return tui.WifiActionResultMsg{Err: err.Error()}
				}
				return tui.WifiActionResultMsg{Snapshot: snap}
			}
		},
		Forget: func(adapter, ssid string) tea.Cmd {
			return func() tea.Msg {
				snap, err := wifiRequest(nc, bus.SubjectWifiForgetCmd, adapter, ssid)
				if err != nil {
					return tui.WifiActionResultMsg{Err: err.Error()}
				}
				return tui.WifiActionResultMsg{Snapshot: snap}
			}
		},
		SetPower: func(adapter string, powered bool) tea.Cmd {
			return func() tea.Msg {
				return wifiBoolRequest(nc, bus.SubjectWifiPowerCmd, adapter, "", powered)
			}
		},
		SetAutoConnect: func(ssid string, enabled bool) tea.Cmd {
			return func() tea.Msg {
				return wifiBoolRequest(nc, bus.SubjectWifiAutoconnectCmd, "", ssid, enabled)
			}
		},
		StartAP: func(adapter, ssid, passphrase string) tea.Cmd {
			return func() tea.Msg {
				return wifiConnectRequest(nc, bus.SubjectWifiAPStartCmd, adapter, ssid, passphrase)
			}
		},
		StopAP: func(adapter string) tea.Cmd {
			return func() tea.Msg {
				snap, err := wifiRequest(nc, bus.SubjectWifiAPStopCmd, adapter, "")
				if err != nil {
					return tui.WifiActionResultMsg{Err: err.Error()}
				}
				return tui.WifiActionResultMsg{Snapshot: snap}
			}
		},
	}

	bluetoothActions := newBluetoothActions(nc)
	audioActions := newAudioActions(nc)
	displayActions := newDisplayActions(nc)
	networkActions := newNetworkActions(nc)
	appstoreActions := newAppstoreActions(nc)

	// Best-effort: if we can't reach the session bus, the notifier
	// stays nil and the model's notifyError helper becomes a no-op.
	// dark still runs perfectly fine without desktop notifications.
	notifier, err := notify.New("dark")
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: notifications disabled:", err)
		notifier = nil
	}
	defer notifier.Close()

	model := tui.New(state, binPath, wifiActions, bluetoothActions, audioActions, networkActions, displayActions, notifier, appstoreActions)

	p := tea.NewProgram(model, tea.WithAltScreen())

	// Wire NATS connection lifecycle handlers AFTER the program exists so
	// they can use p.Send to push status changes into the bubble tea loop.
	nc.SetDisconnectErrHandler(func(_ *nats.Conn, _ error) {
		p.Send(tui.BusStatusMsg(false))
	})
	nc.SetReconnectHandler(func(_ *nats.Conn) {
		p.Send(tui.BusStatusMsg(true))
	})
	nc.SetClosedHandler(func(_ *nats.Conn) {
		p.Send(tui.BusStatusMsg(false))
	})

	// Subscribe to system snapshots and forward them into the bubble tea
	// program. Bubble tea is goroutine-safe through Program.Send so this
	// is the standard pattern for piping external events into the model.
	// warnDecode fires a debounced notification when a subscription
	// callback can't unmarshal a message from darkd — typically means
	// the daemon and client are running different builds.
	warnDecode := func(section string, err error) {
		if notifier != nil {
			notifier.Send(notify.Message{
				Summary: "dark · " + section,
				Body:    "failed to decode update from daemon — try ctrl+r to rebuild",
				Urgency: notify.UrgencyNormal,
				Icon:    "dialog-warning",
			})
		}
	}

	sub, err := nc.Subscribe(bus.SubjectSystemInfo, func(m *nats.Msg) {
		var info sysinfo.SystemInfo
		if err := json.Unmarshal(m.Data, &info); err != nil {
			warnDecode("System", err)
			return
		}
		p.Send(tui.SysInfoMsg(info))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: subscribe:", err)
		os.Exit(1)
	}
	defer sub.Unsubscribe()

	wifiSub, err := nc.Subscribe(bus.SubjectWifiAdapters, func(m *nats.Msg) {
		var snap wifi.Snapshot
		if err := json.Unmarshal(m.Data, &snap); err != nil {
			warnDecode("Wi-Fi", err)
			return
		}
		p.Send(tui.WifiMsg(snap))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: subscribe wifi:", err)
		os.Exit(1)
	}
	defer wifiSub.Unsubscribe()

	btSub, err := nc.Subscribe(bus.SubjectBluetoothAdapters, func(m *nats.Msg) {
		var snap btsvc.Snapshot
		if err := json.Unmarshal(m.Data, &snap); err != nil {
			warnDecode("Bluetooth", err)
			return
		}
		p.Send(tui.BluetoothMsg(snap))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: subscribe bluetooth:", err)
		os.Exit(1)
	}
	defer btSub.Unsubscribe()

	audioSub, err := nc.Subscribe(bus.SubjectAudioDevices, func(m *nats.Msg) {
		var snap audiosvc.Snapshot
		if err := json.Unmarshal(m.Data, &snap); err != nil {
			warnDecode("Sound", err)
			return
		}
		p.Send(tui.AudioMsg(snap))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: subscribe audio:", err)
		os.Exit(1)
	}
	defer audioSub.Unsubscribe()

	audioLevelsSub, err := nc.Subscribe(bus.SubjectAudioLevels, func(m *nats.Msg) {
		var levels audiosvc.Levels
		if err := json.Unmarshal(m.Data, &levels); err != nil {
			// Levels arrive at 20 Hz — never notify for decode failures
			// on this high-frequency channel; just drop silently.
			return
		}
		p.Send(tui.AudioLevelsMsg(levels))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: subscribe audio levels:", err)
		os.Exit(1)
	}
	defer audioLevelsSub.Unsubscribe()

	displaySub, err := nc.Subscribe(bus.SubjectDisplayMonitors, func(m *nats.Msg) {
		var snap displaysvc.Snapshot
		if err := json.Unmarshal(m.Data, &snap); err != nil {
			warnDecode("Displays", err)
			return
		}
		p.Send(tui.DisplayMsg(snap))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: subscribe display:", err)
		os.Exit(1)
	}
	defer displaySub.Unsubscribe()

	networkSub, err := nc.Subscribe(bus.SubjectNetworkSnapshot, func(m *nats.Msg) {
		var snap netsvc.Snapshot
		if err := json.Unmarshal(m.Data, &snap); err != nil {
			warnDecode("Network", err)
			return
		}
		p.Send(tui.NetworkMsg(snap))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: subscribe network:", err)
		os.Exit(1)
	}
	defer networkSub.Unsubscribe()

	appstoreSub, err := nc.Subscribe(bus.SubjectAppstoreCatalog, func(m *nats.Msg) {
		var snap appstoresvc.Snapshot
		if err := json.Unmarshal(m.Data, &snap); err != nil {
			warnDecode("App Store", err)
			return
		}
		p.Send(tui.AppstoreMsg(snap))
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dark: subscribe appstore:", err)
		os.Exit(1)
	}
	defer appstoreSub.Unsubscribe()

	// Request current snapshots up front so each tab has data on the
	// first frame instead of waiting for the next periodic publish.
	if reply, err := nc.Request(bus.SubjectSystemInfoCmd, nil, core.TimeoutFast); err == nil {
		var info sysinfo.SystemInfo
		if err := json.Unmarshal(reply.Data, &info); err == nil {
			state.SetSysInfo(info)
		}
	}
	if reply, err := nc.Request(bus.SubjectWifiAdaptersCmd, nil, core.TimeoutFast); err == nil {
		var snap wifi.Snapshot
		if err := json.Unmarshal(reply.Data, &snap); err == nil {
			state.SetWifi(snap)
		}
	}
	if snap, ok := requestInitialBluetooth(nc); ok {
		state.SetBluetooth(snap)
	}
	if snap, ok := requestInitialAudio(nc); ok {
		state.SetAudio(snap)
	}
	if snap, ok := requestInitialDisplay(nc); ok {
		state.SetDisplay(snap)
	}
	if snap, ok := requestInitialNetwork(nc); ok {
		state.SetNetwork(snap)
	}
	if snap, ok := requestInitialAppstore(nc); ok {
		state.SetAppstore(snap)
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "dark:", err)
		os.Exit(1)
	}

	// Shutdown timeout: if the deferred cleanup (nc.Drain, service
	// Close, notifier Close, etc.) hangs on a stuck connection, force
	// exit after 5 seconds. The restart path skips this because
	// syscall.Exec replaces the process immediately.
	if !state.RestartRequested {
		go func() {
			time.Sleep(core.ShutdownTimeout)
			fmt.Fprintln(os.Stderr, "dark: shutdown timeout — force exit")
			os.Exit(1)
		}()
	}

	if state.RestartRequested {
		if err := syscall.Exec(binPath, os.Args, os.Environ()); err != nil {
			fmt.Fprintln(os.Stderr, "dark: restart failed:", err)
			os.Exit(1)
		}
	}
}

// wifiConnectRequest is a wifiRequest variant that also carries a
// passphrase. Used for credentialed connect and hidden-network connect.
func wifiConnectRequest(nc *nats.Conn, subject, adapter, ssid, passphrase string) tui.WifiActionResultMsg {
	payload, _ := json.Marshal(map[string]string{
		"adapter":    adapter,
		"ssid":       ssid,
		"passphrase": passphrase,
	})
	reply, err := nc.Request(subject, payload, core.TimeoutLong)
	if err != nil {
		return tui.WifiActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot wifi.Snapshot `json:"snapshot"`
		Error    string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.WifiActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.WifiActionResultMsg{Err: resp.Error}
	}
	return tui.WifiActionResultMsg{Snapshot: resp.Snapshot}
}

// wifiBoolRequest is a wifiRequest variant that also carries a bool flag
// in the payload. Used for power toggle and autoconnect toggle.
func wifiBoolRequest(nc *nats.Conn, subject, adapter, ssid string, flag bool) tui.WifiActionResultMsg {
	payload, _ := json.Marshal(map[string]any{
		"adapter": adapter,
		"ssid":    ssid,
		"powered": flag,
	})
	reply, err := nc.Request(subject, payload, core.TimeoutNormal)
	if err != nil {
		return tui.WifiActionResultMsg{Err: err.Error()}
	}
	var resp struct {
		Snapshot wifi.Snapshot `json:"snapshot"`
		Error    string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return tui.WifiActionResultMsg{Err: err.Error()}
	}
	if resp.Error != "" {
		return tui.WifiActionResultMsg{Err: resp.Error}
	}
	return tui.WifiActionResultMsg{Snapshot: resp.Snapshot}
}

// wifiRequest sends a wifi action request to darkd and returns the
// refreshed snapshot from the reply. Used by every action command.
func wifiRequest(nc *nats.Conn, subject, adapter, ssid string) (wifi.Snapshot, error) {
	payload, _ := json.Marshal(map[string]string{"adapter": adapter, "ssid": ssid})
	reply, err := nc.Request(subject, payload, core.TimeoutConnect)
	if err != nil {
		return wifi.Snapshot{}, err
	}
	var resp struct {
		Snapshot wifi.Snapshot `json:"snapshot"`
		Error    string        `json:"error,omitempty"`
	}
	if err := json.Unmarshal(reply.Data, &resp); err != nil {
		return wifi.Snapshot{}, err
	}
	if resp.Error != "" {
		return wifi.Snapshot{}, fmt.Errorf("%s", resp.Error)
	}
	return resp.Snapshot, nil
}
