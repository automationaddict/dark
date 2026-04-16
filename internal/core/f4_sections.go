package core

// F4Section is one top-level entry in the F4 outer sidebar. Today
// the list is populated with a single entry (SSH). Future power-
// user features add peers without touching rendering code.
type F4Section struct {
	ID    string
	Icon  string
	Label string
}

// F4Sections returns the ordered list of F4 top-level items.
// Matching style: same `{id, icon, label}` shape F1/F3 use.
func F4Sections() []F4Section {
	return []F4Section{
		{"ssh", "󰣀", "SSH"},
	}
}
