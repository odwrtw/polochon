package papi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

var seasonResponse = `
{
	"show_imdb_id": "tt0944947",
	"season": 6,
	"episodes": [
		4
	]
}
`

func TestGetSeason(t *testing.T) {
	var requestURI string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestURI = r.RequestURI
		fmt.Fprintf(w, "%s", seasonResponse)
	}))
	defer ts.Close()

	expected := &Season{
		ShowImdbID: "tt0944947",
		Season:     6,
		Episodes: map[int]*Episode{
			4: {
				ShowEpisode: &polochon.ShowEpisode{
					ShowImdbID: "tt0944947",
					Season:     6,
					Episode:    4,
				},
			},
		},
	}

	c, err := New(ts.URL)
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}

	got, err := c.GetSeason("tt0944947", 6)
	if err != nil {
		t.Fatalf("failed to get season: %q", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("expected: %+v, got %+v", expected, got)
	}

	expectedRequestURI := "/shows/tt0944947/seasons/6"
	if requestURI != expectedRequestURI {
		t.Fatalf("expected URL %q, got %q", expectedRequestURI, requestURI)
	}
}

func TestSeasonURI(t *testing.T) {
	for _, test := range []struct {
		Season        *Season
		ExpectedURI   string
		ExpectedError error
	}{
		{
			Season:        &Season{ShowImdbID: "tt2357547", Season: 1},
			ExpectedURI:   "shows/tt2357547/seasons/1",
			ExpectedError: nil,
		},
		{
			Season:        &Season{Season: 1},
			ExpectedURI:   "",
			ExpectedError: ErrMissingShowImdbID,
		},
		{
			Season:        &Season{ShowImdbID: "tt2357547"},
			ExpectedURI:   "",
			ExpectedError: ErrMissingSeason,
		},
	} {
		gotURI, err := test.Season.uri()
		if err != test.ExpectedError {
			t.Fatalf("expected error %q, got %q", test.ExpectedError, err)
		}

		if gotURI != test.ExpectedURI {
			t.Errorf("expected %q, got %q", test.ExpectedURI, gotURI)
		}
	}
}
