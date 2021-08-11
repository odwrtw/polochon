package papi

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

func TestSubtitleURL(t *testing.T) {
	c, err := New("http://mock.url")
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}

	for _, test := range []struct {
		Downloadable Downloadable
		lang         polochon.Language
		ExpectedURL  string
		ExpectedErr  error
	}{
		{
			Downloadable: &Movie{Movie: &polochon.Movie{ImdbID: "tt001"}},
			lang:         polochon.FR,
			ExpectedURL:  "http://mock.url/movies/tt001/subtitles/fr_FR/download",
			ExpectedErr:  nil,
		},
	} {
		got, err := c.SubtitleURL(test.Downloadable, test.lang)
		if err != test.ExpectedErr {
			t.Fatalf("expected err %q, got %q", test.ExpectedErr, err)
		}

		if got != test.ExpectedURL {
			t.Fatalf("expected %q, got %q", test.ExpectedURL, got)
		}
	}
}

func TestUpdateSubtitles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			[{"lang":"fr_FR", "size": 1000}, {"lang":"en_US", "size": 2000}]
		`))
	}))
	defer ts.Close()

	expectedSubs := []*index.Subtitle{
		{Lang: polochon.FR, Size: 1000},
		{Lang: polochon.EN, Size: 2000},
	}

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("expected no error doing new client, got %q", err)
	}

	subs, err := client.UpdateSubtitles(&Movie{Movie: &polochon.Movie{ImdbID: "fake_id"}})
	if err != nil {
		t.Fatalf("Expected no error, got %+v", err)
	}
	if !reflect.DeepEqual(subs, expectedSubs) {
		t.Fatalf("expected: %+v, got %+v", expectedSubs, subs)
	}
}
