package papi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestShowEpisode(t *testing.T) {
	for _, test := range []struct {
		ShowEpisode         *Episode
		ExpectedURI         string
		ExpectedDownloadURL string
		ExpectedError       error
	}{
		{
			ShowEpisode: &Episode{ShowEpisode: &polochon.ShowEpisode{
				ShowImdbID: "tt2357547", Season: 1, Episode: 6,
			}},
			ExpectedURI:         "shows/tt2357547/seasons/1/episodes/6",
			ExpectedDownloadURL: "shows/tt2357547/seasons/1/episodes/6/download",
			ExpectedError:       nil,
		},
		{
			ShowEpisode: &Episode{ShowEpisode: &polochon.ShowEpisode{
				Season: 1, Episode: 6,
			}},
			ExpectedURI:         "",
			ExpectedDownloadURL: "",
			ExpectedError:       ErrMissingShowEpisodeInformations,
		},
		{
			ShowEpisode: &Episode{ShowEpisode: &polochon.ShowEpisode{
				ShowImdbID: "tt2357547",
			}},
			ExpectedURI:         "",
			ExpectedDownloadURL: "",
			ExpectedError:       ErrMissingShowEpisodeInformations,
		},
	} {
		gotURI, err := test.ShowEpisode.uri()
		if err != test.ExpectedError {
			t.Fatalf("expected error %q, got %q", test.ExpectedError, err)
		}

		if gotURI != test.ExpectedURI {
			t.Errorf("expected %q, got %q", test.ExpectedURI, gotURI)
		}

		gotDownloadURL, err := test.ShowEpisode.downloadURL()
		if err != test.ExpectedError {
			t.Fatalf("expected error %q, got %q", test.ExpectedError, err)
		}

		if gotDownloadURL != test.ExpectedDownloadURL {
			t.Errorf("expected %q, got %q", test.ExpectedDownloadURL, gotDownloadURL)
		}
	}
}

var episodeResponse = `
{
	"title": "Book of the Stranger",
	"season": 6,
	"episode": 4,
	"tvdb_id": 5599364,
	"aired": "2016-05-15",
	"plot": "Awesome plot",
	"runtime": 55,
	"thumb": "http://thetvdb.com/banners/episodes/121361/5599364.jpg",
	"rating": 8.3,
	"show_imdb_id": "tt0944947",
	"show_tvdb_id": 121361,
	"imdb_id": "tt4283016",
	"release_group": "",
	"torrents": null,
	"subtitles": null
}
`

func TestGetEpisode(t *testing.T) {
	var requestURI string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.RequestURI
		fmt.Fprintf(w, "%s", episodeResponse)
	}))
	defer ts.Close()

	expected := &Episode{ShowEpisode: &polochon.ShowEpisode{
		ShowImdbID:    "tt0944947",
		ShowTvdbID:    121361,
		EpisodeImdbID: "tt4283016",
		Season:        6,
		Episode:       4,
		TvdbID:        5599364,
		Plot:          "Awesome plot",
		Title:         "Book of the Stranger",
		Runtime:       55,
		Thumb:         "http://thetvdb.com/banners/episodes/121361/5599364.jpg",
		Rating:        8.3,
		Aired:         "2016-05-15",
	}}

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}

	got, err := c.GetEpisode("tt0944947", 6, 4)
	if err != nil {
		t.Fatalf("failed to get season: %q", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected: \n%+v\n, got \n%+v", expected, got)
	}

	expectedRequestURI := "/shows/tt0944947/seasons/6/episodes/4"
	if requestURI != expectedRequestURI {
		t.Fatalf("expected URL %q, got %q", expectedRequestURI, requestURI)
	}
}

func TestShowEpisodeSubtitleURL(t *testing.T) {
	for _, test := range []struct {
		ShowEpisode         *Episode
		ExpectedSubtitleURL string
		lang                polochon.Language
		ExpectedError       error
	}{
		{
			ShowEpisode: &Episode{ShowEpisode: &polochon.ShowEpisode{
				ShowImdbID: "tt2357547", Season: 1, Episode: 6,
			}},
			lang:                polochon.FR,
			ExpectedSubtitleURL: "shows/tt2357547/seasons/1/episodes/6/subtitles/fr_FR/download",
			ExpectedError:       nil,
		},
		{
			ShowEpisode: &Episode{ShowEpisode: &polochon.ShowEpisode{
				Season: 1, Episode: 6,
			}},
			lang:                polochon.FR,
			ExpectedSubtitleURL: "",
			ExpectedError:       ErrMissingShowEpisodeInformations,
		},
		{
			ShowEpisode: &Episode{ShowEpisode: &polochon.ShowEpisode{
				ShowImdbID: "tt2357547",
			}},
			lang:                polochon.FR,
			ExpectedSubtitleURL: "",
			ExpectedError:       ErrMissingShowEpisodeInformations,
		},
	} {
		gotSubtitleURL, err := test.ShowEpisode.subtitleURL(test.lang)
		if err != test.ExpectedError {
			t.Fatalf("expected error %q, got %q", test.ExpectedError, err)
		}

		if gotSubtitleURL != test.ExpectedSubtitleURL {
			t.Errorf("expected %q, got %q", test.ExpectedSubtitleURL, gotSubtitleURL)
		}
	}
}
