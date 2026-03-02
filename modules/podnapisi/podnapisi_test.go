package podnapisi

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

var fakeLogger = &logrus.Logger{Out: io.Discard}
var fakeLoggerEntry = logrus.NewEntry(fakeLogger)

// makeZip builds an in-memory zip archive containing the given files.
func makeZip(files map[string]string) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			panic(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			panic(err)
		}
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestExtractBestSRT(t *testing.T) {
	for _, tc := range []struct {
		name        string
		files       map[string]string
		releaseName string
		wantContent string
		wantErr     error
	}{
		{
			name:        "single srt",
			files:       map[string]string{"sub.srt": "subtitle data"},
			releaseName: "anything",
			wantContent: "subtitle data",
		},
		{
			name: "picks best matching srt",
			files: map[string]string{
				"The.Matrix.1999.BluRay.srt": "good sub",
				"The.Matrix.1999.WEBRip.srt": "other sub",
			},
			releaseName: "The.Matrix.1999.BluRay",
			wantContent: "good sub",
		},
		{
			name:    "no srt in archive",
			files:   map[string]string{"readme.txt": "nothing"},
			wantErr: ErrNoSRTInArchive,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			zipData := makeZip(tc.files)
			got, err := extractBestSRT(zipData, tc.releaseName)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected err %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tc.wantContent {
				t.Fatalf("expected content %q, got %q", tc.wantContent, string(got))
			}
		})
	}
}

func TestBestSubtitle(t *testing.T) {
	subs := []*subtitle{
		{
			PublishID:      "AAA",
			CustomReleases: []string{"The.Matrix.1999.BluRay.x264"},
		},
		{
			PublishID:      "BBB",
			CustomReleases: []string{"The.Matrix.1999.WEBRip.x264"},
		},
	}

	for _, tc := range []struct {
		name    string
		subs    []*subtitle
		release string
		wantID  string
		wantErr error
	}{
		{
			name:    "picks closest release",
			subs:    subs,
			release: "The.Matrix.1999.BluRay.x264",
			wantID:  "AAA",
		},
		{
			name:    "picks other closest release",
			subs:    subs,
			release: "The.Matrix.1999.WEBRip.x264",
			wantID:  "BBB",
		},
		{
			name:    "no subtitles",
			subs:    nil,
			release: "anything",
			wantErr: polochon.ErrNoSubtitleFound,
		},
		{
			name:    "all subs missing publish_id",
			subs:    []*subtitle{{PublishID: "", CustomReleases: []string{"foo"}}},
			release: "foo",
			wantErr: polochon.ErrNoSubtitleFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := bestSubtitle(tc.subs, tc.release)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected err %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.PublishID != tc.wantID {
				t.Fatalf("expected publishID %q, got %q", tc.wantID, got.PublishID)
			}
		})
	}
}

var fakeSubs = []*subtitle{
	{
		PublishID:      "XYZ",
		Language:       "en",
		CustomReleases: []string{"The.Matrix.1999.BluRay.x264-GROUP"},
	},
}

var fakeZipData = makeZip(map[string]string{
	"The.Matrix.1999.BluRay.x264-GROUP.srt": "1\n00:00:01,000 --> 00:00:02,000\nHello.\n",
})

func TestGetSubtitle(t *testing.T) {
	// Inject fakes.
	origSearch := podnapisiSearch
	origDownload := podnapisiDownload
	defer func() {
		podnapisiSearch = origSearch
		podnapisiDownload = origDownload
	}()

	podnapisiSearch = func(_ url.Values) ([]*subtitle, error) {
		return fakeSubs, nil
	}
	podnapisiDownload = func(_, _ string) ([]byte, error) {
		return extractBestSRT(fakeZipData, "The.Matrix.1999.BluRay.x264-GROUP")
	}

	movie := polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{
		Path: "/movies/The.Matrix.1999.BluRay.x264-GROUP.mkv",
	})
	movie.Title = "The Matrix"
	movie.Year = 1999

	episode := polochon.NewShowEpisodeFromFile(polochon.ShowConfig{}, polochon.File{
		Path: "/shows/Breaking.Bad/Season01/Breaking.Bad.S01E01.mkv",
	})
	episode.ShowTitle = "Breaking Bad"
	episode.Season = 1
	episode.Episode = 1

	c := &Client{}

	for _, tc := range []struct {
		name    string
		input   any
		lang    polochon.Language
		wantErr error
	}{
		{
			name:  "movie EN",
			input: movie,
			lang:  polochon.EN,
		},
		{
			name:  "show episode EN",
			input: episode,
			lang:  polochon.EN,
		},
		{
			name:    "unsupported type",
			input:   "not a video",
			lang:    polochon.EN,
			wantErr: ErrNotAVideo,
		},
		{
			name:    "unsupported language",
			input:   movie,
			lang:    polochon.Language("de_DE"),
			wantErr: polochon.ErrNoSubtitleFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sub, err := c.GetSubtitle(tc.input, tc.lang, fakeLoggerEntry)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("expected err %v, got %v", tc.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(sub.Data) == 0 {
				t.Fatal("expected non-empty subtitle data")
			}
		})
	}
}

func TestGetSubtitleSearchError(t *testing.T) {
	origSearch := podnapisiSearch
	defer func() { podnapisiSearch = origSearch }()

	fakeErr := errors.New("network failure")
	podnapisiSearch = func(_ url.Values) ([]*subtitle, error) {
		return nil, fakeErr
	}

	movie := polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{
		Path: "/movies/The.Matrix.1999.BluRay.x264-GROUP.mkv",
	})
	movie.Title = "The Matrix"
	movie.Year = 1999

	c := &Client{}
	_, err := c.GetSubtitle(movie, polochon.EN, fakeLoggerEntry)
	if !errors.Is(err, fakeErr) {
		t.Fatalf("expected %v, got %v", fakeErr, err)
	}
}

func TestGetSubtitleNoResults(t *testing.T) {
	origSearch := podnapisiSearch
	defer func() { podnapisiSearch = origSearch }()

	podnapisiSearch = func(_ url.Values) ([]*subtitle, error) {
		return nil, nil
	}

	movie := polochon.NewMovieFromFile(polochon.MovieConfig{}, polochon.File{
		Path: "/movies/The.Matrix.1999.BluRay.x264-GROUP.mkv",
	})
	movie.Title = "The Matrix"
	movie.Year = 1999

	c := &Client{}
	_, err := c.GetSubtitle(movie, polochon.EN, fakeLoggerEntry)
	if !errors.Is(err, polochon.ErrNoSubtitleFound) {
		t.Fatalf("expected ErrNoSubtitleFound, got %v", err)
	}
}

func TestSearch429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	origClient := httpClient
	origBase := baseURL
	httpClient = srv.Client()
	baseURL = srv.URL
	defer func() {
		httpClient = origClient
		baseURL = origBase
	}()

	_, err := search(url.Values{})
	if !errors.Is(err, ErrTooManyRequests) {
		t.Fatalf("expected ErrTooManyRequests, got %v", err)
	}
}

func TestDownloadBest429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	origClient := httpClient
	origBase := baseURL
	httpClient = srv.Client()
	baseURL = srv.URL
	defer func() {
		httpClient = origClient
		baseURL = origBase
	}()

	_, err := downloadBest("some-id", "some-release")
	if !errors.Is(err, ErrTooManyRequests) {
		t.Fatalf("expected ErrTooManyRequests, got %v", err)
	}
}
