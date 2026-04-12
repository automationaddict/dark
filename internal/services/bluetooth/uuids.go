package bluetooth

import "strings"

// KnownUUID maps a full 128-bit Bluetooth SIG UUID string to the
// short human name for that profile or service. Only entries dark
// displays in the device-info panel live here; the full SIG list is
// thousands of entries long and most are never surfaced to users.
//
// Entries come from:
//   - Bluetooth SIG Assigned Numbers (profile identifiers)
//   - BlueZ lib/uuid.h
//
// Coverage prioritizes audio (A2DP/AVRCP/HFP/HSP), HID, PAN, and the
// common GATT services a user-facing tool would actually recognize.
var knownUUIDs = map[string]string{
	// Classic profile UUIDs (base 00001000-...-00805f9b34fb)
	"00001000-0000-1000-8000-00805f9b34fb": "Service Discovery",
	"00001101-0000-1000-8000-00805f9b34fb": "Serial Port",
	"00001105-0000-1000-8000-00805f9b34fb": "OBEX Object Push",
	"00001106-0000-1000-8000-00805f9b34fb": "OBEX File Transfer",
	"00001108-0000-1000-8000-00805f9b34fb": "Headset (HSP)",
	"0000110a-0000-1000-8000-00805f9b34fb": "Audio Source (A2DP)",
	"0000110b-0000-1000-8000-00805f9b34fb": "Audio Sink (A2DP)",
	"0000110c-0000-1000-8000-00805f9b34fb": "AVRCP Target",
	"0000110d-0000-1000-8000-00805f9b34fb": "Advanced Audio Distribution",
	"0000110e-0000-1000-8000-00805f9b34fb": "AVRCP Controller",
	"0000110f-0000-1000-8000-00805f9b34fb": "AVRCP",
	"00001112-0000-1000-8000-00805f9b34fb": "Headset Audio Gateway",
	"00001115-0000-1000-8000-00805f9b34fb": "PAN (PANU)",
	"00001116-0000-1000-8000-00805f9b34fb": "PAN (NAP)",
	"00001117-0000-1000-8000-00805f9b34fb": "PAN (GN)",
	"0000111e-0000-1000-8000-00805f9b34fb": "Handsfree (HFP)",
	"0000111f-0000-1000-8000-00805f9b34fb": "Handsfree Audio Gateway",
	"00001124-0000-1000-8000-00805f9b34fb": "Human Interface Device",
	"00001132-0000-1000-8000-00805f9b34fb": "Message Access",
	"00001133-0000-1000-8000-00805f9b34fb": "Message Notification",
	"00001200-0000-1000-8000-00805f9b34fb": "PnP Information",
	"00001203-0000-1000-8000-00805f9b34fb": "Generic Audio",
	"00001800-0000-1000-8000-00805f9b34fb": "Generic Access",
	"00001801-0000-1000-8000-00805f9b34fb": "Generic Attribute",

	// GATT services
	"0000180a-0000-1000-8000-00805f9b34fb": "Device Information",
	"0000180f-0000-1000-8000-00805f9b34fb": "Battery Service",
	"00001812-0000-1000-8000-00805f9b34fb": "HID over GATT",
	"0000181d-0000-1000-8000-00805f9b34fb": "Weight Scale",
	"0000180d-0000-1000-8000-00805f9b34fb": "Heart Rate",
	"00001802-0000-1000-8000-00805f9b34fb": "Immediate Alert",
	"00001803-0000-1000-8000-00805f9b34fb": "Link Loss",
	"00001804-0000-1000-8000-00805f9b34fb": "Tx Power",
}

// LookupUUIDName returns a short human name for a UUID string, or "" if
// the UUID is not in the curated list. Comparison is case-insensitive.
func LookupUUIDName(uuid string) string {
	return knownUUIDs[strings.ToLower(uuid)]
}

// MajorClassFromClass extracts the major device class from a BlueZ
// Device1.Class value and returns a short label. The Class of Device
// layout is documented in the Bluetooth SIG Assigned Numbers
// (baseband section) — bits 8–12 hold the major class.
func MajorClassFromClass(class uint32) string {
	if class == 0 {
		return ""
	}
	major := (class >> 8) & 0x1f
	switch major {
	case 0x00:
		return "Miscellaneous"
	case 0x01:
		return "Computer"
	case 0x02:
		return "Phone"
	case 0x03:
		return "LAN / Network Access Point"
	case 0x04:
		return "Audio / Video"
	case 0x05:
		return "Peripheral"
	case 0x06:
		return "Imaging"
	case 0x07:
		return "Wearable"
	case 0x08:
		return "Toy"
	case 0x09:
		return "Health"
	case 0x1f:
		return "Uncategorized"
	default:
		return "Unknown"
	}
}
