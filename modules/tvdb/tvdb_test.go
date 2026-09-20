package tvdb

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestGetEpisodeDetails(t *testing.T) {
	var loginRequests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path != "/login" && r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("unexpected authorization header: %q", r.Header.Get("Authorization"))
		}

		switch r.URL.Path {
		case "/login":
			loginRequests++
			var request struct {
				APIKey string `json:"apikey"`
				Pin    string `json:"pin"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Errorf("decoding login request: %v", err)
			}
			if request.APIKey != "api-key" || request.Pin != "subscriber-pin" {
				t.Errorf("unexpected login request: %+v", request)
			}
			_, _ = w.Write([]byte(`{"status":"success","data":{"token":"token"}}`))
		case "/search":
			if got := r.URL.Query().Get("query"); got != "PONIES" {
				t.Errorf("got query %q, expected PONIES", got)
			}
			if got := r.URL.Query().Get("type"); got != "series" {
				t.Errorf("got type %q, expected series", got)
			}
			if got := r.URL.Query().Get("year"); got != "2026" {
				t.Errorf("got year %q, expected 2026", got)
			}
			_, _ = w.Write([]byte(`{
				"status":"success",
				"data":[{"name":"PONIES","tvdb_id":"453786","year":"2026"}]
			}`))
		case "/series/453786/extended":
			_, _ = w.Write([]byte(`{
				"status":"success",
				"data":{
					"id":453786,
					"name":"Ponies",
					"overview":"Two embassy secretaries become CIA operatives.",
					"firstAired":"2026-01-15",
					"averageRuntime":48,
					"remoteIds":[{"id":"tt33270420","sourceName":"IMDB"}],
					"artworks":[
						{"type":1,"image":"https://artworks.example/banner.jpg","score":10},
						{"type":2,"image":"https://artworks.example/poster.jpg","score":10},
						{"type":3,"image":"https://artworks.example/background.jpg","score":10}
					]
				}
			}`))
		case "/series/453786/episodes/default":
			if got := r.URL.Query().Get("season"); got != "1" {
				t.Errorf("got season %q, expected 1", got)
			}
			if got := r.URL.Query().Get("episodeNumber"); got != "3" {
				t.Errorf("got episode number %q, expected 3", got)
			}
			_, _ = w.Write([]byte(`{
				"status":"success",
				"data":{"episodes":[{
					"id":112233,
					"name":"Episode Three",
					"seasonNumber":1,
					"number":3,
					"aired":"2026-01-15",
					"overview":"Episode plot.",
					"image":"https://artworks.example/episode.jpg",
					"remoteIds":[{"id":"tt12345678","sourceName":"IMDB"}]
				}]},
				"links":{"next":null}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newAPIClient("api-key", "subscriber-pin")
	client.baseURL = server.URL
	client.http = server.Client()
	module := &TvDB{client: client, configured: true}
	show := polochon.NewShow(polochon.ShowConfig{})
	show.Title = "PONIES"
	show.Year = 2026
	episode := polochon.NewShowEpisode(polochon.ShowConfig{})
	episode.Show = show
	episode.ShowTitle = show.Title
	episode.Season = 1
	episode.Episode = 3

	if err := module.GetDetails(context.Background(), episode); err != nil {
		t.Fatalf("GetDetails() error = %v", err)
	}

	if loginRequests != 1 {
		t.Errorf("got %d login requests, expected 1", loginRequests)
	}
	if episode.Title != "Episode Three" || episode.TvdbID != 112233 {
		t.Errorf("unexpected episode details: %+v", episode)
	}
	if episode.ShowImdbID != "tt33270420" || episode.EpisodeImdbID != "tt12345678" {
		t.Errorf("unexpected IMDb IDs: show=%q episode=%q", episode.ShowImdbID, episode.EpisodeImdbID)
	}
	if episode.Runtime != 48 {
		t.Errorf("got runtime %d, expected 48", episode.Runtime)
	}
	if show.TvdbID != 453786 || show.Year != 2026 {
		t.Errorf("unexpected show details: %+v", show)
	}
	if show.BannerURL == "" || show.PosterURL == "" || show.FanartURL == "" {
		t.Errorf("show artwork URLs were not populated: %+v", show)
	}
}

func TestBestShowMatch(t *testing.T) {
	shows := []series{
		{ID: 1, Name: "Ponies", Year: "2012"},
		{ID: 2, Name: "Ponies", Year: "2026"},
		{ID: 3, Name: "The Ponies", Year: "2025", Aliases: []alias{{Name: "Ponies"}}},
	}

	for _, tc := range []struct {
		name  string
		query string
		year  int
		want  int
	}{
		{name: "matching title and year", query: "PONIES", year: 2026, want: 2},
		{name: "matching title without year", query: "PONIES", want: 1},
		{name: "matching alias and year", query: "Ponies", year: 2025, want: 3},
		{name: "unknown year falls back to title", query: "Ponies", year: 2030, want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := bestShowMatch(shows, tc.query, tc.year)
			if got == nil {
				t.Fatal("expected a match")
			}
			if got.ID != tc.want {
				t.Errorf("got series %d, expected %d", got.ID, tc.want)
			}
		})
	}
}

func TestLoginWithoutSubscriberPIN(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decoding login request: %v", err)
		}
		if _, ok := request["pin"]; ok {
			t.Error("project-key login unexpectedly included a subscriber PIN")
		}
		_, _ = w.Write([]byte(`{"status":"success","data":{"token":"token"}}`))
	}))
	defer server.Close()

	client := newAPIClient("project-api-key", "")
	client.baseURL = server.URL
	client.http = server.Client()
	if err := client.login(context.Background()); err != nil {
		t.Fatalf("login() error = %v", err)
	}
}

func TestInitWithParamsRequiresAPIKey(t *testing.T) {
	if err := (&TvDB{}).InitWithParams(&Params{}); err != ErrMissingAPIKey {
		t.Errorf("got %v, expected %v", err, ErrMissingAPIKey)
	}
}
