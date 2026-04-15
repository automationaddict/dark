package core

import "github.com/automationaddict/dark/internal/services/appstore"

// AppstoreFocus identifies which region of the App Store view owns the
// keyboard. The search bar, sidebar, results list, and detail panel each
// take focus in turn as the user navigates.
type AppstoreFocus string

const (
	AppstoreFocusSidebar AppstoreFocus = "sidebar"
	AppstoreFocusResults AppstoreFocus = "results"
	AppstoreFocusSearch  AppstoreFocus = "search"
	AppstoreFocusDetail  AppstoreFocus = "detail"
)

// SetAppstore ingests the periodic catalog snapshot from darkd. The
// sidebar category selection is clamped so a shrinking category list
// doesn't leave an out-of-bounds cursor. Busy/error state is cleared on
// every successful snapshot because any in-flight command has by then
// either returned its own snapshot (success) or already populated
// AppstoreStatusMsg (failure) via the command response path.
func (s *State) SetAppstore(snap appstore.Snapshot) {
	s.Appstore = snap
	s.AppstoreLoaded = true
	if s.AppstoreCategoryIdx >= len(snap.Categories) {
		s.AppstoreCategoryIdx = 0
	}
	if s.AppstoreFocus == "" {
		s.AppstoreFocus = AppstoreFocusSidebar
	}
}

// SetAppstoreResults stores the latest search result so the view can
// render it. The result index is reset to 0 on every new query because
// preserving selection across queries would land on a row from the
// previous set which is rarely what the user wants.
func (s *State) SetAppstoreResults(res appstore.SearchResult) {
	s.AppstoreResults = res
	s.AppstoreResultsLoaded = true
	s.AppstoreResultIdx = 0
	s.AppstoreBusy = false
	if res.AURLimit.Active {
		s.AppstoreStatusMsg = res.AURLimit.Message
	} else {
		s.AppstoreStatusMsg = ""
	}
}

// ApplyAppstoreAction patches the cached search results and open
// detail view after an install or remove completes. The daemon's
// post-action Snapshot is intentionally light (sidebar counts only)
// and does not carry the full catalog, so without this the list the
// user is looking at would still render freshly-removed packages as
// Installed until they run a new search. installed is the target
// state to apply to every Package whose name is in names.
func (s *State) ApplyAppstoreAction(names []string, installed bool) {
	if len(names) == 0 {
		return
	}
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	for i := range s.AppstoreResults.Packages {
		if _, ok := set[s.AppstoreResults.Packages[i].Name]; ok {
			s.AppstoreResults.Packages[i].Installed = installed
		}
	}
	for i := range s.Appstore.Featured {
		if _, ok := set[s.Appstore.Featured[i].Name]; ok {
			s.Appstore.Featured[i].Installed = installed
		}
	}
	if _, ok := set[s.AppstoreDetail.Name]; ok {
		s.AppstoreDetail.Installed = installed
	}
}

// SetAppstoreDetail loads the full detail panel for one package and
// shifts focus into it. Called from the tea.Cmd that answers a Detail
// request.
func (s *State) SetAppstoreDetail(detail appstore.Detail) {
	s.AppstoreDetail = detail
	s.AppstoreDetailLoaded = true
	s.AppstoreDetailOpen = true
	s.AppstoreDetailScroll = 0
	s.AppstoreFocus = AppstoreFocusDetail
	s.AppstoreBusy = false
}

// ScrollAppstoreDetail moves the detail panel viewport by delta lines.
// Clamped to [0, totalLines - viewportHeight] by the render layer
// which sets AppstoreDetailLines after each render.
func (s *State) ScrollAppstoreDetail(delta int) {
	s.AppstoreDetailScroll += delta
	if s.AppstoreDetailScroll < 0 {
		s.AppstoreDetailScroll = 0
	}
	max := s.AppstoreDetailLines - s.AppstoreDetailViewH
	if max < 0 {
		max = 0
	}
	if s.AppstoreDetailScroll > max {
		s.AppstoreDetailScroll = max
	}
}

// SetAppstoreError records a user-facing error message from a failed
// search or detail request. Busy state clears so the UI stops showing
// the spinner and the message renders in the status line.
func (s *State) SetAppstoreError(msg string) {
	s.AppstoreStatusMsg = msg
	s.AppstoreBusy = false
}

// MarkAppstoreBusy toggles the spinner that renders while a search or
// detail request is in flight.
func (s *State) MarkAppstoreBusy() {
	s.AppstoreBusy = true
	s.AppstoreStatusMsg = ""
}

