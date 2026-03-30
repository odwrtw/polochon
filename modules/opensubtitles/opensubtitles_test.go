package opensubtitles

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	polochon "github.com/odwrtw/polochon/lib"
)

var silentLog = logrus.NewEntry(&logrus.Logger{Out: io.Discard})

func TestName(t *testing.T) {
	o := &opensubs{}
	if got := o.Name(); got != moduleName {
		t.Errorf("Name() = %q, want %q", got, moduleName)
	}
}

func TestInitWithParams(t *testing.T) {
	for _, tc := range []struct {
		name        string
		params      Params
		wantKey     string
		wantAPIBase string
	}{
		{"sets api key", Params{APIKey: "abc123"}, "abc123", apiBase},
		{"empty key", Params{}, "", apiBase},
		{"custom api base", Params{APIKey: "k", APIBase: "https://mock.example.com"}, "k", "https://mock.example.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &opensubs{}
			if err := o.InitWithParams(&tc.params); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if o.apiKey != tc.wantKey {
				t.Errorf("got apiKey %q, want %q", o.apiKey, tc.wantKey)
			}
			if o.apiBase != tc.wantAPIBase {
				t.Errorf("got apiBase %q, want %q", o.apiBase, tc.wantAPIBase)
			}
		})
	}
}

func TestStatus(t *testing.T) {
	for _, tc := range []struct {
		name       string
		apiKey     string
		httpStatus int
		wantOK     bool
	}{
		{"ok", "abc123", http.StatusOK, true},
		{"missing key", "", 0, false},
		{"unauthorized", "badkey", http.StatusUnauthorized, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &opensubs{apiKey: tc.apiKey, apiBase: apiBase}

			if tc.apiKey != "" {
				orig := doRequest
				defer func() { doRequest = orig }()
				doRequest = func(_ *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: tc.httpStatus,
						Body:       io.NopCloser(bytes.NewReader(nil)),
					}, nil
				}
			}

			status, err := o.Status()
			if tc.wantOK {
				if err != nil || status != polochon.StatusOK {
					t.Errorf("expected StatusOK/nil, got %v/%v", status, err)
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				}
			}
		})
	}
}

func TestHashFile(t *testing.T) {
	f, err := os.CreateTemp("", "hashtest_*.dat")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()

	// Write 200KB of patterned data so first and last chunks are distinct
	data := make([]byte, 200*1024)
	for i := range data {
		data[i] = byte(i)
	}
	if _, err := f.Write(data); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	hash, err := hashFile(f.Name())
	if err != nil {
		t.Fatalf("hashFile error: %v", err)
	}
	if len(hash) != 16 {
		t.Errorf("expected 16-char hex hash, got %q (len %d)", hash, len(hash))
	}
	// Must be deterministic
	hash2, err := hashFile(f.Name())
	if err != nil || hash != hash2 {
		t.Errorf("hash not deterministic: %q vs %q", hash, hash2)
	}
}

func TestHashFile_SmallFile(t *testing.T) {
	f, err := os.CreateTemp("", "hashtest_small_*.dat")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Remove(f.Name()) }()
	_ = f.Close()

	_, err = hashFile(f.Name())
	if err == nil {
		t.Error("expected error for file smaller than 64KB, got nil")
	}
}

func TestDownloadSubtitle_FileFetchError(t *testing.T) {
	o := &opensubs{apiKey: "key", username: "user", password: "pass"}
	movie := &polochon.Movie{}
	movie.Path = "/media/movie.mkv"
	entry := &polochon.SubtitleEntry{ID: "42", Language: polochon.EN}

	orig := doRequest
	defer func() { doRequest = orig }()

	calls := 0
	doRequest = func(_ *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1: // POST /login
			return loginResp(), nil
		case 2: // POST /download
			return downloadResp("https://example.com/subtitle.srt"), nil
		case 3: // GET temp URL → error
			return &http.Response{
				StatusCode: http.StatusGone,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		default: // DELETE /logout (best-effort, ignored)
			return logoutResp(), nil
		}
	}

	_, err := o.DownloadSubtitle(movie, entry, silentLog)
	if err == nil {
		t.Error("expected error for non-200 file fetch, got nil")
	}
}

