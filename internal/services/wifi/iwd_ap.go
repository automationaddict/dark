package wifi

import (
	"fmt"
	"time"
)

// StartAP switches the device to AP mode if necessary, then starts an
// access point with the given SSID and passphrase. The caller is
// responsible for any prior disconnect; iwd performs its own teardown
// when the mode transitions.
func (b *iwdBackend) StartAP(ifaceName, ssid, passphrase string) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	if ssid == "" {
		return fmt.Errorf("iwd: missing ssid for AP start")
	}

	// Look up current mode; only switch if needed so we don't perturb
	// a device that's already correctly configured.
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	dev := b.conn.Object(iwdBusName, devPath)
	modeProp, err := dev.GetProperty(iwdIfDevice + ".Mode")
	if err != nil {
		return fmt.Errorf("read Device.Mode: %w", err)
	}
	currentMode, _ := modeProp.Value().(string)

	if currentMode != "ap" {
		if err := b.SetMode(ifaceName, "ap"); err != nil {
			return err
		}
		// iwd takes a moment to tear down the station and bring up
		// the AccessPoint interface on the same device path.
		time.Sleep(iwdAPTransitionWait)
	}

	call := b.conn.Object(iwdBusName, devPath).Call(iwdIfAP+".Start", 0, ssid, passphrase)
	if call.Err != nil {
		return fmt.Errorf("iwd ap start: %w", call.Err)
	}
	return nil
}

// StopAP stops a running access point and switches the device back to
// station mode so the normal scan / connect flow works again.
func (b *iwdBackend) StopAP(ifaceName string) error {
	if b.conn == nil {
		return fmt.Errorf("iwd: no D-Bus connection")
	}
	devPath, err := b.findDevicePath(ifaceName)
	if err != nil {
		return err
	}
	// Stop is legal even if no AP is currently running — iwd returns
	// an error which we surface up to the TUI.
	call := b.conn.Object(iwdBusName, devPath).Call(iwdIfAP+".Stop", 0)
	if call.Err != nil {
		return fmt.Errorf("iwd ap stop: %w", call.Err)
	}
	// Move the device back to station mode so the Networks / Known
	// Networks UI becomes meaningful again. If this fails the device
	// is stuck in AP mode with no AP running, so surface it.
	if err := b.SetMode(ifaceName, "station"); err != nil {
		return fmt.Errorf("restore station mode after ap stop: %w", err)
	}
	return nil
}
