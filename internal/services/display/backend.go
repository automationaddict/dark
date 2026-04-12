package display

// Backend abstracts the display compositor. The current implementation
// shells out to hyprctl; a future native Hyprland IPC client could
// replace it without touching the TUI or daemon.
type Backend interface {
	Name() string
	Snapshot() Snapshot
	Close()

	SetResolution(name string, width, height int, refreshRate float64) error
	SetScale(name string, scale float64) error
	SetTransform(name string, transform int) error
	SetPosition(name string, x, y int) error
	SetDpms(name string, on bool) error
	SetVrr(name string, mode int) error
	SetMirror(name, mirrorOf string) error
	ToggleEnabled(name string) error
	Identify() error

	SetBrightness(pct int) error
	SetKbdBrightness(pct int) error
	SetNightLight(enable bool, tempK int, gamma int) error
	SetGamma(pct int) error
	SaveProfile(name string) error
	ApplyProfile(name string) error
	DeleteProfile(name string) error
	Events() <-chan struct{}
}
