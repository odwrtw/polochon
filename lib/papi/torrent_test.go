package papi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

var torrentResponse = `
{
	"message": "Torrent added"
}`
var torrentError = `
{
	"error": "Invalid torrent"
}`
var noSuchTorrentError = `
{
	"error": "No such torrent"
}`

var torrentList = `
[
  {
	"imdb_id": "tt001",
	"type": "movie",
	"season": 0,
	"episode": 0,
	"quality": "1080p",
	"status": {
		"id": "129734284as8",
		"ratio": -1,
		"is_finished": false,
		"download_rate": 30000,
		"upload_rate": 0,
		"total_size": 797027726,
		"downloaded_size": 34435662,
		"percent_done": 4.3,
		"file_paths": [
		  "Awesome movie.mp4",
		  "Awesome movie.srt"
		],
		"name": "Awesome"
	}
  }
]`

func TestAddTorrent(t *testing.T) {
	for _, test := range []struct {
		serverHeader  int
		expectedError error
		polochonError string
	}{
		{
			expectedError: nil,
			serverHeader:  http.StatusOK,
			polochonError: torrentResponse,
		},
		{
			serverHeader:  http.StatusForbidden,
			expectedError: fmt.Errorf("papi: HTTP error status 403 Forbidden: Invalid torrent"),
			polochonError: torrentError,
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.serverHeader)
			fmt.Fprintf(w, "%s", test.polochonError)
		}))
		defer ts.Close()

		c, err := New(ts.URL)
		if err != nil {
			t.Fatalf("invalid endpoint: %q", err)
		}

		torrent := &polochon.Torrent{
			Result: &polochon.TorrentResult{
				URL: "AABBCCDD",
			},
		}
		err = c.AddTorrent(torrent)
		if err != nil {
			if err.Error() != test.expectedError.Error() {
				t.Fatalf("expected %q but got %q", test.expectedError, err)
			}
		} else if test.expectedError != nil {
			t.Fatalf("expected %q but got nil", test.expectedError)
		}
	}
}

func TestGetTorrents(t *testing.T) {
	for _, test := range []struct {
		serverHeader     int
		expectedError    error
		polochonError    string
		expectedTorrents []*polochon.Torrent
	}{
		{
			expectedError: nil,
			serverHeader:  http.StatusOK,
			polochonError: torrentList,
			expectedTorrents: []*polochon.Torrent{
				&polochon.Torrent{
					ImdbID:  "tt001",
					Type:    polochon.TypeMovie,
					Quality: polochon.Quality1080p,
					Status: &polochon.TorrentStatus{
						ID:             "129734284as8",
						Ratio:          -1,
						IsFinished:     false,
						UploadRate:     0,
						DownloadRate:   30000,
						TotalSize:      797027726,
						DownloadedSize: 34435662,
						PercentDone:    4.3,
						FilePaths: []string{
							"Awesome movie.mp4",
							"Awesome movie.srt",
						},
						Name: "Awesome",
					},
				},
			},
		},
		{
			serverHeader:  http.StatusForbidden,
			expectedError: fmt.Errorf("papi: HTTP error status 403 Forbidden: Invalid torrent"),
			polochonError: torrentError,
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.serverHeader)
			fmt.Fprintf(w, "%s", test.polochonError)
		}))
		defer ts.Close()

		c, err := New(ts.URL)
		if err != nil {
			t.Fatalf("invalid endpoint: %q", err)
		}

		torrents, err := c.GetTorrents()
		if err != nil {
			if test.expectedError == nil {
				t.Fatalf("expected no errors but got %q", err)
			}
			if err.Error() != test.expectedError.Error() {
				t.Fatalf("expected %q but got %q", test.expectedError, err)
			}
		} else if test.expectedError != nil {
			t.Fatalf("expected %q but got nil", test.expectedError)
		}

		if !reflect.DeepEqual(torrents, test.expectedTorrents) {
			t.Fatalf("expected: \n%+v , got \n%+v", torrents, test.expectedTorrents)
		}
	}
}

func TestRemoveTorrents(t *testing.T) {
	for _, test := range []struct {
		serverHeader  int
		expectedError error
		polochonError string
	}{
		{
			expectedError: nil,
			serverHeader:  http.StatusOK,
			polochonError: "",
		},
		{
			serverHeader:  http.StatusNotFound,
			expectedError: fmt.Errorf("papi: resource not found"),
			polochonError: noSuchTorrentError,
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(test.serverHeader)
			fmt.Fprintf(w, "%s", test.polochonError)
		}))
		defer ts.Close()

		c, err := New(ts.URL)
		if err != nil {
			t.Fatalf("invalid endpoint: %q", err)
		}

		err = c.RemoveTorrent("29r472398")
		if err != nil {
			if test.expectedError == nil {
				t.Fatalf("expected no errors but got %q", err)
			}
			if err.Error() != test.expectedError.Error() {
				t.Fatalf("expected %q but got %q", test.expectedError, err)
			}
		} else if test.expectedError != nil {
			t.Fatalf("expected %q but got nil", test.expectedError)
		}
	}
}