// searchResp builds a fake GET /subtitles response with one result.
func searchResp(release string, fileID int) *http.Response {
	body, _ := json.Marshal(map[string]any{
		"data": []map[string]any{
			{
				"attributes": map[string]any{
					"language": "en",
					"release":  release,
					"ratings":  8.0,
					"files":    []map[string]any{{"file_id": fileID}},
				},
			},
		},
	})
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func emptySearchResp() *http.Response {
	body, _ := json.Marshal(map[string]any{"data": []any{}})
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func TestListSubtitles_IMDBFallback(t *testing.T) {
	o := &opensubs{apiKey: "key"}
	// No Path → hash search skipped; IMDB ID present → IMDB search used
	movie := &polochon.Movie{}
	movie.ImdbID = "tt0133093"
	movie.Title = "The Matrix"

	orig := doRequest
	defer func() { doRequest = orig }()
	calls := 0
	doRequest = func(_ *http.Request) (*http.Response, error) {
		calls++
		return searchResp("The.Matrix.1999.BluRay", 99), nil
	}

	entries, err := o.ListSubtitles(movie, polochon.EN, silentLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected entries")
	}
	if entries[0].Language != polochon.EN {
		t.Errorf("language not set: got %v, want %v", entries[0].Language, polochon.EN)
	}
	if calls != 1 {
		t.Errorf("expected 1 HTTP call (IMDB tier), got %d", calls)
	}
}

func TestListSubtitles_TitleFallback(t *testing.T) {
	o := &opensubs{apiKey: "key"}
	// No Path, no IMDB ID → title search used
	movie := &polochon.Movie{}
	movie.Title = "Unknown Film"

	orig := doRequest
	defer func() { doRequest = orig }()
	doRequest = func(_ *http.Request) (*http.Response, error) {
		return searchResp("Unknown.Film.2020", 7), nil
	}

	entries, err := o.ListSubtitles(movie, polochon.EN, silentLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries[0].ID != "7" {
		t.Errorf("got id %q, want %q", entries[0].ID, "7")
	}
}

func TestListSubtitles_NoResults(t *testing.T) {
	o := &opensubs{apiKey: "key"}
	movie := &polochon.Movie{}
	movie.Title = "Totally Unknown"

	orig := doRequest
	defer func() { doRequest = orig }()
	doRequest = func(_ *http.Request) (*http.Response, error) {
		return emptySearchResp(), nil
	}

	_, err := o.ListSubtitles(movie, polochon.EN, silentLog)
	if err != polochon.ErrNoSubtitleFound {
		t.Errorf("expected polochon.ErrNoSubtitleFound, got %v", err)
	}
}

func TestListSubtitles_TransportError(t *testing.T) {
	o := &opensubs{apiKey: "key"}
	movie := &polochon.Movie{}
	movie.Title = "Some Film"

	orig := doRequest
	defer func() { doRequest = orig }()
	doRequest = func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(bytes.NewReader(nil)),
		}, nil
	}

	_, err := o.ListSubtitles(movie, polochon.EN, silentLog)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err == polochon.ErrNoSubtitleFound {
		t.Error("transport error must not be masked as ErrNoSubtitleFound")
	}
}

func TestListSubtitles_ShowEpisode(t *testing.T) {
	o := &opensubs{apiKey: "key"}
	ep := &polochon.ShowEpisode{}
	ep.ShowImdbID = "tt0411008"
	ep.ShowTitle = "Lost"
	ep.Season = 1
	ep.Episode = 3

	orig := doRequest
	defer func() { doRequest = orig }()
	var capturedQuery string
	doRequest = func(req *http.Request) (*http.Response, error) {
		capturedQuery = req.URL.RawQuery
		return searchResp("Lost.S01E03.BluRay", 55), nil
	}

	entries, err := o.ListSubtitles(ep, polochon.FR, silentLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if entries[0].Language != polochon.FR {
		t.Errorf("expected FR language, got %v", entries[0].Language)
	}
	if !strings.Contains(capturedQuery, "season_number=1") {
		t.Errorf("season_number missing from query: %s", capturedQuery)
	}
	if !strings.Contains(capturedQuery, "episode_number=3") {
		t.Errorf("episode_number missing from query: %s", capturedQuery)
	}
}

func TestSearch(t *testing.T) {
	o := &opensubs{apiKey: "testkey"}

	orig := doRequest
	defer func() { doRequest = orig }()
	doRequest = func(req *http.Request) (*http.Response, error) {
		if req.Header.Get("Api-Key") != "testkey" {
			t.Errorf("missing Api-Key header, got %q", req.Header.Get("Api-Key"))
		}
		if req.Header.Get("User-Agent") != "polochon" {
			t.Errorf("missing User-Agent header, got %q", req.Header.Get("User-Agent"))
		}
		return searchResp("Movie.2020.BluRay", 42), nil
	}

	params := url.Values{"imdb_id": {"133093"}, "languages": {"en"}}
	entries, err := o.search(params)
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ID != "42" {
		t.Errorf("got id %q, want %q", entries[0].ID, "42")
	}
	if entries[0].Description != "Movie.2020.BluRay (Rating: 8.0)" {
		t.Errorf("got description %q, want %q", entries[0].Description, "Movie.2020.BluRay (Rating: 8.0)")
	}
	if entries[0].Source != moduleName {
		t.Errorf("got source %q, want %q", entries[0].Source, moduleName)
	}
}

func loginResp() *http.Response {
	body, _ := json.Marshal(map[string]any{"token": "test-token", "status": 200})
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func logoutResp() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(nil)),
	}
}

