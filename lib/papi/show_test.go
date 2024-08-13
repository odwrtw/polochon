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
		"fanart_file": {
		  "name": "fanart.jpg",
		  "size": 111
		},
		"banner_file": {
		  "name": "banner.jpg",
		  "size": 222
		},
		"poster_file": {
		  "name": "poster.jpg",
		  "size": 333
		},
		"nfo_file": {
		  "name": "tvshow.nfo",
		  "size": 444
		},
		"seasons": {
			"01": {
				"01": {
					"quality": "720p",
					"release_group": "R1",
					"audio_codec": "AAC",
					"video_codec": "H.264",
					"container": "mkv",
					"filename": "Chefs.table.S01E01.mkv",
					"size": 546325850,
					"nfo_file": {
					  "name": "Chefs.table.S01E01.nfo",
					  "size": 555
					},
					"subtitles": [
					  {
						"size": 35037,
						"lang": "fr_FR"
					  },
					  {
						"size": 39349,
						"lang": "en_US"
					  }
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

	expectedShowEpisode := &polochon.ShowEpisode{
		BaseVideo: polochon.BaseVideo{
			File: polochon.File{
				Size: 546325850,
				Path: "Chefs.table.S01E01.mkv",
			},
			VideoMetadata: polochon.VideoMetadata{
				Quality:      polochon.Quality720p,
				ReleaseGroup: "R1",
				AudioCodec:   "AAC",
				VideoCodec:   "H.264",
				Container:    "mkv",
			},
		},
		ShowImdbID: "tt4295140",
		Season:     1,
		Episode:    1,
	}

	expected := &ShowCollection{
		shows: map[string]*Show{
			"tt4295140": {
				Show: &polochon.Show{
					ImdbID: "tt4295140",
					Title:  "Chef's table",
				},
				Fanart: &File{File: &index.File{Name: "fanart.jpg", Size: 111}},
				Banner: &File{File: &index.File{Name: "banner.jpg", Size: 222}},
				Poster: &File{File: &index.File{Name: "poster.jpg", Size: 333}},
				NFO:    &File{File: &index.File{Name: "tvshow.nfo", Size: 444}},
				Seasons: map[int]*Season{
					1: {
						ShowImdbID: "tt4295140",
						Season:     1,
						Episodes: map[int]*Episode{
							1: {
								ShowEpisode: expectedShowEpisode,
								NFO: &File{
									File: &index.File{
										Name: "Chefs.table.S01E01.nfo",
										Size: 555,
									},
								},
								Subtitles: []*Subtitle{
									{Subtitle: &polochon.Subtitle{
										Lang:  polochon.FR,
										File:  polochon.File{Size: 35037},
										Video: expectedShowEpisode,
									}},
									{Subtitle: &polochon.Subtitle{
										Lang:  polochon.EN,
										File:  polochon.File{Size: 39349},
										Video: expectedShowEpisode,
									}},
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

	// Files hacks
	show := expected.shows["tt4295140"]
	show.Banner.resource = show
	show.Poster.resource = show
	show.Fanart.resource = show
	show.NFO.resource = show
	episode := show.Seasons[1].Episodes[1]
	episode.NFO.resource = episode

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
						BaseVideo: polochon.BaseVideo{
							VideoMetadata: polochon.VideoMetadata{
								Quality:      polochon.Quality720p,
								ReleaseGroup: "R1",
								AudioCodec:   "AAC",
								VideoCodec:   "H.264",
								Container:    "mkv",
							},
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
		BaseVideo: polochon.BaseVideo{
			VideoMetadata: polochon.VideoMetadata{
				Quality:      polochon.Quality720p,
				ReleaseGroup: "R1",
				AudioCodec:   "AAC",
				VideoCodec:   "H.264",
				Container:    "mkv",
			},
		},
	}}

	episode := expected.GetEpisode(6, 1)
	if !reflect.DeepEqual(episode, expectedEpisode) {
		t.Fatalf("expected episode %+v, got %+v", expectedEpisode, episode)
	}
}
