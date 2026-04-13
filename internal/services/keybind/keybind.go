package keybind

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Source string

const (
	SourceDefault Source = "default"
	SourceUser    Source = "user"
)

type Binding struct {
	Mods       string `json:"mods"`
	Key        string `json:"key"`
	Desc       string `json:"desc"`
	Dispatcher string `json:"dispatcher"`
	Args       string `json:"args"`
	Source     Source `json:"source"`
	Category   string `json:"category"`
	BindType   string `json:"bind_type"`
}

type Snapshot struct {
	Bindings []Binding `json:"bindings"`
}

type Conflict struct {
	Existing Binding
}

type unbindEntry struct {
	Mods string
	Key  string
}

func ReadSnapshot() Snapshot {
	var allBindings []Binding
	var allUnbinds []unbindEntry

	// Parse default binding files sourced by hyprland.conf.
	// Falls back to reading all *.conf in the default dir if
	// hyprland.conf can't be parsed.
	defaultDir := defaultBindDir()
	userPath := userBindFile()
	sourced := sourcedBindFiles()
	if len(sourced) > 0 {
		for _, path := range sourced {
			if path == userPath || !strings.HasPrefix(path, defaultDir) {
				continue
			}
			bindings, _ := parseFile(path, SourceDefault)
			allBindings = append(allBindings, bindings...)
		}
	} else {
		entries, _ := os.ReadDir(defaultDir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
				continue
			}
			path := filepath.Join(defaultDir, e.Name())
			bindings, _ := parseFile(path, SourceDefault)
			allBindings = append(allBindings, bindings...)
		}
	}

	// Parse user bindings file.
	userBindings, userUnbinds := parseFile(userPath, SourceUser)
	allUnbinds = append(allUnbinds, userUnbinds...)

	// Filter out defaults that have been unbound by the user.
	if len(allUnbinds) > 0 {
		unbindSet := make(map[string]bool, len(allUnbinds))
		for _, u := range allUnbinds {
			unbindSet[bindKey(u.Mods, u.Key)] = true
		}
		filtered := allBindings[:0]
		for _, b := range allBindings {
			if !unbindSet[bindKey(b.Mods, b.Key)] {
				filtered = append(filtered, b)
			}
		}
		allBindings = filtered
	}

	allBindings = append(allBindings, userBindings...)

	return Snapshot{Bindings: allBindings}
}

func parseFile(path string, source Source) ([]Binding, []unbindEntry) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}

	category := strings.TrimSuffix(filepath.Base(path), ".conf")
	var bindings []Binding
	var unbinds []unbindEntry

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "$") {
			continue
		}

		if strings.HasPrefix(line, "unbind") {
			if u, ok := parseUnbind(line); ok {
				unbinds = append(unbinds, u)
			}
			continue
		}

		if b, ok := parseLine(line, source, category); ok {
			bindings = append(bindings, b)
		}
	}

	return bindings, unbinds
}

func parseLine(line string, source Source, category string) (Binding, bool) {
	// Match lines starting with "bind" (bindd, bindeld, bindld, bindmd, etc.)
	if !strings.HasPrefix(line, "bind") {
		return Binding{}, false
	}

	eqIdx := strings.Index(line, "=")
	if eqIdx < 0 {
		return Binding{}, false
	}

	bindType := strings.TrimSpace(line[:eqIdx])
	rest := strings.TrimSpace(line[eqIdx+1:])

	// Determine if this bind type includes a description field.
	// bindd, bindeld, bindld, etc. — the 'd' flag means "has description".
	hasDesc := strings.Contains(bindType, "d")

	// Split on commas. Format depends on description presence:
	//   with desc: MODS, KEY, Description, dispatcher, args...
	//   no desc:   MODS, KEY, dispatcher, args...
	parts := strings.SplitN(rest, ",", -1)

	minParts := 3
	if hasDesc {
		minParts = 4
	}
	if len(parts) < minParts {
		return Binding{}, false
	}

	mods := strings.TrimSpace(parts[0])
	key := strings.TrimSpace(parts[1])

	var desc, dispatcher, args string
	if hasDesc {
		desc = strings.TrimSpace(parts[2])
		dispatcher = strings.TrimSpace(parts[3])
		if len(parts) > 4 {
			args = strings.TrimSpace(strings.Join(parts[4:], ","))
		}
	} else {
		dispatcher = strings.TrimSpace(parts[2])
		if len(parts) > 3 {
			args = strings.TrimSpace(strings.Join(parts[3:], ","))
		}
	}

	return Binding{
		Mods:       mods,
		Key:        key,
		Desc:       desc,
		Dispatcher: dispatcher,
		Args:       args,
		Source:     source,
		Category:   category,
		BindType:   bindType,
	}, true
}

