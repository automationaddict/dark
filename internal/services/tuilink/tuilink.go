package tuilink

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type TUIApp struct {
	Name    string
	Command string
	Style   string // "float" or "tile"
	Icon    string
}

func desktopDir() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "applications")
}

func iconDir() string {
	return filepath.Join(desktopDir(), "icons")
}

func ListTUIApps() ([]TUIApp, error) {
	dir := desktopDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var apps []TUIApp
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".desktop") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		app, ok := parseDesktopFile(path)
		if ok {
			apps = append(apps, app)
		}
	}
	return apps, nil
}

func Install(name, command, style, iconURL string) error {
	if style != "float" && style != "tile" {
		style = "float"
	}

	if err := os.MkdirAll(iconDir(), 0o755); err != nil {
		return err
	}

	iconPath := filepath.Join(iconDir(), name+".png")
	if iconURL != "" {
		fetchIcon(iconURL, iconPath)
	}

	appClass := "TUI." + style
	desktop := filepath.Join(desktopDir(), name+".desktop")
	content := fmt.Sprintf(`[Desktop Entry]
Version=1.0
Name=%s
Comment=%s
Exec=xdg-terminal-exec --app-id=%s -e %s
Terminal=false
Type=Application
Icon=%s
StartupNotify=true
`, name, name, appClass, command, iconPath)

	return os.WriteFile(desktop, []byte(content), 0o755)
}

func Remove(name string) error {
	desktop := filepath.Join(desktopDir(), name+".desktop")
	icon := filepath.Join(iconDir(), name+".png")
	os.Remove(icon)
	return os.Remove(desktop)
}

func fetchIcon(url, dest string) {
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return
	}

	f, err := os.Create(dest)
	if err != nil {
		return
	}
	defer f.Close()
	io.Copy(f, resp.Body)
}

func parseDesktopFile(path string) (TUIApp, bool) {
	f, err := os.Open(path)
	if err != nil {
		return TUIApp{}, false
	}
	defer f.Close()

	var name, exec, icon string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "Name="):
			name = strings.TrimPrefix(line, "Name=")
		case strings.HasPrefix(line, "Exec="):
			exec = strings.TrimPrefix(line, "Exec=")
		case strings.HasPrefix(line, "Icon="):
			icon = strings.TrimPrefix(line, "Icon=")
		}
	}

	if !strings.Contains(exec, "xdg-terminal-exec --app-id=TUI.") {
		return TUIApp{}, false
	}

	command, style := parseExec(exec)
	return TUIApp{Name: name, Command: command, Style: style, Icon: icon}, true
}

func parseExec(exec string) (command, style string) {
	style = "float"
	if strings.Contains(exec, "TUI.tile") {
		style = "tile"
	}

	idx := strings.Index(exec, " -e ")
	if idx < 0 {
		return exec, style
	}
	return strings.TrimSpace(exec[idx+4:]), style
}