// MoveF2Sidebar walks the F2 sidebar selection. Appstore categories
// are at indices 0..len(cats)-1, Updates is at len(cats). Disabled
// appstore categories are skipped.
func (s *State) MoveF2Sidebar(delta int) {
	cats := s.Appstore.Categories
	total := len(cats) + 1 // categories + Updates
	if total <= 1 {
		return
	}
	idx := s.F2SidebarIdx
	for step := 0; step < total; step++ {
		idx = (idx + delta + total) % total
		if idx == len(cats) {
			// Updates entry is always enabled
			s.F2SidebarIdx = idx
			return
		}
		if idx < len(cats) && cats[idx].Enabled {
			s.F2SidebarIdx = idx
			s.AppstoreCategoryIdx = idx
			return
		}
	}
}

// F2OnUpdates returns true when the Updates entry is selected.
func (s *State) F2OnUpdates() bool {
	return s.F2SidebarIdx == len(s.Appstore.Categories)
}

// SelectedAppstoreCategory returns the currently highlighted category.
// The bool is false when the catalog hasn't loaded or the sidebar is
// empty.
func (s *State) SelectedAppstoreCategory() (appstore.Category, bool) {
	cats := s.Appstore.Categories
	if len(cats) == 0 {
		return appstore.Category{}, false
	}
	if s.AppstoreCategoryIdx >= len(cats) {
		s.AppstoreCategoryIdx = 0
	}
	return cats[s.AppstoreCategoryIdx], true
}

// MoveAppstoreResult walks the result list highlight, wrapping at the
// ends. No-op on an empty list.
func (s *State) MoveAppstoreResult(delta int) {
	n := len(s.AppstoreResults.Packages)
	if n == 0 {
		return
	}
	s.AppstoreResultIdx = (s.AppstoreResultIdx + delta + n) % n
}

// SelectedAppstorePackage returns the highlighted result row. The bool
// is false when the result list is empty.
func (s *State) SelectedAppstorePackage() (appstore.Package, bool) {
	pkgs := s.AppstoreResults.Packages
	if len(pkgs) == 0 {
		return appstore.Package{}, false
	}
	if s.AppstoreResultIdx >= len(pkgs) {
		s.AppstoreResultIdx = 0
	}
	return pkgs[s.AppstoreResultIdx], true
}

// FocusAppstoreResults shifts focus from the sidebar into the results
// pane. No-op when there are no results yet — the user has nothing to
// navigate to.
func (s *State) FocusAppstoreResults() {
	if len(s.AppstoreResults.Packages) == 0 {
		return
	}
	s.AppstoreFocus = AppstoreFocusResults
}

// FocusAppstoreSidebar moves focus back to the categories list. Called
// on Esc from the results pane.
func (s *State) FocusAppstoreSidebar() {
	s.AppstoreFocus = AppstoreFocusSidebar
	s.AppstoreDetailOpen = false
}

// OpenAppstoreSearch puts the app store into search-input mode. The
// existing query stays prefilled so users can refine without retyping.
func (s *State) OpenAppstoreSearch() {
	s.AppstoreFocus = AppstoreFocusSearch
	s.AppstoreSearchActive = true
}

// CloseAppstoreSearch exits search-input mode without clearing the
// query. Used by Esc while the search bar is focused.
func (s *State) CloseAppstoreSearch() {
	s.AppstoreSearchActive = false
	if len(s.AppstoreResults.Packages) > 0 {
		s.AppstoreFocus = AppstoreFocusResults
	} else {
		s.AppstoreFocus = AppstoreFocusSidebar
	}
}

// AppendAppstoreSearchRune grows the search input by one character.
// Only valid while search-input mode is active.
func (s *State) AppendAppstoreSearchRune(r rune) {
	if !s.AppstoreSearchActive {
		return
	}
	s.AppstoreSearchInput += string(r)
}

// BackspaceAppstoreSearch drops the trailing character from the search
// input. No-op on an empty query.
func (s *State) BackspaceAppstoreSearch() {
	if !s.AppstoreSearchActive || s.AppstoreSearchInput == "" {
		return
	}
	r := []rune(s.AppstoreSearchInput)
	s.AppstoreSearchInput = string(r[:len(r)-1])
}

// CloseAppstoreDetail hides the detail panel and returns focus to the
// results list.
func (s *State) CloseAppstoreDetail() {
	s.AppstoreDetailOpen = false
	s.AppstoreFocus = AppstoreFocusResults
}