func downloadResp(link string) *http.Response {
	body, _ := json.Marshal(map[string]any{
		"link":           link,
		"remaining":      4,
		"reset_time_utc": "2099-01-01T00:00:00.000Z",
	})
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func downloadRespExhausted(link string) *http.Response {
	body, _ := json.Marshal(map[string]any{
		"link":           link,
		"remaining":      0,
		"reset_time_utc": "2099-01-01T00:00:00.000Z",
	})
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
}

func fileResp(content string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader([]byte(content))),
	}
}

func TestDownloadSubtitle(t *testing.T) {
	o := &opensubs{apiKey: "key", username: "user", password: "pass"}
	movie := &polochon.Movie{}
	movie.Path = "/media/The.Matrix.1999.mkv"
	entry := &polochon.SubtitleEntry{ID: "42", Language: polochon.EN}

	orig := doRequest
	defer func() { doRequest = orig }()

	calls := 0
	doRequest = func(req *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1: // POST /login
			if req.Method != http.MethodPost {
				t.Errorf("call 1: expected POST, got %s", req.Method)
			}
			return loginResp(), nil
		case 2: // POST /download
			if req.Method != http.MethodPost {
				t.Errorf("call 2: expected POST, got %s", req.Method)
			}
			if req.Header.Get("Authorization") != "Bearer test-token" {
				t.Errorf("call 2: missing Authorization header")
			}
			return downloadResp("https://example.com/subtitle.srt"), nil
		case 3: // GET temp URL
			if req.Method != http.MethodGet {
				t.Errorf("call 3: expected GET, got %s", req.Method)
			}
			return fileResp("1\n00:00:01,000 --> 00:00:02,000\nHello"), nil
		case 4: // DELETE /logout
			if req.Method != http.MethodDelete {
				t.Errorf("call 4: expected DELETE, got %s", req.Method)
			}
			return logoutResp(), nil
		default:
			t.Fatalf("unexpected request %d to %s", calls, req.URL)
			return nil, nil
		}
	}

	sub, err := o.DownloadSubtitle(movie, entry, silentLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub.Lang != polochon.EN {
		t.Errorf("got lang %v, want %v", sub.Lang, polochon.EN)
	}
	if len(sub.Data) == 0 {
		t.Error("expected subtitle data, got empty")
	}
	if sub.Video == nil {
		t.Error("expected Video to be set")
	}
	if sub.Path == "" {
		t.Error("expected Path to be set")
	}
}

