package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/johnnelson/dark/internal/services/appstore"
)

// AppstoreActions is the set of asynchronous commands the App Store
// view can dispatch at darkd. Each returns a tea.Cmd that issues a
// NATS request and posts the result back into the program. Mirrors
// WifiActions / BluetoothActions in shape so the view layer stays
// consistent with the rest of the TUI.
type AppstoreActions struct {
	Search  func(query appstore.SearchQuery) tea.Cmd
	Detail  func(req appstore.DetailRequest) tea.Cmd
	Refresh func() tea.Cmd
}

// AppstoreMsg is dispatched whenever darkd publishes a catalog
// snapshot. The subscriber in cmd/dark unmarshals the NATS payload and
// forwards it as this typed value.
type AppstoreMsg appstore.Snapshot

// AppstoreSearchResultMsg is the reply from a search command. Err is
// non-empty when the request failed — either the NATS round-trip or
// the daemon-side error field.
type AppstoreSearchResultMsg struct {
	Result appstore.SearchResult
	Err    string
}

// AppstoreDetailResultMsg is the reply from a detail command.
type AppstoreDetailResultMsg struct {
	Detail appstore.Detail
	Err    string
}

// AppstoreRefreshResultMsg is the reply from a refresh command. The
// daemon answers with the post-refresh snapshot so the client can
// update immediately without waiting for the next periodic publish.
type AppstoreRefreshResultMsg struct {
	Snapshot appstore.Snapshot
	Err      string
}
