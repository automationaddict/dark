package links

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type WebLink struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type TUILink struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Style   string `yaml:"style"`
}

type HelpLink struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type LinksFile struct {
	WebLinks  []WebLink  `yaml:"web_links"`
	TUILinks  []TUILink  `yaml:"tui_links"`
	HelpLinks []HelpLink `yaml:"help_links"`
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "dark", "links.yaml")
}

func Load() (LinksFile, error) {
	path := configPath()
	if path == "" {
		return LinksFile{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file yet — import from .desktop files
			lf := importFromDesktop()
			if len(lf.WebLinks) > 0 || len(lf.TUILinks) > 0 {
				Save(lf)
			}
			return lf, nil
		}
		return LinksFile{}, err
	}
	var lf LinksFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return LinksFile{}, err
	}
	// If web/tui links are empty, try importing from .desktop files
	if len(lf.WebLinks) == 0 && len(lf.TUILinks) == 0 {
		imported := importFromDesktop()
		if len(imported.WebLinks) > 0 || len(imported.TUILinks) > 0 {
			lf.WebLinks = imported.WebLinks
			lf.TUILinks = imported.TUILinks
			Save(lf)
		}
	}
	return lf, nil
}

func Save(lf LinksFile) error {
	path := configPath()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(&lf)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func AddWebLink(name, url string) error {
	lf, _ := Load()
	lf.WebLinks = append(lf.WebLinks, WebLink{Name: name, URL: url})
	return Save(lf)
}

func RemoveWebLink(name string) error {
	lf, _ := Load()
	out := lf.WebLinks[:0]
	for _, l := range lf.WebLinks {
		if l.Name != name {
			out = append(out, l)
		}
	}
	lf.WebLinks = out
	return Save(lf)
}

func AddTUILink(name, command, style string) error {
	lf, _ := Load()
	if style != "float" && style != "tile" {
		style = "float"
	}
	lf.TUILinks = append(lf.TUILinks, TUILink{Name: name, Command: command, Style: style})
	return Save(lf)
}

func RemoveTUILink(name string) error {
	lf, _ := Load()
	out := lf.TUILinks[:0]
	for _, l := range lf.TUILinks {
		if l.Name != name {
			out = append(out, l)
		}
	}
	lf.TUILinks = out
	return Save(lf)
}

func AddHelpLink(name, url string) error {
	lf, _ := Load()
	lf.HelpLinks = append(lf.HelpLinks, HelpLink{Name: name, URL: url})
	return Save(lf)
}

func RemoveHelpLink(name string) error {
	lf, _ := Load()
	out := lf.HelpLinks[:0]
	for _, l := range lf.HelpLinks {
		if l.Name != name {
			out = append(out, l)
		}
	}
	lf.HelpLinks = out
	return Save(lf)
}

func desktopDir() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	return filepath.Join(home, ".local", "share", "applications")
}

// importFromDesktop reads .desktop files and extracts web links and
// TUI links based on their Exec lines.
func importFromDesktop() LinksFile {
	var lf LinksFile
	dir := desktopDir()
	if dir == "" {
		return lf
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return lf
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".desktop") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		name, exec := parseDesktopEntry(path)
		if name == "" || exec == "" {
			continue
		}
		if strings.Contains(exec, "omarchy-launch-webapp") || strings.Contains(exec, "omarchy-webapp-handler") {
			url := extractURL(exec)
			if url != "" {
				lf.WebLinks = append(lf.WebLinks, WebLink{Name: name, URL: url})
			}
		} else if strings.Contains(exec, "xdg-terminal-exec --app-id=TUI.") {
			command, style := parseTUIExec(exec)
			if command != "" {
				lf.TUILinks = append(lf.TUILinks, TUILink{Name: name, Command: command, Style: style})
			}
		}
	}
	return lf
}

func parseDesktopEntry(path string) (name, exec string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "Name="):
			name = strings.TrimPrefix(line, "Name=")
		case strings.HasPrefix(line, "Exec="):
			exec = strings.TrimPrefix(line, "Exec=")
		}
	}
	return name, exec
}

func extractURL(exec string) string {
	for _, part := range strings.Fields(exec) {
		if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
			return part
		}
	}
	return ""
}

func parseTUIExec(exec string) (command, style string) {
	style = "float"
	if strings.Contains(exec, "TUI.tile") {
		style = "tile"
	}
	idx := strings.Index(exec, " -e ")
	if idx < 0 {
		return "", style
	}
	return strings.TrimSpace(exec[idx+4:]), style
}
