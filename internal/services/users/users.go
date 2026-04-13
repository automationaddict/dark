package users

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type User struct {
	Username  string   `json:"username"`
	UID       int      `json:"uid"`
	GID       int      `json:"gid"`
	FullName  string   `json:"full_name"`
	HomeDir   string   `json:"home_dir"`
	Shell     string   `json:"shell"`
	Groups    []string `json:"groups"`
	Primary   string   `json:"primary_group"`
	IsAdmin     bool `json:"is_admin"`
	IsLocked    bool `json:"is_locked"`
	HasPasswd   bool `json:"has_passwd"`
	ShadowAvail bool `json:"shadow_avail"`

	// Password aging (from /etc/shadow via helper).
	LastChanged  string `json:"last_changed,omitempty"`
	PasswordAge  string `json:"password_age,omitempty"`
	MaxDays      int    `json:"max_days,omitempty"`
	MinDays      int    `json:"min_days,omitempty"`
	WarnDays     int    `json:"warn_days,omitempty"`
	InactiveDays int    `json:"inactive_days,omitempty"`
	Expires      string `json:"expires,omitempty"`

	// Session info.
	Sessions    []Session `json:"sessions,omitempty"`
	LastLogins  []Login   `json:"last_logins,omitempty"`
	LoginCount  int       `json:"login_count"`
	ProcessCount int      `json:"process_count"`
}

type Session struct {
	ID     string `json:"id"`
	TTY    string `json:"tty"`
	State  string `json:"state"`
	Remote string `json:"remote"`
	Since  string `json:"since"`
}

type Login struct {
	TTY      string `json:"tty"`
	Host     string `json:"host"`
	Time     string `json:"time"`
	Duration string `json:"duration"`
}

type Snapshot struct {
	Users    []User   `json:"users"`
	Shells   []string `json:"shells"`
	Hostname string   `json:"hostname"`
}

func ReadSnapshot() Snapshot {
	s := Snapshot{}
	s.Users = readUsers()
	s.Shells = readShells()
	s.Hostname, _ = os.Hostname()

	// Enrich with shadow info (best-effort, may require root).
	shadowMap := readShadow()
	for i := range s.Users {
		enrichShadow(&s.Users[i], shadowMap)
		s.Users[i].Sessions = readSessions(s.Users[i].Username)
		s.Users[i].LastLogins = readLastLogins(s.Users[i].Username)
		s.Users[i].ProcessCount = countProcesses(s.Users[i].UID)
	}

	return s
}

func readUsers() []User {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil
	}
	defer f.Close()

	groupMap := readGroupMap()
	memberMap := readMemberMap()

	var users []User
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 7)
		if len(parts) < 7 {
			continue
		}

		uid, _ := strconv.Atoi(parts[2])
		gid, _ := strconv.Atoi(parts[3])

		// Only show human users (UID >= 1000) and root.
		if uid < 1000 && uid != 0 {
			continue
		}
		// Skip nobody.
		if uid == 65534 {
			continue
		}

		u := User{
			Username: parts[0],
			UID:      uid,
			GID:      gid,
			FullName: parseGECOS(parts[4]),
			HomeDir:  parts[5],
			Shell:    parts[6],
			Primary:  groupMap[gid],
			Groups:   memberMap[parts[0]],
		}

		// Check admin status (member of wheel).
		for _, g := range u.Groups {
			if g == "wheel" {
				u.IsAdmin = true
				break
			}
		}

		users = append(users, u)
	}

	sort.Slice(users, func(i, j int) bool {
		if users[i].UID == 0 {
			return false
		}
		if users[j].UID == 0 {
			return false
		}
		return users[i].Username < users[j].Username
	})

	return users
}

func parseGECOS(gecos string) string {
	// GECOS field is comma-separated; first field is full name.
	if idx := strings.Index(gecos, ","); idx >= 0 {
		return gecos[:idx]
	}
	return gecos
}

