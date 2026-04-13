package wifi

import (
	"strings"

	"github.com/godbus/dbus/v5"
)

func isAlreadyScanning(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "InProgress") || strings.Contains(msg, "Busy")
}

// Typed variant extractors shared by Augment and the agent.

func stringOpt(m map[string]dbus.Variant, key string, dst *string) {
	if v, ok := m[key]; ok {
		if s, ok := v.Value().(string); ok {
			*dst = s
		}
	}
}

func boolOpt(m map[string]dbus.Variant, key string, dst *bool) {
	if v, ok := m[key]; ok {
		if b, ok := v.Value().(bool); ok {
			*dst = b
		}
	}
}

func uint16Opt(m map[string]dbus.Variant, key string, dst *uint16) {
	if v, ok := m[key]; ok {
		if n, ok := v.Value().(uint16); ok {
			*dst = n
		}
	}
}

func uint32Opt(m map[string]dbus.Variant, key string, dst *uint32) {
	if v, ok := m[key]; ok {
		if n, ok := v.Value().(uint32); ok {
			*dst = n
		}
	}
}

func int16Opt(m map[string]dbus.Variant, key string, dst *int16) {
	if v, ok := m[key]; ok {
		if n, ok := v.Value().(int16); ok {
			*dst = n
		}
	}
}
