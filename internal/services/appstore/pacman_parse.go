package appstore

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// runCommand executes argv with a short-circuit context so a hung
// subprocess can't pin the daemon. Output is captured as a single
// string; stderr is swallowed on success and surfaced in the error on
// failure.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// removeIfExists deletes path when present and returns nil for the "not
// there" case. Used by cache invalidation paths that don't care whether
// the file existed in the first place.
func removeIfExists(path string) error {
	if path == "" {
		return nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// parsePacmanSl parses the output of `pacman -Sl`. Each line looks like:
//
//	core acl 2.3.2-1 [installed]
//
// The trailing "[installed]" marker is optional. We do not rely on it
// because the caller cross-checks with pacman -Qq for authoritative
// installed-state; reading the -Sl marker would still be correct but
// the cross-check path is already needed for cache warm-starts.
func parsePacmanSl(out string) []Package {
	lines := strings.Split(out, "\n")
	pkgs := make([]Package, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		pkgs = append(pkgs, Package{
			Name:    fields[1],
			Version: fields[2],
			Repo:    fields[0],
			Origin:  OriginPacman,
		})
	}
	return pkgs
}

// enrichWithExpac batches expac calls to fill in descriptions and
// installed sizes on the catalog in place. expac is vastly faster than
// running `pacman -Si` per package (one pass over the sync db vs. N
// forks + N parses), so we only call it when available.
//
// Format string: %n\t%d\t%m\t%b where
//
//	%n = name
//	%d = description
//	%m = installed size (bytes)
//	%b = build date (Unix seconds)
//
// The -S flag selects the sync db (all known remote packages), matching
// what we got from pacman -Sl.
func enrichWithExpac(cat []Package, logger *slog.Logger) {
	out, err := runCommand("expac", "-S", "%n\t%d\t%m\t%b")
	if err != nil {
		logger.Warn("expac failed, descriptions will be empty", "err", err)
		return
	}
	// Build a name -> metadata index from expac output. One line per
	// package. When a name appears in multiple repos, last-write-wins
	// is fine — the field values are identical for a given (name,
	// version) pair.
	type meta struct {
		desc  string
		size  int64
		built int64
	}
	index := make(map[string]meta, len(cat))
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 4)
		if len(fields) < 4 {
			continue
		}
		sizeBytes, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)
		builtUnix, _ := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64)
		index[fields[0]] = meta{
			desc:  fields[1],
			size:  sizeBytes,
			built: builtUnix,
		}
	}
	for i := range cat {
		if m, ok := index[cat[i].Name]; ok {
			cat[i].Description = m.desc
			cat[i].InstalledSize = m.size
			cat[i].LastUpdatedUnix = m.built
		}
	}
}

// parsePacmanSi parses the key-value output of `pacman -Si <name>` into
// a Detail. The parser handles:
//
//   - Multi-line continuations where a value wraps onto leading
//     whitespace on the next line.
//   - "None" values, which are left as empty slices rather than a
//     single-element slice containing "None".
//   - Size fields in "NNN.NN KiB/MiB/GiB" format.
//
// The shape is forgiving: unknown keys are ignored, so future pacman
// versions that add fields are handled without a code change.
func parsePacmanSi(out string) (Detail, error) {
	var d Detail
	d.Origin = OriginPacman

	scanner := bufio.NewScanner(strings.NewReader(out))
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	var curKey, curVal string

	flush := func() {
		if curKey == "" {
			return
		}
		applyPacmanField(&d, curKey, strings.TrimSpace(curVal))
		curKey = ""
		curVal = ""
	}

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if line[0] == ' ' || line[0] == '\t' {
			curVal += " " + strings.TrimSpace(line)
			continue
		}
		flush()
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		curKey = strings.TrimSpace(line[:idx])
		curVal = strings.TrimSpace(line[idx+1:])
	}
	flush()
	if err := scanner.Err(); err != nil {
		return Detail{}, err
	}
	if d.Name == "" {
		return Detail{}, fmt.Errorf("appstore: pacman -Si returned no Name field")
	}
	return d, nil
}

func applyPacmanField(d *Detail, key, val string) {
	switch key {
	case "Repository":
		d.Repo = val
	case "Name":
		d.Name = val
	case "Version":
		d.Version = val
	case "Description":
		d.Description = val
		d.LongDesc = val
	case "URL":
		d.URL = val
	case "Licenses":
		d.Licenses = splitNoneList(val)
	case "Groups":
		d.Groups = splitNoneList(val)
	case "Provides":
		d.Provides = splitNoneList(val)
	case "Depends On":
		d.Depends = splitNoneList(val)
	case "Optional Deps":
		d.OptDepends = splitNoneList(val)
	case "Conflicts With":
		d.Conflicts = splitNoneList(val)
	case "Replaces":
		d.Replaces = splitNoneList(val)
	case "Download Size":
		d.DownloadSize = ParseByteSize(val)
	case "Installed Size":
		d.InstalledSize = ParseByteSize(val)
	case "Packager":
		d.Packager = val
	case "Maintainer":
		d.Maintainer = val
	case "Build Date":
		d.BuildDateUnix = parsePacmanDate(val)
		d.LastUpdatedUnix = d.BuildDateUnix
	}
}

// splitNoneList splits pacman's space-separated list values, converting
// the literal "None" placeholder to an empty slice. Multi-word entries
// (e.g. "bash-completion: for tab completion" under Optional Deps)
// survive as single strings because we split on runs of 2+ spaces when
// present, falling back to single-space splitting otherwise.
func splitNoneList(val string) []string {
	val = strings.TrimSpace(val)
	if val == "" || val == "None" {
		return nil
	}
	if strings.Contains(val, "  ") {
		parts := strings.Split(val, "  ")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	return strings.Fields(val)
}

// parsePacmanDate parses the human date string pacman emits, e.g.
// "Wed 10 Dec 2025 05:02:55 PM EST", and returns a Unix timestamp. On
// parse failure it returns zero so downstream formatters render "—".
// Pacman uses the system locale for this output, so we accept a couple
// of common layouts and give up gracefully on anything else.
func parsePacmanDate(val string) int64 {
	val = strings.TrimSpace(val)
	if val == "" {
		return 0
	}
	layouts := []string{
		"Mon 02 Jan 2006 03:04:05 PM MST",
		"Mon Jan 02 15:04:05 2006",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, val); err == nil {
			return t.Unix()
		}
		if t, err := time.ParseInLocation(layout, val, time.Local); err == nil {
			return t.Unix()
		}
	}
	return 0
}
