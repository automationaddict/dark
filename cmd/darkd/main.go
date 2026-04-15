package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/automationaddict/dark/internal/bus"
	"github.com/automationaddict/dark/internal/core"
	"github.com/automationaddict/dark/internal/lock"
	"github.com/automationaddict/dark/internal/logging"
	"github.com/automationaddict/dark/internal/scripting"
	appstoresvc "github.com/automationaddict/dark/internal/services/appstore"
	fwsvc "github.com/automationaddict/dark/internal/services/firmware"
	liminesvc "github.com/automationaddict/dark/internal/services/limine"
	audiosvc "github.com/automationaddict/dark/internal/services/audio"
	btsvc "github.com/automationaddict/dark/internal/services/bluetooth"
	displaysvc "github.com/automationaddict/dark/internal/services/display"
	netsvc "github.com/automationaddict/dark/internal/services/network"
	"github.com/automationaddict/dark/internal/services/sysinfo"
	wifisvc "github.com/automationaddict/dark/internal/services/wifi"
)

type heartbeatMsg struct {
	Time    time.Time `json:"time"`
	Seq     uint64    `json:"seq"`
	Version string    `json:"version"`
}

func main() {
	logging.Setup("darkd")
	lock.LogWarn = func(op string, err error) {
		slog.Warn("lock lifecycle", "op", op, "error", err)
	}

	lk, err := lock.Acquire("darkd")
	if err != nil {
		fmt.Fprintln(os.Stderr, "darkd:", err)
		os.Exit(1)
	}
	defer lk.Release()

	srv, nc, err := bus.StartServer()
	if err != nil {
		fmt.Fprintln(os.Stderr, "darkd:", err)
		os.Exit(1)
	}
	defer func() {
		nc.Drain()
		srv.Shutdown()
		bus.CleanupDiscoveryFile()
	}()

	binPath, err := os.Executable()
	if err != nil {
		binPath = os.Args[0]
	}

	dn := newDaemonNotifier()
	defer dn.Close()

	wifiService, err := wifisvc.NewService()
	if err != nil {
		dn.Warn("Wi-Fi", fmt.Sprintf("%v — falling back to sysfs-only detection", err))
	}
	if wifiService != nil {
		defer wifiService.Close()
		if err := wifiService.StartAgent(); err != nil {
			dn.Warn("Wi-Fi", fmt.Sprintf("iwd agent failed: %v — unknown-network connects will fail", err))
		} else {
			slog.Info("agent registered", "service", "wifi", "agent", "iwd")
		}
	}

	bluetoothService, err := btsvc.NewService()
	if err != nil {
		dn.Warn("Bluetooth", fmt.Sprintf("%v — bluetooth controls unavailable", err))
	}
	if bluetoothService != nil {
		defer bluetoothService.Close()
		if err := bluetoothService.StartAgent(); err != nil {
			dn.Warn("Bluetooth", fmt.Sprintf("bluez agent failed: %v — pairing prompts will not be handled", err))
		} else {
			slog.Info("agent registered", "service", "bluetooth", "agent", "bluez")
		}
	}

	audioService, err := audiosvc.NewService()
	if err != nil {
		dn.Warn("Sound", fmt.Sprintf("%v — sound controls unavailable", err))
	}
	if audioService != nil {
		defer audioService.Close()
		slog.Info("service connected", "service", "audio", "backend", "pulse")
	}

	networkService, err := netsvc.NewService()
	if err != nil {
		dn.Warn("Network", fmt.Sprintf("%v — network backend unavailable", err))
	}
	if networkService != nil {
		defer networkService.Close()
	}

	displayService, err := displaysvc.NewService()
	if err != nil {
		dn.Warn("Displays", fmt.Sprintf("%v — display controls unavailable", err))
	}
	if displayService != nil {
		defer displayService.Close()
		slog.Info("service connected", "service", "display", "backend", "hyprland")
	} else {
		displayService = &displaysvc.Service{}
	}

	appstoreLog := appstoreLogger()
	scriptEngine := scripting.New(appstoreLog)
	defer scriptEngine.Close()
	scriptEngine.SetRequester(func(subject string, data []byte) ([]byte, error) {
		reply, err := nc.Request(subject, data, core.TimeoutPkexec)
		if err != nil {
			return nil, err
		}
		return reply.Data, nil
	})
	scriptEngine.SetNotifier(func(summary, body, urgency string) {
		dn.Script(summary, body, urgency)
	})
	registerScriptActions(scriptEngine)
	scripting.SeedExampleScripts(scriptEngine)
	scripting.LoadAllUserScripts(scriptEngine)
	appstoreService := appstoresvc.NewService(appstoreLog, scriptEngine)
	defer appstoreService.Close()
	appstoreLog.Info("appstore: service ready")

	slog.Info("listening", "url", srv.ClientURL(), "discovery", bus.DiscoveryPath())

	// Reply handler for on-demand sysinfo requests so a freshly launched
	// TUI doesn't have to wait for the next periodic publish to render.
	if _, err := nc.Subscribe(bus.SubjectSystemInfoCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(sysinfo.Gather(binPath))
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectSystemInfoCmd, "error", err); os.Exit(1)
	}

	// Wifi snapshot responder. We reuse the long-lived Service built at
	// startup so every poll reuses the same D-Bus connection.
	if _, err := nc.Subscribe(bus.SubjectWifiAdaptersCmd, func(m *nats.Msg) {
		data, _ := json.Marshal(snapshotWifi(wifiService))
		respond(m, data)
	}); err != nil {
		slog.Error("subscribe failed", "subject", bus.SubjectWifiAdaptersCmd, "error", err); os.Exit(1)
	}

	// Wifi action responders. Each one parses a request, delegates to a
	// handler that talks to the wifi service, then broadcasts the
	// refreshed snapshot so every connected TUI updates in lockstep.
	registerWifiAction := func(subject string, handler func(*wifisvc.Service, wifiActionRequest) wifiActionResponse) {
		if _, err := nc.Subscribe(subject, func(m *nats.Msg) {
			var req wifiActionRequest
			if err := json.Unmarshal(m.Data, &req); err != nil {
				resp := wifiActionResponse{Error: "malformed request: " + err.Error()}
				data, _ := json.Marshal(resp)
				respond(m, data)
				return
			}
			resp := handler(wifiService, req)
			data, _ := json.Marshal(resp)
			if err := m.Respond(data); err != nil {
				dn.Error("Wi-Fi", "failed to send response: "+err.Error())
			}
			if resp.Error == "" {
				snapData, _ := json.Marshal(resp.Snapshot)
				if err := nc.Publish(bus.SubjectWifiAdapters, snapData); err != nil {
					dn.Error("Wi-Fi", "failed to publish snapshot: "+err.Error())
				}
			}
		}); err != nil {
			slog.Error("subscribe failed", "subject", subject, "error", err); os.Exit(1)
		}
	}

	registerWifiAction(bus.SubjectWifiScanCmd, func(svc *wifisvc.Service, req wifiActionRequest) wifiActionResponse {
		return handleScan(svc, req.Adapter)
	})
	registerWifiAction(bus.SubjectWifiConnectCmd, handleConnect)
	registerWifiAction(bus.SubjectWifiDisconnectCmd, handleDisconnect)
	registerWifiAction(bus.SubjectWifiForgetCmd, handleForget)
	registerWifiAction(bus.SubjectWifiPowerCmd, handlePower)
	registerWifiAction(bus.SubjectWifiAutoconnectCmd, handleAutoconnect)
	registerWifiAction(bus.SubjectWifiConnectHiddenCmd, handleConnectHidden)
	registerWifiAction(bus.SubjectWifiAPStartCmd, handleAPStart)
	registerWifiAction(bus.SubjectWifiAPStopCmd, handleAPStop)

	publishBluetooth := wireBluetooth(nc, bluetoothService, dn)
	publishAudio := wireAudio(nc, audioService, dn)
	publishDisplay := wireDisplay(nc, displayService, dn)
	publishNetwork := wireNetwork(nc, networkService, dn)
	publishDateTime := wireDateTime(nc, dn)
	publishNotifyCfg := wireNotifyCfg(nc, dn)
	publishInput := wireInput(nc, dn)
	publishPower := wirePower(nc, dn)
	publishAppstore := wireAppstore(nc, appstoreService, scriptEngine, appstoreLog, dn)
	publishKeybind := wireKeybind(nc, dn)
	publishUsers := wireUsers(nc, dn)
	publishPrivacy := wirePrivacy(nc, dn)
	publishAppearance := wireAppearance(nc, dn)
	publishUpdate := wireUpdate(nc, dn)

	firmwareService, err := fwsvc.NewService()
	if err != nil {
		dn.Warn("Firmware", fmt.Sprintf("%v — firmware controls unavailable", err))
	}
	if firmwareService != nil {
		defer firmwareService.Close()
	}
	publishFirmware := wireFirmware(nc, firmwareService, dn)

	limineService, err := liminesvc.NewService()
	if err != nil {
		dn.Warn("Limine", fmt.Sprintf("%v — limine controls unavailable", err))
	}
	if limineService != nil {
		defer limineService.Close()
	}
	publishLimine := wireLimine(nc, limineService, dn)

	publishScreensaver := wireScreensaver(nc)

	publishTopBar := wireTopBar(nc)

	publishWorkspaces := wireWorkspaces(nc)

	publishDarkUpdate := wireDarkUpdate(nc)

	wireScripting(nc, scriptEngine)
	wireScriptEvents(nc, scriptEngine)
	wireScriptClientEvents(nc, scriptEngine)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	heartbeat := time.NewTicker(core.TickHeartbeat)
	defer heartbeat.Stop()

	sysTick := time.NewTicker(core.TickSysInfo)
	defer sysTick.Stop()

	wifiTick := time.NewTicker(core.TickWifi)
	defer wifiTick.Stop()

	bluetoothTick := time.NewTicker(core.TickBluetooth)
	defer bluetoothTick.Stop()

	audioTick := time.NewTicker(core.TickAudio)
	defer audioTick.Stop()

	displayTick := time.NewTicker(core.TickDisplay)
	defer displayTick.Stop()

	networkTick := time.NewTicker(core.TickNetwork)
	defer networkTick.Stop()

	appstoreTick := time.NewTicker(core.TickAppstore)
	defer appstoreTick.Stop()

	workspacesTick := time.NewTicker(core.TickWorkspaces)
	defer workspacesTick.Stop()

	// Publish initial snapshots immediately so any subscriber that
	// connects in the gap before the first tick still gets pushed data.
	publishSysInfo(nc, binPath, dn)
	publishWifi(nc, wifiService, dn)
	publishBluetooth()
	publishAudio()
	publishDisplay()
	publishNetwork()
	publishDateTime()
	publishNotifyCfg()
	publishInput()
	publishPower()
	publishAppstore()
	publishKeybind()
	publishUsers()
	publishPrivacy()
	publishAppearance()
	publishUpdate()
	publishFirmware()
	publishLimine()
	publishScreensaver()
	publishTopBar()
	publishWorkspaces()
	publishDarkUpdate()

	var seq uint64
	for {
		select {
		case sig := <-sigs:
			if sig == syscall.SIGHUP {
				slog.Info("reloading (SIGHUP)")
				publishSysInfo(nc, binPath, dn)
				publishWifi(nc, wifiService, dn)
				publishBluetooth()
				publishAudio()
				publishDisplay()
				publishNetwork()
				publishDateTime()
				publishNotifyCfg()
				publishInput()
				publishPower()
				publishAppstore()
				publishKeybind()
				publishUsers()
				publishPrivacy()
				publishAppearance()
				publishUpdate()
				publishFirmware()
				publishLimine()
				publishScreensaver()
				publishTopBar()
				publishWorkspaces()
				publishDarkUpdate()
				continue
			}
			slog.Info("shutting down", "signal", sig.String())
			go func() {
				time.Sleep(core.ShutdownTimeout)
				slog.Error("shutdown timeout — force exit")
				os.Exit(1)
			}()
			return
		case t := <-heartbeat.C:
			seq++
			data, _ := json.Marshal(heartbeatMsg{
				Time:    t,
				Seq:     seq,
				Version: sysinfo.DarkVersion,
			})
			publish(nc, bus.SubjectDaemonHeartbeat, data)
		case <-sysTick.C:
			publishSysInfo(nc, binPath, dn)
		case <-wifiTick.C:
			publishWifi(nc, wifiService, dn)
		case <-bluetoothTick.C:
			publishBluetooth()
		case <-audioTick.C:
			publishAudio()
		case <-displayTick.C:
			publishDisplay()
		case <-networkTick.C:
			publishNetwork()
		case <-appstoreTick.C:
			publishAppstore()
		case <-workspacesTick.C:
			publishWorkspaces()
		}
	}
}

func publishSysInfo(nc *nats.Conn, binPath string, dn *daemonNotifier) {
	data, err := json.Marshal(sysinfo.Gather(binPath))
	if err != nil {
		dn.Error("System", "marshal failed: "+err.Error())
		return
	}
	if err := nc.Publish(bus.SubjectSystemInfo, data); err != nil {
		dn.Error("System", "publish failed: "+err.Error())
	}
}

func publishWifi(nc *nats.Conn, svc *wifisvc.Service, dn *daemonNotifier) {
	data, err := json.Marshal(snapshotWifi(svc))
	if err != nil {
		dn.Error("Wi-Fi", "marshal failed: "+err.Error())
		return
	}
	if err := nc.Publish(bus.SubjectWifiAdapters, data); err != nil {
		dn.Error("Wi-Fi", "publish failed: "+err.Error())
	}
}
