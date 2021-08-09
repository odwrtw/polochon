package papi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

var showByIDsResponse = `
{
	"tt4295140": {
		"title": "Chef's table",
		"seasons": {
			"01": {
				"01": {
					"quality": "720p",
					"release_group": "R1",
					"audio_codec": "AAC",
					"video_codec": "H.264",
					"container": "mkv",
					"subtitles": [
					  "fr_FR",
					  "en_US"
					]
				}
			}
		}
	},
	"tt4428122": {
		"title": "Quantico",
		"seasons": {
			"01": {
				"01":{},
				"02":{}
			}
		}
	}
}
`

func TestGetShows(t *testing.T) {
	var requestURI string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.RequestURI
		fmt.Fprintf(w, "%s", showByIDsResponse)
	}))
	defer ts.Close()

	expected := &ShowCollection{
		shows: map[string]*Show{
			"tt4295140": {
				Show: &polochon.Show{
					ImdbID: "tt4295140",
					Title:  "Chef's table",
				},
				Seasons: map[int]*Season{
					1: {
						ShowImdbID: "tt4295140",
						Season:     1,
						Episodes: map[int]*Episode{
							1: {
								ShowEpisode: &polochon.ShowEpisode{
									VideoMetadata: polochon.VideoMetadata{
										Quality:      polochon.Quality720p,
										ReleaseGroup: "R1",
										AudioCodec:   "AAC",
										VideoCodec:   "H.264",
										Container:    "mkv",
									},
									ShowImdbID: "tt4295140",
									Season:     1,
									Episode:    1,
								},
								Subtitles: []*index.Subtitle{
									{
										Lang: polochon.FR,
									},
									{
										Lang: polochon.EN,
									},
								},
							},
						},
					},
				},
			},
			"tt4428122": {
				Show: &polochon.Show{
					ImdbID: "tt4428122",
					Title:  "Quantico",
				},
				Seasons: map[int]*Season{
					1: {
						ShowImdbID: "tt4428122",
						Season:     1,
						Episodes: map[int]*Episode{
							1: {ShowEpisode: &polochon.ShowEpisode{
								ShowImdbID: "tt4428122",
								Season:     1,
								Episode:    1,
							}},
							2: {ShowEpisode: &polochon.ShowEpisode{
								ShowImdbID: "tt4428122",
								Season:     1,
								Episode:    2,
							}},
						},
					},
				},
			},
		},
	}

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}

	// Sort the response to be deteministic
	got, err := c.GetShows()
	if err != nil {
		t.Fatalf("failed to get shows by IDs: %q", err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected: %+v, got %+v", expected, got)
	}

	expectedRequestURI := "/shows"
	if requestURI != expectedRequestURI {
		t.Fatalf("expected URL %q, got %q", expectedRequestURI, requestURI)
	}
}

func TestShowURI(t *testing.T) {
	for _, test := range []struct {
		Show          *Show
		ExpectedURI   string
		ExpectedError error
	}{
		{
			Show:          &Show{Show: &polochon.Show{ImdbID: "tt2357547"}},
			ExpectedURI:   "shows/tt2357547",
			ExpectedError: nil,
		},
		{
			Show:          &Show{},
			ExpectedURI:   "",
			ExpectedError: ErrMissingShow,
		},
		{
			Show:          &Show{Show: &polochon.Show{}},
			ExpectedURI:   "",
			ExpectedError: ErrMissingShowImdbID,
		},
	} {
		gotURI, err := test.Show.uri()
		if err != test.ExpectedError {
			t.Fatalf("expected error %q, got %q", test.ExpectedError, err)
		}

		if gotURI != test.ExpectedURI {
			t.Errorf("expected %q, got %q", test.ExpectedURI, gotURI)
		}
	}
}

var showResponse = `
{
	"title": "Game of Thrones",
	"rating": 9.5,
	"plot": "Awesome plot",
	"tvdb_id": 121361,
	"imdb_id": "tt0944947",
	"year": 2011,
	"first_aired": "2011-04-17T00:00:00Z",
	"seasons": {
		"06": {
			"01":{
				"quality": "720p",
				"release_group": "R1",
				"audio_codec": "AAC",
				"video_codec": "H.264",
				"container": "mkv"
			}
		}
	}
}
`

func TestGetShow(t *testing.T) {
	var requestURI string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.RequestURI
		fmt.Fprintf(w, "%s", showResponse)
	}))
	defer ts.Close()

	date := time.Date(2011, time.April, 17, 0, 0, 0, 0, time.UTC)
	expected := &Show{
		Show: &polochon.Show{
			ImdbID:     "tt0944947",
			Title:      "Game of Thrones",
			Plot:       "Awesome plot",
			TvdbID:     121361,
			Year:       2011,
			Rating:     9.5,
			FirstAired: &date,
		},
		Seasons: map[int]*Season{
			6: {
				ShowImdbID: "tt0944947",
				Season:     6,
				Episodes: map[int]*Episode{
					1: {ShowEpisode: &polochon.ShowEpisode{
						ShowImdbID: "tt0944947",
						Season:     6,
						Episode:    1,
						VideoMetadata: polochon.VideoMetadata{
							Quality:      polochon.Quality720p,
							ReleaseGroup: "R1",
							AudioCodec:   "AAC",
							VideoCodec:   "H.264",
							Container:    "mkv",
						},
					}},
				},
			},
		},
	}

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}

	got, err := c.GetShow("tt0944947")
	if err != nil {
		t.Fatalf("failed to get show: %q", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected: %+v, got %+v", expected, got)
	}

	expectedRequestURI := "/shows/tt0944947"
	if requestURI != expectedRequestURI {
		t.Fatalf("expected URL %q, got %q", expectedRequestURI, requestURI)
	}

	if !expected.HasEpisode(6, 1) {
		t.Fatal("the show should have the episode S06E01")
	}

	if expected.HasEpisode(1, 1) {
		t.Fatal("the show shouldn't have the episode S01E01")
	}

	expectedEpisode := &Episode{ShowEpisode: &polochon.ShowEpisode{
		ShowImdbID: "tt0944947",
		Season:     6,
		Episode:    1,
		VideoMetadata: polochon.VideoMetadata{
			Quality:      polochon.Quality720p,
			ReleaseGroup: "R1",
			AudioCodec:   "AAC",
			VideoCodec:   "H.264",
			Container:    "mkv",
		},
	}}

	episode := expected.GetEpisode(6, 1)
	if !reflect.DeepEqual(episode, expectedEpisode) {
		t.Fatalf("expected episode %+v, got %+v", expectedEpisode, episode)
	}
}
