package users

import (
	"testing"
)

func TestParseShadowData(t *testing.T) {
	data := []byte(`# comment line
root:$6$abc$hash:19000:0:99999:7:::
alice:$6$xyz$hash:19100:1:90:14:30:20000:
bob:!:19050::99999:::::
carol:*::0:99999:7:::
dave::19200:0:99999:7:::
eve:!!::0:99999:7:::
malformed:line:with:too:few:fields

`)
	m := parseShadowData(data)

	tests := []struct {
		user        string
		wantLocked  bool
		wantHasPass bool
		wantLastChg int64
		wantMax     int
		wantMin     int
		wantWarn    int
		wantInact   int
		wantExp     int64
	}{
		{"root", false, true, 19000, 99999, 0, 7, 0, -1},
		{"alice", false, true, 19100, 90, 1, 14, 30, 20000},
		{"bob", true, false, 19050, 99999, 0, 0, 0, -1},
		{"carol", true, false, 0, 99999, 0, 7, 0, -1},
		{"dave", false, false, 19200, 99999, 0, 7, 0, -1},
		{"eve", true, false, 0, 99999, 0, 7, 0, -1},
	}

	if _, ok := m["malformed"]; ok {
		t.Error("malformed line should have been skipped")
	}
	if _, ok := m["# comment line"]; ok {
		t.Error("comment line should have been skipped")
	}
	if len(m) != len(tests) {
		t.Errorf("parseShadowData returned %d entries, want %d", len(m), len(tests))
	}

	for _, tt := range tests {
		e, ok := m[tt.user]
		if !ok {
			t.Errorf("%s: missing from map", tt.user)
			continue
		}
		if e.locked != tt.wantLocked {
			t.Errorf("%s: locked = %v, want %v", tt.user, e.locked, tt.wantLocked)
		}
		if e.hasPass != tt.wantHasPass {
			t.Errorf("%s: hasPass = %v, want %v", tt.user, e.hasPass, tt.wantHasPass)
		}
		if e.lastChg != tt.wantLastChg {
			t.Errorf("%s: lastChg = %d, want %d", tt.user, e.lastChg, tt.wantLastChg)
		}
		if e.maxDays != tt.wantMax {
			t.Errorf("%s: maxDays = %d, want %d", tt.user, e.maxDays, tt.wantMax)
		}
		if e.minDays != tt.wantMin {
			t.Errorf("%s: minDays = %d, want %d", tt.user, e.minDays, tt.wantMin)
		}
		if e.warnDays != tt.wantWarn {
			t.Errorf("%s: warnDays = %d, want %d", tt.user, e.warnDays, tt.wantWarn)
		}
		if e.inactive != tt.wantInact {
			t.Errorf("%s: inactive = %d, want %d", tt.user, e.inactive, tt.wantInact)
		}
		if e.expires != tt.wantExp {
			t.Errorf("%s: expires = %d, want %d", tt.user, e.expires, tt.wantExp)
		}
	}
}

func TestParseShadowDataGarbage(t *testing.T) {
	// Numeric fields with garbage should be skipped gracefully, leaving
	// zero values — the parser must not panic or reject the whole entry.
	data := []byte("alice:$6$x$h:notanumber:bad:also-bad:x:y:z:\n")
	m := parseShadowData(data)
	e, ok := m["alice"]
	if !ok {
		t.Fatal("alice missing — parser rejected entry with garbage numbers")
	}
	if e.lastChg != 0 || e.minDays != 0 || e.maxDays != 0 || e.warnDays != 0 {
		t.Errorf("garbage numeric fields should produce zero values, got %+v", e)
	}
}

func TestParseGECOS(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Alice Smith,,,", "Alice Smith"},
		{"Bob", "Bob"},
		{"", ""},
		{",,,", ""},
		{"Dave,room 42,555-1234,", "Dave"},
	}
	for _, tt := range tests {
		if got := parseGECOS(tt.in); got != tt.want {
			t.Errorf("parseGECOS(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
