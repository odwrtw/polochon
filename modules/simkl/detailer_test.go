package simkl

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

func metadataModule(t *testing.T, overrides map[string]string) *Simkl {
	t.Helper()
	bodies := map[string]string{}
	for path, name := range map[string]string{
		"/movies/42": "movie.json", "/tv/84": "show.json", "/tv/episodes/84": "episodes.json",
	} {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		bodies[path] = string(data)
	}
	for path, body := range overrides {
		bodies[path] = body
	}
	return testModule(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			path := "movies/42/inception"
			if r.URL.Query().Get("type") == "tv" {
				path = "tv/84/game-of-thrones"
			}
			w.Header().Set("Location", "//simkl.com/"+path)
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
		body, ok := bodies[r.URL.Path]
		if !ok {
			t.Errorf("unexpected request: %s", r.URL)
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	})
}

func TestMovieDetails(t *testing.T) {
	s := metadataModule(t, nil)
	movie := polochon.NewMovie(polochon.MovieConfig{Detailers: []polochon.Detailer{s}})
	movie.ImdbID = "tt001"
	movie.Path = "movie.mkv"
	movie.OriginalTitle, movie.Tagline = "Known original title", "Known tagline"
	if err := s.GetDetails(context.Background(), movie); err != nil {
		t.Fatal(err)
	}
	if movie.Title != "Inception" || movie.SortTitle != movie.Title || movie.Year != 2010 || movie.TmdbID != 27205 || movie.Runtime != 148 || movie.Plot != "A dream within a dream." {
		t.Errorf("unexpected movie: %+v", movie)
	}
	if movie.Rating != 8.6 || movie.Votes != 11454 || !reflect.DeepEqual(movie.Genres, []string{"Action", "Science Fiction"}) {
		t.Errorf("unexpected movie ratings/genres: %+v", movie)
	}
	if movie.Path != "movie.mkv" || len(movie.Detailers) != 1 || movie.OriginalTitle != "Known original title" || movie.Tagline != "Known tagline" {
		t.Errorf("lost caller fields: %+v", movie)
	}
	checkImage(t, movie.ThumbURL, "posters/14/poster_m.webp")
	checkImage(t, movie.FanartURL, "fanart/27/background_medium.webp")
}

func TestShowDetails(t *testing.T) {
	s := metadataModule(t, nil)
	show := polochon.NewShow(polochon.ShowConfig{Detailers: []polochon.Detailer{s}, Calendar: s})
	show.ImdbID = "tt002"
	if err := s.GetDetails(context.Background(), show); err != nil {
		t.Fatal(err)
	}
	if show.Title != "Game of Thrones" || show.TvdbID != 121361 || show.Year != 2011 || show.Rating != 9.1 || show.URL != "https://simkl.com/tv/84/game-of-thrones" || show.FirstAired == nil || show.FirstAired.Format(time.DateOnly) != "2011-04-17" {
		t.Errorf("unexpected show: %+v", show)
	}
	checkImage(t, show.PosterURL, "posters/57/poster_m.webp")
	checkImage(t, show.BannerURL, "posters/57/poster_w.webp")
	checkImage(t, show.FanartURL, "fanart/65/background_medium.webp")
	if len(show.Episodes) != 2 {
		t.Fatalf("got %d episodes, want 2", len(show.Episodes))
	}
	first, second := show.Episodes[0], show.Episodes[1]
	if first.Title != "Winter Is Coming" || first.Aired != "2011-04-17" || first.Runtime != 62 || first.Season != 1 || first.Episode != 1 || first.ShowImdbID != show.ImdbID || first.ShowTvdbID != show.TvdbID || len(first.Detailers) != 1 {
		t.Errorf("unexpected first episode: %+v", first)
	}
	if second.Runtime != 52 || second.Plot != "" || second.ThumbURL != "" {
		t.Errorf("nullable episode fields: %+v", second)
	}
	if first.TvdbID != 0 || first.EpisodeImdbID != "" {
		t.Error("SIMKL episode ID leaked into external IDs")
	}
	checkImage(t, first.ThumbURL, "episodes/26/still_w.webp")
	known := *first
	known.TvdbID, known.EpisodeImdbID = 123, "tt-episode"
	show.Episodes = []*polochon.ShowEpisode{nil, &known}
	if err := s.GetDetails(context.Background(), show); err != nil {
		t.Fatal(err)
	}
	if show.Episodes[0].TvdbID != 123 || show.Episodes[0].EpisodeImdbID != "tt-episode" {
		t.Error("refresh discarded known external episode IDs")
	}
}

