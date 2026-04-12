package core

type SettingsSection struct {
	ID    string
	Label string
	Icon  string
}

func SettingsSections() []SettingsSection {
	return []SettingsSection{
		{"wifi", "Wi-Fi", "󰖩"},
		{"bluetooth", "Bluetooth", "󰂯"},
		{"network", "Network", "󰛳"},
		{"display", "Displays", "󰍹"},
		{"sound", "Sound", "󰕾"},
		{"power", "Power", "󰂄"},
		{"input", "Input Devices", "󰌌"},
		{"appearance", "Appearance", "󰸉"},
		{"workspaces", "Workspaces", "󰕮"},
		{"notifications", "Notifications", "󰂚"},
		{"privacy", "Privacy", "󰒃"},
		{"users", "Users", "󰀉"},
		{"datetime", "Date & Time", "󰃰"},
		{"about", "About", "󰋽"},
	}
}
