package appstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	aurBaseURL     = "https://aur.archlinux.org/rpc/v5"
	aurHTTPTimeout = 15 * time.Second
	aurMaxInflight = 2
	aurUserAgent   = "dark-appstore/0.1 (+https://github.com/johnnelson/dark)"
)

// aurClient is the HTTP client wrapper enforcing the ≤2 inflight-request
// policy and emitting typed errors the backend can classify as
// rate-limit vs transient.
type aurClient struct {
	logger *slog.Logger
	http   *http.Client
	sem    chan struct{}
}

func newAURClient(logger *slog.Logger) *aurClient {
	return &aurClient{
		logger: logger,
		http: &http.Client{
			Timeout: aurHTTPTimeout,
		},
		sem: make(chan struct{}, aurMaxInflight),
	}
}

// aurRPCResponse is the envelope aurweb returns on every endpoint.
// Error is non-empty when aurweb reports a problem at the app layer
// (HTTP 200 with error text) — we surface that as a typed error.
type aurRPCResponse struct {
	Version     int       `json:"version"`
	Type        string    `json:"type"`
	ResultCount int       `json:"resultcount"`
	Results     []aurRow  `json:"results"`
	Error       string    `json:"error,omitempty"`
}

// aurRow is one result row. The search endpoint returns a subset of
// these fields; info returns the full shape. We decode both into the
// same struct and rely on zero-values for missing fields.
type aurRow struct {
	ID             int      `json:"ID"`
	Name           string   `json:"Name"`
	PackageBaseID  int      `json:"PackageBaseID"`
	PackageBase    string   `json:"PackageBase"`
	Version        string   `json:"Version"`
	Description    string   `json:"Description"`
	URL            string   `json:"URL"`
	NumVotes       int      `json:"NumVotes"`
	Popularity     float64  `json:"Popularity"`
	OutOfDate      *int64   `json:"OutOfDate"`
	Maintainer     string   `json:"Maintainer"`
	Submitter      string   `json:"Submitter"`
	FirstSubmitted int64    `json:"FirstSubmitted"`
	LastModified   int64    `json:"LastModified"`
	URLPath        string   `json:"URLPath"`
	Depends        []string `json:"Depends"`
	MakeDepends    []string `json:"MakeDepends"`
	CheckDepends   []string `json:"CheckDepends"`
	OptDepends     []string `json:"OptDepends"`
	Conflicts      []string `json:"Conflicts"`
	Provides       []string `json:"Provides"`
	Replaces       []string `json:"Replaces"`
	Groups         []string `json:"Groups"`
	License        []string `json:"License"`
	Keywords       []string `json:"Keywords"`
}

func (r aurRow) toPackage() Package {
	return Package{
		Name:            r.Name,
		Version:         r.Version,
		Description:     r.Description,
		Origin:          OriginAUR,
		Votes:           r.NumVotes,
		Popularity:      r.Popularity,
		LastUpdatedUnix: r.LastModified,
		// InstalledSize intentionally stays zero — the AUR doesn't
		// report size and faking a number would mislead the user.
		// The UI renders zero as "—".
	}
}

func (r aurRow) toDetail() Detail {
	return Detail{
		Package:       r.toPackage(),
		URL:           r.URL,
		Licenses:      r.License,
		Maintainer:    r.Maintainer,
		Groups:        r.Groups,
		Provides:      r.Provides,
		Depends:       r.Depends,
		OptDepends:    r.OptDepends,
		MakeDepends:   r.MakeDepends,
		CheckDepends:  r.CheckDepends,
		Conflicts:     r.Conflicts,
		Replaces:      r.Replaces,
		BuildDateUnix: r.LastModified,
		LongDesc:      r.Description,
	}
}

// aurThrottleError is the sentinel the backend looks for to decide
// whether to enter backoff. retryAfter is zero when the server didn't
// provide a Retry-After header, in which case the backend uses its
// exponential schedule.
type aurThrottleError struct {
	status     int
	retryAfter time.Duration
	body       string
}