func TestDownloadSubtitle_QuotaExceeded(t *testing.T) {
	o := &opensubs{apiKey: "key", username: "user", password: "pass"}
	movie := &polochon.Movie{}
	movie.Path = "/media/movie.mkv"
	entry := &polochon.SubtitleEntry{ID: "42"}

	orig := doRequest
	defer func() { doRequest = orig }()

	calls := 0
	doRequest = func(_ *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1: // POST /login
			return loginResp(), nil
		default: // POST /download → 429
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(bytes.NewReader(nil)),
			}, nil
		}
	}

	_, err := o.DownloadSubtitle(movie, entry, silentLog)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestGetSubtitle(t *testing.T) {
	o := &opensubs{apiKey: "key", username: "user", password: "pass"}
	movie := &polochon.Movie{}
	movie.ImdbID = "tt0133093"
	movie.Title = "The Matrix"

	orig := doRequest
	defer func() { doRequest = orig }()

	calls := 0
	doRequest = func(_ *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1: // GET /subtitles (IMDB tier)
			return searchResp("The.Matrix.1999.BluRay", 42), nil
		case 2: // POST /login
			return loginResp(), nil
		case 3: // POST /download
			return downloadResp("https://example.com/matrix.srt"), nil
		case 4: // GET temp URL
			return fileResp("subtitle content"), nil
		case 5: // DELETE /logout
			return logoutResp(), nil
		default:
			t.Fatalf("unexpected request %d", calls)
			return nil, nil
		}
	}

	sub, err := o.GetSubtitle(movie, polochon.EN, silentLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(sub.Data) != "subtitle content" {
		t.Errorf("got data %q, want %q", string(sub.Data), "subtitle content")
	}
}

func TestUpdateQuota(t *testing.T) {
	o := &opensubs{}

	// Valid RFC3339 timestamp updates state
	o.updateQuota(3, "2099-06-01T12:00:00.000Z")
	o.mu.Lock()
	remaining, resetAt := o.remaining, o.resetAt
	o.mu.Unlock()
	if remaining != 3 {
		t.Errorf("remaining = %d, want 3", remaining)
	}
	if resetAt.IsZero() {
		t.Error("resetAt should be set")
	}

	// Invalid timestamp is a no-op
	o.updateQuota(0, "not-a-date")
	o.mu.Lock()
	remaining = o.remaining
	o.mu.Unlock()
	if remaining != 3 {
		t.Errorf("invalid timestamp should be no-op, remaining = %d, want 3", remaining)
	}
}

func TestQuotaReached(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Minute)

	for _, tc := range []struct {
		name      string
		remaining int
		resetAt   time.Time
		want      bool
	}{
		{"zero state", 0, time.Time{}, false},
		{"remaining > 0", 2, future, false},
		{"reset in past", 0, past, false},
		{"exhausted", 0, future, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o := &opensubs{remaining: tc.remaining, resetAt: tc.resetAt}
			if got := o.quotaReached(); got != tc.want {
				t.Errorf("quotaReached() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDownloadSubtitle_QuotaReached(t *testing.T) {
	o := &opensubs{remaining: 0, resetAt: time.Now().Add(time.Hour)}
	movie := &polochon.Movie{}
	entry := &polochon.SubtitleEntry{ID: "42"}

	_, err := o.DownloadSubtitle(movie, entry, silentLog)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestListSubtitles_QuotaReached(t *testing.T) {
	o := &opensubs{remaining: 0, resetAt: time.Now().Add(time.Hour)}
	movie := &polochon.Movie{} // no doRequest mock needed — returns before any HTTP call

	_, err := o.ListSubtitles(movie, polochon.EN, silentLog)
	if err != ErrQuotaExceeded {
		t.Errorf("expected ErrQuotaExceeded, got %v", err)
	}
}

func TestDownloadSubtitle_StoresQuota(t *testing.T) {
	o := &opensubs{apiKey: "key", username: "user", password: "pass"}
	movie := &polochon.Movie{}
	movie.Path = "/media/movie.mkv"
	entry := &polochon.SubtitleEntry{ID: "42", Language: polochon.EN}

	orig := doRequest
	defer func() { doRequest = orig }()

	calls := 0
	doRequest = func(_ *http.Request) (*http.Response, error) {
		calls++
		switch calls {
		case 1:
			return loginResp(), nil
		case 2:
			return downloadRespExhausted("https://example.com/sub.srt"), nil
		case 3:
			return fileResp("content"), nil
		default:
			return logoutResp(), nil
		}
	}

	_, err := o.DownloadSubtitle(movie, entry, silentLog)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	o.mu.Lock()
	remaining, resetAt := o.remaining, o.resetAt
	o.mu.Unlock()

	if remaining != 0 {
		t.Errorf("remaining = %d, want 0", remaining)
	}
	if resetAt.IsZero() {
		t.Error("resetAt should be set after download")
	}
}