func TestEpisodeDetails(t *testing.T) {
	s := metadataModule(t, nil)
	show := &polochon.Show{Title: "Old title", ImdbID: "tt-stale"}
	e := polochon.NewShowEpisode(polochon.ShowConfig{Detailers: []polochon.Detailer{s}})
	e.Show, e.Season, e.Episode = show, 1, 1
	e.ShowImdbID = "tt002"
	e.Path, e.TvdbID, e.EpisodeImdbID = "episode.mkv", 123, "tt-episode"
	if err := s.GetDetails(context.Background(), e); err != nil {
		t.Fatal(err)
	}
	if e.Title != "Winter Is Coming" || e.ShowTitle != "Game of Thrones" || e.ShowImdbID != "tt002" || e.ShowTvdbID != 121361 || e.Aired != "2011-04-17" || e.Runtime != 62 {
		t.Errorf("unexpected episode: %+v", e)
	}
	if e.Path != "episode.mkv" || e.TvdbID != 123 || e.EpisodeImdbID != "tt-episode" || len(e.Detailers) != 1 {
		t.Errorf("lost caller fields: %+v", e)
	}
	if e.Show == show || show.Title != "Old title" || show.ImdbID != "tt-stale" || e.Show.Title != "Game of Thrones" {
		t.Errorf("unexpected parent update: old=%+v, new=%+v", show, e.Show)
	}
	// A parent pointer is optional; the episode's parent IDs suffice.
	e = &polochon.ShowEpisode{ShowImdbID: "tt002", Season: 1, Episode: 2}
	if err := s.GetDetails(context.Background(), e); err != nil || e.Title != "The Kingsroad" {
		t.Errorf("episode without parent: %+v, %v", e, err)
	}
}

func checkImage(t *testing.T, value, path string) {
	t.Helper()
	u, err := url.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if u.Scheme != "https" || u.Host != "wsrv.nl" || q.Get("url") != "https://simkl.in/"+path || q.Get("output") != "jpg" || q.Get("q") != "90" {
		t.Errorf("unexpected JPEG image URL: %s", value)
	}
}

func TestDetailsFailuresAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		name      string
		overrides map[string]string
		kind      string
	}{
		{"unknown movie", map[string]string{"/movies/42": `[]`}, "movie"},
		{"movie date", map[string]string{"/movies/42": `{"title":"New","type":"movie","released":"invalid","ids":{"simkl":42,"imdb":"tt001"}}`}, "movie"},
		{"movie artwork", map[string]string{"/movies/42": `{"title":"New","type":"movie","ids":{"simkl":42,"imdb":"tt001"}}`}, "movie"},
		{"conflicting ID", map[string]string{"/movies/42": `{"title":"Wrong","type":"movie","ids":{"simkl":42,"imdb":"tt-other"}}`}, "movie"},
		{"unknown show", map[string]string{"/tv/84": `[]`}, "show"},
		{"show artwork", map[string]string{"/tv/84": `{"title":"New","type":"show","ids":{"simkl":84,"imdb":"tt002"}}`}, "show"},
		{"bad episodes", map[string]string{"/tv/episodes/84": `{`}, "show"},
		{"episode date", map[string]string{"/tv/episodes/84": `[{"type":"episode","season":1,"episode":1,"date":"invalid"}]`}, "episode"},
		{"episode missing", map[string]string{"/tv/episodes/84": `[]`}, "episode"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := metadataModule(t, tc.overrides)
			movie := &polochon.Movie{Title: "Old", ImdbID: "tt001", Genres: []string{"Existing"}}
			show := &polochon.Show{Title: "Old", ImdbID: "tt002", Episodes: []*polochon.ShowEpisode{{Title: "Existing"}}}
			e := &polochon.ShowEpisode{Title: "Old", Show: show, Season: 1, Episode: 1}
			beforeMovie, beforeShow, beforeEpisode := *movie, *show, *e
			var target any = movie
			switch tc.kind {
			case "show":
				target = show
			case "episode":
				target = e
			}
			if err := s.GetDetails(context.Background(), target); err == nil {
				t.Fatal("expected error")
			}
			if !reflect.DeepEqual(*movie, beforeMovie) || !reflect.DeepEqual(*show, beforeShow) || !reflect.DeepEqual(*e, beforeEpisode) {
				t.Fatal("failed request mutated caller metadata")
			}
		})
	}
}

func TestDetailArguments(t *testing.T) {
	s := testModule(t, func(_ http.ResponseWriter, _ *http.Request) { t.Error("unexpected request") })
	for _, target := range []any{nil, "movie", (*polochon.Movie)(nil), (*polochon.Show)(nil), (*polochon.ShowEpisode)(nil), &polochon.Movie{}, &polochon.Show{}, &polochon.ShowEpisode{}, &polochon.ShowEpisode{Season: 1, Episode: 1}} {
		if err := s.GetDetails(context.Background(), target); err != ErrInvalidArgument {
			t.Errorf("GetDetails(%#v) = %v", target, err)
		}
	}
	if err := (&Simkl{}).GetDetails(context.Background(), &polochon.Movie{}); err != ErrMissingClientID {
		t.Errorf("unconfigured detailer = %v", err)
	}
}

func TestIdentifierPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		kind   string
		target any
		query  url.Values
	}{
		{"movie IMDb", "movies", &polochon.Movie{ImdbID: "tt001", TmdbID: 12, Title: "Title"}, url.Values{"imdb": {"tt001"}}},
		{"movie TMDB", "movies", &polochon.Movie{TmdbID: 12, Title: "Title"}, url.Values{"tmdb": {"12"}}},
		{"movie title", "movies", &polochon.Movie{Title: "Title", Year: 2026}, url.Values{"title": {"Title"}, "year": {"2026"}}},
		{"show IMDb", "tv", &polochon.Show{ImdbID: "tt002", TvdbID: 34, Title: "Title"}, url.Values{"imdb": {"tt002"}}},
		{"show TVDB", "tv", &polochon.Show{TvdbID: 34, Title: "Title"}, url.Values{"tvdb": {"34"}}},
		{"show title", "tv", &polochon.Show{Title: "Title", Year: 2026}, url.Values{"title": {"Title"}, "year": {"2026"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := testModule(t, func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				for _, key := range []string{"imdb", "tmdb", "tvdb", "title", "year"} {
					if q.Get(key) != tc.query.Get(key) {
						t.Errorf("%s = %q, want %q", key, q.Get(key), tc.query.Get(key))
					}
				}
				wantType := "movie"
				if tc.kind == "tv" {
					wantType = "tv"
				}
				if q.Get("type") != wantType {
					t.Errorf("type = %s", q.Get("type"))
				}
				w.Header().Set("Location", "//simkl.com")
				w.WriteHeader(http.StatusMovedPermanently)
			})
			if err := s.GetDetails(context.Background(), tc.target); err != ErrNotFound {
				t.Errorf("GetDetails = %v", err)
			}
		})
	}
}

func TestTitleLookupRequiresImdbIdentity(t *testing.T) {
	s := metadataModule(t, map[string]string{
		"/movies/42": `{"title":"Unidentified","type":"movie","ids":{"simkl":42},"poster":"p","fanart":"f"}`,
	})
	movie := &polochon.Movie{Title: "Unidentified"}
	before := *movie
	if err := s.GetDetails(context.Background(), movie); err != ErrNotFound || !reflect.DeepEqual(*movie, before) {
		t.Errorf("unidentified movie = %+v, %v", movie, err)
	}
}

func TestOptionalDetails(t *testing.T) {
	s := metadataModule(t, map[string]string{
		"/movies/42":      `{"title":"Minimal","type":"movie","released":"2010-07-15","ids":{"simkl":42},"poster":"p","fanart":"f"}`,
		"/tv/84":          `{"title":"Minimal","type":"show","first_aired":"2011-04-17","ids":{"simkl":84},"poster":"p","fanart":"f"}`,
		"/tv/episodes/84": `[]`,
	})
	movie := &polochon.Movie{ImdbID: "tt001", TmdbID: 123, Runtime: 42, Year: 1970}
	if err := s.GetDetails(context.Background(), movie); err != nil || movie.ImdbID != "tt001" || movie.TmdbID != 123 || movie.Runtime != 42 || movie.Year != 2010 {
		t.Errorf("optional movie fields: %+v, %v", movie, err)
	}
	show := &polochon.Show{ImdbID: "tt002", TvdbID: 456, Year: 1970}
	if err := s.GetDetails(context.Background(), show); err != nil || show.ImdbID != "tt002" || show.TvdbID != 456 || show.Year != 2011 || len(show.Episodes) != 0 {
		t.Errorf("optional show fields: %+v, %v", show, err)
	}
	s = metadataModule(t, map[string]string{
		"/tv/84":          `{"title":"Minimal","type":"show","ids":{"simkl":84},"poster":"p","fanart":"f"}`,
		"/tv/episodes/84": `[{"type":"episode","season":1,"episode":1,"date":null,"description":null,"runtime":null,"img":null}]`,
	})
	e := &polochon.ShowEpisode{ShowImdbID: "tt002", Season: 1, Episode: 1, Title: "Known", Plot: "Known plot", Aired: "2011-04-17", Runtime: 52, ThumbURL: "known.jpg"}
	if err := s.GetDetails(context.Background(), e); err != nil || e.Title != "Known" || e.Plot != "Known plot" || e.Aired != "2011-04-17" || e.Runtime != 52 || e.ThumbURL != "known.jpg" {
		t.Errorf("optional episode fields: %+v, %v", e, err)
	}
	if imageURL("posters", "", "_m") != "" {
		t.Error("missing image produced a URL")
	}
	if value, err := parseDate(""); err != nil || value != nil {
		t.Errorf("empty date = %v, %v", value, err)
	}
	if r, ok := (listing{Ratings: map[string]rating{"imdb": {Rating: 7}}}).rating(); !ok || r.Rating != 7 {
		t.Error("missing SIMKL rating did not fall back to IMDb")
	}
	if !strings.Contains(show.URL, "simkl.com/tv/84/") {
		t.Errorf("missing show source link: %s", show.URL)
	}
}
