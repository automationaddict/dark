package display

import "fmt"

// Monitor is one output head reported by Hyprland. Fields map to the
// JSON output of `hyprctl monitors -j`.
type Monitor struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Make        string  `json:"make,omitempty"`
	Model       string  `json:"model,omitempty"`
	Serial      string  `json:"serial,omitempty"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	RefreshRate float64 `json:"refreshRate"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	Scale       float64 `json:"scale"`
	Transform   int     `json:"transform"`
	Focused     bool    `json:"focused"`
	DpmsStatus  bool    `json:"dpmsStatus"`
	Vrr         bool    `json:"vrr"`
	Disabled    bool    `json:"disabled"`
	MirrorOf    string  `json:"mirrorOf,omitempty"`

	ActiveWorkspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"activeWorkspace"`

	AvailableModes []string `json:"availableModes"`
}

// Resolution returns a formatted "WxH" string.
func (m Monitor) Resolution() string {
	return fmt.Sprintf("%dx%d", m.Width, m.Height)
}

// RefreshRateHz returns the rate rounded to two decimal places as a
// display string like "60.00Hz".
func (m Monitor) RefreshRateHz() string {
	return fmt.Sprintf("%.2fHz", m.RefreshRate)
}

// TransformLabel returns a human-readable name for the Hyprland
// transform integer.
func (m Monitor) TransformLabel() string {
	switch m.Transform {
	case 0:
		return "Normal"
	case 1:
		return "90°"
	case 2:
		return "180°"
	case 3:
		return "270°"
	case 4:
		return "Flipped"
	case 5:
		return "Flipped 90°"
	case 6:
		return "Flipped 180°"
	case 7:
		return "Flipped 270°"
	default:
		return fmt.Sprintf("Unknown (%d)", m.Transform)
	}
}

// Snapshot is the display domain payload published on the bus.
type Snapshot struct {
	Monitors []Monitor `json:"monitors"`
}

// Service owns the chosen Backend and is the single entry point the
// daemon uses to read or mutate display state.
type Service struct {
	backend Backend
}

func NewService() (*Service, error) {
	backend, err := newHyprlandBackend()
	if err != nil {
		return &Service{backend: newNoopBackend()}, err
	}
	return &Service{backend: backend}, nil
}

func (s *Service) Close() {
	if s.backend != nil {
		s.backend.Close()
		s.backend = nil
	}
}

func (s *Service) Snapshot() Snapshot {
	if s.backend == nil {
		return Snapshot{}
	}
	return s.backend.Snapshot()
}

func (s *Service) SetResolution(name string, width, height int, refreshRate float64) error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.SetResolution(name, width, height, refreshRate)
}

func (s *Service) SetScale(name string, scale float64) error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.SetScale(name, scale)
}

func (s *Service) SetTransform(name string, transform int) error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.SetTransform(name, transform)
}

func (s *Service) SetPosition(name string, x, y int) error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.SetPosition(name, x, y)
}

func (s *Service) SetDpms(name string, on bool) error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.SetDpms(name, on)
}

func (s *Service) SetVrr(name string, mode int) error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.SetVrr(name, mode)
}

func (s *Service) SetMirror(name, mirrorOf string) error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.SetMirror(name, mirrorOf)
}

func (s *Service) ToggleEnabled(name string) error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.ToggleEnabled(name)
}

func (s *Service) Identify() error {
	if s.backend == nil {
		return ErrBackendUnavailable
	}
	return s.backend.Identify()
}

var ErrBackendUnavailable = fmt.Errorf("display backend unavailable")