// readGroupMap returns gid -> group name.
func readGroupMap() map[int]string {
	f, err := os.Open("/etc/group")
	if err != nil {
		return nil
	}
	defer f.Close()

	m := make(map[int]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 4)
		if len(parts) < 4 {
			continue
		}
		gid, _ := strconv.Atoi(parts[2])
		m[gid] = parts[0]
	}
	return m
}

// readMemberMap returns username -> list of groups.
func readMemberMap() map[string][]string {
	f, err := os.Open("/etc/group")
	if err != nil {
		return nil
	}
	defer f.Close()

	m := make(map[string][]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), ":", 4)
		if len(parts) < 4 || parts[3] == "" {
			continue
		}
		groupName := parts[0]
		for _, user := range strings.Split(parts[3], ",") {
			user = strings.TrimSpace(user)
			if user != "" {
				m[user] = append(m[user], groupName)
			}
		}
	}

	// Sort group lists.
	for k := range m {
		sort.Strings(m[k])
	}
	return m
}

type shadowEntry struct {
	locked   bool
	hasPass  bool
	lastChg  int64 // days since epoch
	maxDays  int
	minDays  int
	warnDays int
	inactive int
	expires  int64 // days since epoch, -1 = never
}

func readShadow() map[string]shadowEntry {
	data, err := os.ReadFile("/etc/shadow")
	if err != nil {
		// Try via helper (needs root).
		data, err = readShadowViaHelper()
		if err != nil {
			return nil
		}
	}
	return parseShadowData(data)
}

func parseShadowData(data []byte) map[string]shadowEntry {
	m := make(map[string]shadowEntry)
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 9)
		if len(parts) < 9 {
			continue
		}

		e := shadowEntry{expires: -1}
		pass := parts[1]
		e.locked = strings.HasPrefix(pass, "!") || strings.HasPrefix(pass, "*")
		e.hasPass = pass != "" && pass != "!" && pass != "*" && pass != "!!" && pass != "!*"
		e.lastChg, _ = strconv.ParseInt(parts[2], 10, 64)
		e.minDays, _ = strconv.Atoi(parts[3])
		e.maxDays, _ = strconv.Atoi(parts[4])
		e.warnDays, _ = strconv.Atoi(parts[5])
		if parts[6] != "" {
			e.inactive, _ = strconv.Atoi(parts[6])
		}
		if parts[7] != "" {
			e.expires, _ = strconv.ParseInt(parts[7], 10, 64)
		}

		m[parts[0]] = e
	}
	return m
}

func readShadowViaHelper() ([]byte, error) {
	helper, err := helperPath()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(helper, "read-shadow")
	return cmd.Output()
}

func enrichShadow(u *User, shadow map[string]shadowEntry) {
	if shadow == nil {
		return
	}
	e, ok := shadow[u.Username]
	if !ok {
		return
	}
	u.ShadowAvail = true
	u.IsLocked = e.locked
	u.HasPasswd = e.hasPass

	epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
	if e.lastChg > 0 {
		changed := epoch.AddDate(0, 0, int(e.lastChg))
		u.LastChanged = changed.Format("2006-01-02")
		age := time.Since(changed)
		u.PasswordAge = fmt.Sprintf("%d days", int(age.Hours()/24))
	}
	u.MaxDays = e.maxDays
	u.MinDays = e.minDays
	u.WarnDays = e.warnDays
	u.InactiveDays = e.inactive
	if e.expires > 0 {
		u.Expires = epoch.AddDate(0, 0, int(e.expires)).Format("2006-01-02")
	} else {
		u.Expires = "never"
	}
}

