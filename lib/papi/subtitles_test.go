package papi

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestSubtitleDownloadURL(t *testing.T) {
	baseURL := "http://mock.url"
	c, err := New(baseURL)
	if err != nil {
		t.Fatalf("invalid endpoint: %q", err)
	}

	se := &polochon.ShowEpisode{ShowImdbID: "tt2357547", Season: 1, Episode: 6}
	episodeSub := &Subtitle{
		Subtitle: &polochon.Subtitle{Video: se, Lang: polochon.FR},
	}

	m := &polochon.Movie{ImdbID: "tt001"}
	movieSub := &Subtitle{
		Subtitle: &polochon.Subtitle{Video: m, Lang: polochon.FR},
	}

	for _, test := range []struct {
		sub         Downloadable
		name        string
		expectedURL string
		expectedErr error
	}{
		{
			name:        "valid movie subtitle",
			sub:         movieSub,
			expectedURL: baseURL + "/movies/tt001/subtitles/fr_FR/download",
		},
		{
			name:        "valid espisode subtitle",
			sub:         episodeSub,
			expectedURL: baseURL + "/shows/tt2357547/seasons/1/episodes/6/subtitles/fr_FR/download",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := c.DownloadURL(test.sub)
			if err != test.expectedErr {
				t.Fatalf("expected err %q, got %q", test.expectedErr, err)
			}

			if got != test.expectedURL {
				t.Fatalf("expected %q, got %q", test.expectedURL, got)
			}
		})
	}
}

func TestUpdateSubtitles(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"lang":"fr_FR", "size": 1000}`))
	}))
	defer ts.Close()

	video := &Movie{Movie: &polochon.Movie{ImdbID: "fake_id"}}
	expectedSubs := &Subtitle{Subtitle: &polochon.Subtitle{
		File:  polochon.File{Size: 1000},
		Lang:  polochon.FR,
		Video: video,
	}}

	client, err := New(ts.URL)
	if err != nil {
		t.Fatalf("expected no error doing new client, got %q", err)
	}

	sub, err := client.UpdateSubtitle(video, polochon.FR)
	if err != nil {
		t.Fatalf("Expected no error, got %+v", err)
	}

	if !reflect.DeepEqual(sub, expectedSubs) {
		t.Fatalf("expected: %+v, got %+v", expectedSubs, sub)
	}
}
