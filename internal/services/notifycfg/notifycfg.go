package notifycfg

import "os/exec"

type Snapshot struct {
	Running      bool          `json:"running"`
	Daemon       string        `json:"daemon"`
	Anchor       string        `json:"anchor"`
	TimeoutMS    int           `json:"timeout_ms"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	Padding      string        `json:"padding"`
	BorderSize   int           `json:"border_size"`
	BorderRadius int           `json:"border_radius"`
	Font         string        `json:"font"`
	MaxIcon      int           `json:"max_icon"`
	MaxHistory   int           `json:"max_history"`
	MaxVisible   int           `json:"max_visible"`
	Icons        bool          `json:"icons"`
	Markup       bool          `json:"markup"`
	Actions      bool          `json:"actions"`
	TextColor    string        `json:"text_color"`
	BorderColor  string        `json:"border_color"`
	BgColor      string        `json:"bg_color"`
	Layer        string        `json:"layer"`
	DNDActive    bool          `json:"dnd_active"`
	NotifySound  string        `json:"notify_sound"`
	GroupFormat  string        `json:"group_format"`
	LowTimeout  int           `json:"low_timeout"`
	CritTimeout int           `json:"crit_timeout"`
	CritLayer   string        `json:"crit_layer"`
	Rules        []AppRule     `json:"rules"`
	History      []HistoryItem `json:"history"`
	Sounds       []string      `json:"sounds"`
}

type AppRule struct {
	Criteria string `json:"criteria"`
	Action   string `json:"action"`
}

type HistoryItem struct {
	AppName string `json:"app_name"`
	Summary string `json:"summary"`
	Body    string `json:"body"`
	Urgency string `json:"urgency"`
}

func ReadSnapshot() Snapshot {
	s := Snapshot{
		Daemon:      "mako",
		TimeoutMS:   5000,
		Width:       300,
		Height:      100,
		BorderSize:  2,
		MaxIcon:     64,
		MaxHistory:  5,
		MaxVisible:  5,
		Icons:       true,
		Markup:      true,
		Actions:     true,
		Anchor:      "top-right",
		Layer:       "top",
		LowTimeout:  -1,
		CritTimeout: -1,
	}

	s.Running = isDaemonRunning()
	s.DNDActive = isDNDActive()

	core, theme := readMakoConfigs()
	for k, v := range core {
		s.applyGlobal(k, v)
	}
	for k, v := range theme {
		s.applyGlobal(k, v)
	}

	s.Rules = parseRules()
	parseUrgencyAndNotify(&s)
	s.History = readHistory()
	s.Sounds = listSounds()

	return s
}

func reloadMako() error {
	return exec.Command("makoctl", "reload").Run()
}
