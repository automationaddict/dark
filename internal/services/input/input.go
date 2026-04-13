package input

type Snapshot struct {
	Keyboards []Keyboard `json:"keyboards"`
	Touchpads []Touchpad `json:"touchpads"`
	Mice      []Mouse    `json:"mice"`
	Others    []Device   `json:"others"`
	LEDs      []LED      `json:"leds"`
	Config    InputConfig `json:"config"`
}

type Device struct {
	Name      string `json:"name"`
	Event     string `json:"event"`
	Bus       string `json:"bus"`
	VendorID  string `json:"vendor_id"`
	ProductID string `json:"product_id"`
	Phys      string `json:"phys"`
	Uniq      string `json:"uniq"`
	Inhibited bool   `json:"inhibited"`
}

type Keyboard struct {
	Device
	HasLEDs bool `json:"has_leds"`
}

type Touchpad struct {
	Device
}

type Mouse struct {
	Device
}

type LED struct {
	Name          string `json:"name"`
	Brightness    int    `json:"brightness"`
	MaxBrightness int    `json:"max_brightness"`
}

type InputConfig struct {
	// Keyboard
	KBLayout       string `json:"kb_layout"`
	KBVariant      string `json:"kb_variant"`
	KBModel        string `json:"kb_model"`
	KBOptions      string `json:"kb_options"`
	RepeatRate     int    `json:"repeat_rate"`
	RepeatDelay    int    `json:"repeat_delay"`
	NumlockDefault bool   `json:"numlock_default"`

	// Mouse
	Sensitivity  float64 `json:"sensitivity"`
	AccelProfile string  `json:"accel_profile"`
	ForceNoAccel bool    `json:"force_no_accel"`
	LeftHanded   bool    `json:"left_handed"`
	ScrollMethod string  `json:"scroll_method"`
	FollowMouse  int     `json:"follow_mouse"`

	// Touchpad
	NaturalScroll       bool    `json:"natural_scroll"`
	ScrollFactor        float64 `json:"scroll_factor"`
	DisableWhileTyping  bool    `json:"disable_while_typing"`
	TapToClick          bool    `json:"tap_to_click"`
	TapAndDrag          bool    `json:"tap_and_drag"`
	DragLock            bool    `json:"drag_lock"`
	MiddleButtonEmu     bool    `json:"middle_button_emulation"`
	ClickfingerBehavior bool    `json:"clickfinger_behavior"`
}

func ReadSnapshot() Snapshot {
	devices := parseInputDevices()
	var s Snapshot

	for _, d := range devices {
		switch classifyDevice(d) {
		case "keyboard":
			s.Keyboards = append(s.Keyboards, Keyboard{
				Device:  d,
				HasLEDs: hasCapability(d, "led"),
			})
		case "touchpad":
			s.Touchpads = append(s.Touchpads, Touchpad{Device: d})
		case "mouse":
			s.Mice = append(s.Mice, Mouse{Device: d})
		case "other":
			s.Others = append(s.Others, d)
		}
	}

	s.LEDs = readLEDs()
	s.Config = readHyprlandConfig()
	return s
}