func (e *aurThrottleError) Error() string {
	if e.retryAfter > 0 {
		return fmt.Sprintf("AUR throttled: HTTP %d, retry after %s", e.status, e.retryAfter)
	}
	return fmt.Sprintf("AUR throttled: HTTP %d", e.status)
}

// classifyAURError returns the retry-after window and a bool indicating
// whether the error is a rate-limit signal. Called by the backend to
// decide whether to enter the backoff window.
func classifyAURError(err error) (time.Duration, bool) {
	var te *aurThrottleError
	if errors.As(err, &te) {
		return te.retryAfter, true
	}
	return 0, false
}

// search calls /rpc/v5/search/{term}?by=name-desc and returns the
// decoded rows. The caller is responsible for caching and rate-limit
// bookkeeping.
func (c *aurClient) search(term string) ([]aurRow, error) {
	endpoint := fmt.Sprintf("%s/search/%s?by=name-desc", aurBaseURL, url.PathEscape(term))
	var env aurRPCResponse
	if err := c.do(endpoint, &env); err != nil {
		return nil, err
	}
	if env.Error != "" {
		return nil, fmt.Errorf("AUR search %q: %s", term, env.Error)
	}
	return env.Results, nil
}

// info calls /rpc/v5/info?arg[]=... in chunks so no single request
// exceeds aurMaxInfoArgs. Results are returned in backend order with
// the chunk order preserved. The caller should not depend on input
// order because aurweb may return rows in any order within a chunk.
func (c *aurClient) info(names []string) ([]aurRow, error) {
	if len(names) == 0 {
		return nil, nil
	}
	out := make([]aurRow, 0, len(names))
	for start := 0; start < len(names); start += aurMaxInfoArgs {
		end := start + aurMaxInfoArgs
		if end > len(names) {
			end = len(names)
		}
		chunk := names[start:end]
		q := url.Values{}
		for _, name := range chunk {
			q.Add("arg[]", name)
		}
		endpoint := aurBaseURL + "/info?" + q.Encode()
		var env aurRPCResponse
		if err := c.do(endpoint, &env); err != nil {
			return nil, err
		}
		if env.Error != "" {
			return nil, fmt.Errorf("AUR info: %s", env.Error)
		}
		out = append(out, env.Results...)
	}
	return out, nil
}

// do acquires an inflight slot, issues the GET, decodes the body into
// v, and maps transport / HTTP errors into typed errors. The semaphore
// bounds in-flight requests across the whole process.
func (c *aurClient) do(endpoint string, v any) error {
	c.sem <- struct{}{}
	defer func() { <-c.sem }()

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("build AUR request: %w", err)
	}
	req.Header.Set("User-Agent", aurUserAgent)
	req.Header.Set("Accept", "application/json")

	started := time.Now()
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("AUR GET %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooManyRequests, http.StatusServiceUnavailable:
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		retry := parseRetryAfter(resp.Header.Get("Retry-After"))
		return &aurThrottleError{
			status:     resp.StatusCode,
			retryAfter: retry,
			body:       strings.TrimSpace(string(body)),
		}
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("AUR HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decode AUR response: %w", err)
	}
	c.logger.Debug("request completed", "endpoint", endpoint, "status", resp.StatusCode, "elapsed", time.Since(started))
	return nil
}

// parseRetryAfter handles both the seconds-form ("120") and the
// HTTP-date form of the Retry-After header. Unparseable values return
// zero and the caller falls back to exponential backoff.
func parseRetryAfter(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	if secs, err := strconv.Atoi(s); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(s); err == nil {
		d := time.Until(t)
		if d > 0 {
			return d
		}
	}
	return 0
}

// removeDirContents wipes every file under dir without removing dir
// itself. Used by Refresh to clear the AUR cache. Missing dir is not
// an error.
var removeDirContentsMu sync.Mutex

func removeDirContents(dir string) error {
	removeDirContentsMu.Lock()
	defer removeDirContentsMu.Unlock()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}