func parseUnbind(line string) (unbindEntry, bool) {
	eqIdx := strings.Index(line, "=")
	if eqIdx < 0 {
		return unbindEntry{}, false
	}
	rest := strings.TrimSpace(line[eqIdx+1:])
	parts := strings.SplitN(rest, ",", 2)
	if len(parts) < 2 {
		return unbindEntry{}, false
	}
	return unbindEntry{
		Mods: strings.TrimSpace(parts[0]),
		Key:  strings.TrimSpace(parts[1]),
	}, true
}

// DetectConflicts finds bindings that share the same mods+key combo.
// excludeIdx is the index to skip (for edit); pass -1 for add.
func DetectConflicts(bindings []Binding, mods, key string, excludeIdx int) []Conflict {
	target := bindKey(mods, key)
	var conflicts []Conflict
	for i, b := range bindings {
		if i == excludeIdx {
			continue
		}
		if bindKey(b.Mods, b.Key) == target {
			conflicts = append(conflicts, Conflict{Existing: b})
		}
	}
	return conflicts
}

func AddBinding(b Binding) error {
	path := userBindFile()
	line := formatBindLine(b)

	data, _ := os.ReadFile(path)
	content := string(data)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += line + "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	return reload()
}

func UpdateBinding(old, new Binding) error {
	if old.Source == SourceUser {
		return updateUserBinding(old, new)
	}
	return overrideDefaultBinding(old, new)
}

func RemoveBinding(b Binding) error {
	if b.Source == SourceUser {
		return removeUserBinding(b)
	}
	return disableDefaultBinding(b)
}

func updateUserBinding(old, new Binding) error {
	path := userBindFile()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	oldKey := bindKey(old.Mods, old.Key)
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if b, ok := parseLine(trimmed, SourceUser, ""); ok {
			if bindKey(b.Mods, b.Key) == oldKey {
				lines[i] = formatBindLine(new)
				break
			}
		}
	}

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	return reload()
}

func overrideDefaultBinding(old, new Binding) error {
	path := userBindFile()
	data, _ := os.ReadFile(path)
	content := string(data)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += formatUnbindLine(old.Mods, old.Key) + "\n"
	content += formatBindLine(new) + "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	return reload()
}

func removeUserBinding(b Binding) error {
	path := userBindFile()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	target := bindKey(b.Mods, b.Key)
	lines := strings.Split(string(data), "\n")
	var out []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if parsed, ok := parseLine(trimmed, SourceUser, ""); ok {
			if bindKey(parsed.Mods, parsed.Key) == target {
				continue
			}
		}
		out = append(out, line)
	}

	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		return err
	}
	return reload()
}

func disableDefaultBinding(b Binding) error {
	path := userBindFile()
	data, _ := os.ReadFile(path)
	content := string(data)
	if content != "" && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += formatUnbindLine(b.Mods, b.Key) + "\n"

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	return reload()
}

func formatBindLine(b Binding) string {
	bt := b.BindType
	if bt == "" {
		bt = "bindd"
	}
	if b.Desc != "" {
		if b.Args != "" {
			return bt + " = " + b.Mods + ", " + b.Key + ", " + b.Desc + ", " + b.Dispatcher + ", " + b.Args
		}
		return bt + " = " + b.Mods + ", " + b.Key + ", " + b.Desc + ", " + b.Dispatcher + ","
	}
	if b.Args != "" {
		return bt + " = " + b.Mods + ", " + b.Key + ", " + b.Dispatcher + ", " + b.Args
	}
	return bt + " = " + b.Mods + ", " + b.Key + ", " + b.Dispatcher + ","
}

func formatUnbindLine(mods, key string) string {
	return "unbind = " + mods + ", " + key
}

// bindKey returns a normalized key for mods+key comparison.
func bindKey(mods, key string) string {
	return normalizeMods(mods) + "|" + strings.ToUpper(strings.TrimSpace(key))
}

func normalizeMods(mods string) string {
	tokens := strings.Fields(mods)
	sort.Strings(tokens)
	return strings.ToUpper(strings.Join(tokens, " "))
}

func defaultBindDir() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".local", "share", "omarchy", "default", "hypr", "bindings")
}

func userBindFile() string {
	home := os.Getenv("HOME")
	return filepath.Join(home, ".config", "hypr", "bindings.conf")
}

func reload() error {
	return exec.Command("hyprctl", "reload").Run()
}

// sourcedBindFiles parses hyprland.conf for sourced binding config files.
// Only files under the default bindings directory are returned.
func sourcedBindFiles() []string {
	home := os.Getenv("HOME")
	hyprConf := filepath.Join(home, ".config", "hypr", "hyprland.conf")
	data, err := os.ReadFile(hyprConf)
	if err != nil {
		return nil
	}

	var paths []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "source") {
			continue
		}
		eqIdx := strings.Index(line, "=")
		if eqIdx < 0 {
			continue
		}
		path := strings.TrimSpace(line[eqIdx+1:])
		path = strings.ReplaceAll(path, "~", home)
		if !strings.Contains(path, "binding") && !strings.Contains(path, "bindings") {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}