func readSessions(username string) []Session {
	out, err := exec.Command("loginctl", "list-sessions", "--no-legend", "--no-pager").Output()
	if err != nil {
		return nil
	}
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Fields: SESSION UID USER [SEAT] [TTY]
		if fields[2] != username {
			continue
		}
		s := Session{ID: fields[0]}
		// Get session details.
		detail, err := exec.Command("loginctl", "show-session", fields[0],
			"--property=TTY,State,Remote,RemoteHost,Timestamp").Output()
		if err == nil {
			for _, dline := range strings.Split(string(detail), "\n") {
				k, v, ok := strings.Cut(dline, "=")
				if !ok {
					continue
				}
				switch k {
				case "TTY":
					s.TTY = v
				case "State":
					s.State = v
				case "Remote":
					s.Remote = v
				case "RemoteHost":
					if v != "" {
						s.Remote = v
					}
				case "Timestamp":
					if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", v); err == nil {
						s.Since = t.Format("2006-01-02 15:04")
					} else {
						s.Since = v
					}
				}
			}
		}
		sessions = append(sessions, s)
	}
	return sessions
}

func readLastLogins(username string) []Login {
	out, err := exec.Command("last", "-n", "5", "-F", username).Output()
	if err != nil {
		return nil
	}
	var logins []Login
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" || strings.HasPrefix(line, "wtmp") || strings.HasPrefix(line, "btmp") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 || fields[0] != username {
			continue
		}
		l := Login{TTY: fields[1]}
		// Parse the rest — format varies (with/without host).
		rest := strings.TrimPrefix(line, fields[0])
		rest = strings.TrimPrefix(strings.TrimSpace(rest), fields[1])
		rest = strings.TrimSpace(rest)

		// Check for host in the line.
		if len(fields) > 4 && !strings.HasPrefix(fields[2], "Mon") &&
			!strings.HasPrefix(fields[2], "Tue") &&
			!strings.HasPrefix(fields[2], "Wed") &&
			!strings.HasPrefix(fields[2], "Thu") &&
			!strings.HasPrefix(fields[2], "Fri") &&
			!strings.HasPrefix(fields[2], "Sat") &&
			!strings.HasPrefix(fields[2], "Sun") {
			l.Host = fields[2]
		}

		// Extract timestamp — look for the day-of-week pattern.
		if idx := findDayOfWeek(rest); idx >= 0 {
			timeStr := rest[idx:]
			// Try to find the duration at the end.
			if dashIdx := strings.Index(timeStr, " - "); dashIdx > 0 {
				l.Time = strings.TrimSpace(timeStr[:dashIdx])
				durStr := strings.TrimSpace(timeStr[dashIdx+3:])
				if paren := strings.Index(durStr, "("); paren >= 0 {
					l.Duration = strings.Trim(durStr[paren:], "()")
				} else {
					l.Duration = durStr
				}
			} else {
				l.Time = strings.TrimSpace(timeStr)
				if strings.Contains(l.Time, "still logged in") {
					l.Duration = "active"
					l.Time = strings.TrimSuffix(l.Time, "   still logged in")
					l.Time = strings.TrimSpace(l.Time)
				}
			}
		}

		logins = append(logins, l)
	}
	return logins
}

func findDayOfWeek(s string) int {
	days := []string{"Mon ", "Tue ", "Wed ", "Thu ", "Fri ", "Sat ", "Sun "}
	for _, d := range days {
		if idx := strings.Index(s, d); idx >= 0 {
			return idx
		}
	}
	return -1
}

func countProcesses(uid int) int {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return 0
	}
	count := 0
	uidStr := fmt.Sprintf("Uid:\t%d\t", uid)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Only look at numeric directory names (PIDs).
		if _, err := strconv.Atoi(e.Name()); err != nil {
			continue
		}
		data, err := os.ReadFile("/proc/" + e.Name() + "/status")
		if err != nil {
			continue
		}
		if strings.Contains(string(data), uidStr) {
			count++
		}
	}
	return count
}

func readShells() []string {
	f, err := os.Open("/etc/shells")
	if err != nil {
		return []string{"/bin/bash", "/bin/zsh", "/bin/fish", "/usr/bin/nologin"}
	}
	defer f.Close()

	var shells []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		shells = append(shells, line)
	}
	return shells
}
