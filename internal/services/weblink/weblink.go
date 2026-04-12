package weblink

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type WebApp struct {
	Name string
	URL  string
	Icon string
}

func desktopDir() string {
	return filepath.Join(os.Getenv("HOME"), ".local", "share", "applications")
}

func iconDir() string {
	return filepath.Join(desktopDir(), "icons")
}

func ListWebApps() ([]WebApp, error) {
	dir := desktopDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var apps []WebApp
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

func Install(name, url string) error {
	if !strings.Contains(url, "://") {
		url = "https://" + url
	}

	if err := os.MkdirAll(iconDir(), 0o755); err != nil {
		return err
	}

	iconPath := filepath.Join(iconDir(), name+".png")
	fetchFavicon(url, iconPath)

	desktop := filepath.Join(desktopDir(), name+".desktop")
	content := fmt.Sprintf(`[Desktop Entry]
Version=1.0
Name=%s
Comment=%s
Exec=omarchy-launch-webapp %s
Terminal=false
Type=Application
Icon=%s
StartupNotify=true
`, name, name, url, iconPath)

	return os.WriteFile(desktop, []byte(content), 0o755)
}

func Remove(name string) error {
	desktop := filepath.Join(desktopDir(), name+".desktop")
	icon := filepath.Join(iconDir(), name+".png")
	os.Remove(icon)
	return os.Remove(desktop)
}

func fetchFavicon(url, dest string) {
	domain := url
	if i := strings.Index(url, "://"); i >= 0 {
		domain = url[i+3:]
	}
	if i := strings.Index(domain, "/"); i >= 0 {
		domain = domain[:i]
	}

	faviconURL := "https://www.google.com/s2/favicons?domain=" + domain + "&sz=128"
	resp, err := http.Get(faviconURL)
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

func parseDesktopFile(path string) (WebApp, bool) {
	f, err := os.Open(path)
	if err != nil {
		return WebApp{}, false
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

	if !strings.Contains(exec, "omarchy-launch-webapp") && !strings.Contains(exec, "omarchy-webapp-handler") {
		return WebApp{}, false
	}

	url := extractURL(exec)
	return WebApp{Name: name, URL: url, Icon: icon}, true
}

func extractURL(exec string) string {
	for _, part := range strings.Fields(exec) {
		if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
			return part
		}
	}
	return exec
}
