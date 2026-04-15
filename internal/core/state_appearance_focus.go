package core

// AppearanceThemeFocus values: which box owns the keyboard within
// the Appearance → Theme content region when
// AppearanceContentFocused is true.
const (
	AppearanceFocusTheme       = "theme"
	AppearanceFocusBackgrounds = "backgrounds"
)

// CycleAppearanceThemeFocus flips the focus ring between the two
// stops on the Theme sub-section (Theme box ↔ Backgrounds box).
// delta is accepted for signature parity with other cycle
// helpers but is unused today because there are only two stops —
// tab and shift-tab both toggle.
//
// The Backgrounds stop is skipped when the snapshot has no
// backgrounds to pick from so focus never lands on a dead box.
func (s *State) CycleAppearanceThemeFocus(delta int) {
	_ = delta
	if len(s.Appearance.Backgrounds) == 0 {
		s.AppearanceThemeFocus = AppearanceFocusTheme
		return
	}
	if s.AppearanceThemeFocus == AppearanceFocusBackgrounds {
		s.AppearanceThemeFocus = AppearanceFocusTheme
	} else {
		s.AppearanceThemeFocus = AppearanceFocusBackgrounds
	}
}

// MoveAppearanceBackground shifts the background selection by
// delta, wrapping at the list boundaries. No-op on an empty list.
func (s *State) MoveAppearanceBackground(delta int) {
	n := len(s.Appearance.Backgrounds)
	if n == 0 {
		s.AppearanceBackgroundIdx = 0
		return
	}
	s.AppearanceBackgroundIdx = (s.AppearanceBackgroundIdx + delta + n) % n
}

// SelectedAppearanceBackground returns the highlighted filename
// and a bool that's false when the list is empty.
func (s *State) SelectedAppearanceBackground() (string, bool) {
	n := len(s.Appearance.Backgrounds)
	if n == 0 {
		return "", false
	}
	if s.AppearanceBackgroundIdx >= n {
		s.AppearanceBackgroundIdx = 0
	}
	return s.Appearance.Backgrounds[s.AppearanceBackgroundIdx], true
}

// InitAppearanceThemeFocus sets a sensible starting focus when
// the user enters the Theme content region. Starts on the Theme
// box so the user can spot which box is active before anything
// changes. Also syncs the backgrounds index to the currently-
// active background so enter doesn't flip the wallpaper to an
// unrelated one just because the cursor was on row zero.
func (s *State) InitAppearanceThemeFocus() {
	s.AppearanceThemeFocus = AppearanceFocusTheme
	if s.Appearance.CurrentBackground != "" {
		for i, name := range s.Appearance.Backgrounds {
			if name == s.Appearance.CurrentBackground {
				s.AppearanceBackgroundIdx = i
				return
			}
		}
	}
}
