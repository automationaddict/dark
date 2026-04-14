package update

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Snapshot holds the current update state published to the TUI.
type Snapshot struct {
	CurrentVersion   string `json:"current_version"`
	AvailableVersion string `json:"available_version"`
	UpdateAvailable  bool   `json:"update_available"`
	Channel          string `json:"channel"`
}

// StepResult reports the outcome of a single update step.
type StepResult struct {
	Step   string `json:"step"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// RunResult reports the outcome of the full update.
type RunResult struct {
	Steps         []StepResult `json:"steps"`
	RebootNeeded  bool         `json:"reboot_needed"`
	Error         string       `json:"error,omitempty"`
}

func omarchyPath() string {
	if p := os.Getenv("OMARCHY_PATH"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "omarchy")
}

// Check reads the current installed version and compares it to the
// latest remote git tag to determine if an update is available.
func Check() Snapshot {
	snap := Snapshot{
		CurrentVersion: currentVersion(),
		Channel:        currentChannel(),
	}
	avail, err := availableVersion()
	if err == nil && avail != "" && avail != snap.CurrentVersion {
		snap.AvailableVersion = avail
		snap.UpdateAvailable = true
	} else {
		snap.AvailableVersion = snap.CurrentVersion
	}
	return snap
}

func currentVersion() string {
	path := filepath.Join(omarchyPath(), "version")
	data, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}

func currentChannel() string {
	out, err := exec.Command("omarchy-version-channel").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func availableVersion() (string, error) {
	oPath := omarchyPath()
	out, err := exec.Command("git", "-C", oPath, "ls-remote", "--tags", "origin").Output()
	if err != nil {
		return "", fmt.Errorf("git ls-remote: %w", err)
	}
	var latest string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "{}") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		tag := strings.TrimPrefix(parts[1], "refs/tags/")
		tag = strings.TrimPrefix(tag, "v")
		latest = tag
	}
	if latest == "" {
		return "", fmt.Errorf("no tags found")
	}
	return latest, nil
}

// Run executes the full Omarchy update sequence. All privileged
// operations (time sync, keyring, pacman, orphan removal) are batched
// into a single pkexec call so the user only authenticates once.
func Run(helperPath string) RunResult {
	var result RunResult

	// 1. Git pull (unprivileged)
	r := stepGitPull()
	result.Steps = append(result.Steps, r)

	// 2. All privileged steps in one pkexec call
	r = stepPrivileged(helperPath)
	result.Steps = append(result.Steps, r)
	if r.Error != "" {
		result.Error = "System update failed: " + r.Error
		return result
	}

	// 3. Reset waybar indicator (unprivileged)
	exec.Command("pkill", "-RTMIN+7", "waybar").Run()

	// 4. Run migrations (unprivileged)
	r = stepMigrations()
	result.Steps = append(result.Steps, r)

	// 5. AUR packages (unprivileged — AUR helpers run as user)
	r = stepAURPkgs()
	result.Steps = append(result.Steps, r)

	// 6. Post-update hook (unprivileged)
	r = stepPostHook()
	result.Steps = append(result.Steps, r)

	result.RebootNeeded = checkRebootNeeded()
	return result
}

func stepGitPull() StepResult {
	oPath := omarchyPath()
	cmd := exec.Command("git", "-C", oPath, "pull", "--autostash")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		exec.Command("git", "-C", oPath, "reset", "--merge").Run()
		return StepResult{Step: "Update Omarchy", Output: stdout.String(), Error: stderr.String()}
	}
	return StepResult{Step: "Update Omarchy", Output: stdout.String()}
}

func stepPrivileged(helperPath string) StepResult {
	out, err := runHelper(helperPath, "update-full")
	if err != nil {
		return StepResult{Step: "System update", Error: err.Error()}
	}
	return StepResult{Step: "System update", Output: out}
}

func stepMigrations() StepResult {
	cmd := exec.Command("omarchy-migrate")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return StepResult{Step: "Run migrations", Output: stdout.String(), Error: stderr.String()}
	}
	return StepResult{Step: "Run migrations", Output: stdout.String()}
}

func stepAURPkgs() StepResult {
	if err := exec.Command("pacman", "-Qem").Run(); err != nil {
		return StepResult{Step: "AUR packages", Output: "no AUR packages installed"}
	}
	helper := detectAURHelper()
	if helper == "" {
		return StepResult{Step: "AUR packages", Output: "no AUR helper available"}
	}
	cmd := exec.Command(helper, "-Sua", "--noconfirm", "--cleanafter")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return StepResult{Step: "AUR packages", Output: stdout.String(), Error: stderr.String()}
	}
	return StepResult{Step: "AUR packages", Output: stdout.String()}
}

func stepPostHook() StepResult {
	cmd := exec.Command("omarchy-hook", "post-update")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Run() // Best-effort
	return StepResult{Step: "Post-update hook", Output: stdout.String()}
}

// ChangeChannel switches the release channel via dark-helper.
func ChangeChannel(helperPath, channel string) error {
	_, err := runHelper(helperPath, "update-channel", channel)
	return err
}

func checkRebootNeeded() bool {
	// Check if running kernel modules dir is gone (kernel updated)
	uname, err := exec.Command("uname", "-r").Output()
	if err == nil {
		modDir := filepath.Join("/usr/lib/modules", strings.TrimSpace(string(uname)))
		if _, err := os.Stat(modDir); os.IsNotExist(err) {
			return true
		}
	}
	// Check for explicit reboot-required flag
	home, _ := os.UserHomeDir()
	flag := filepath.Join(home, ".local", "state", "omarchy", "reboot-required")
	if _, err := os.Stat(flag); err == nil {
		return true
	}
	return false
}

func detectAURHelper() string {
	for _, name := range []string{"paru", "yay"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	return ""
}

func runHelper(helperPath string, args ...string) (string, error) {
	full := append([]string{helperPath}, args...)
	cmd := exec.Command("pkexec", full...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return stdout.String(), fmt.Errorf("%s", msg)
		}
		return stdout.String(), err
	}
	return stdout.String(), nil
}
