package tmdb

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	tmdb "github.com/cyruzin/golang-tmdb"
	polochon "github.com/odwrtw/polochon/lib"
)

func fakeTVFindByID(t *testing.T, id int64) *tmdb.FindByID {
	t.Helper()

	result := &tmdb.FindByID{}
	if err := json.Unmarshal([]byte(`{"tv_results":[{"id":42}]}`), result); err != nil {
		t.Fatalf("decode TMDB find response: %v", err)
	}
	result.TvResults[0].ID = id
	return result
}

func TestTmdbGetShowDetailsUsesImdbWhenTMDBIDIsMissing(t *testing.T) {
	oldFind, oldInfo, oldExternal := tmdbFindTVByExternalID, tmdbGetTVInfo, tmdbGetTVExternalIDs
	t.Cleanup(func() {
		tmdbFindTVByExternalID = oldFind
		tmdbGetTVInfo = oldInfo
		tmdbGetTVExternalIDs = oldExternal
	})
	tmdbFindTVByExternalID = func(_ *tmdb.Client, id, source string, _ map[string]string) (*tmdb.FindByID, error) {
		if id != "tt0133093" || source != "imdb_id" {
			t.Fatalf("external lookup = (%q, %q), want (%q, %q)", id, source, "tt0133093", "imdb_id")
		}
		return fakeTVFindByID(t, 42), nil
	}
	tmdbGetTVInfo = func(_ *tmdb.Client, id int, _ map[string]string) (*tmdb.TVDetails, error) {
		if id != 42 {
			t.Fatalf("show TMDB ID = %d, want 42", id)
		}
		return &tmdb.TVDetails{ID: 42, Name: "The Matrix"}, nil
	}
	tmdbGetTVExternalIDs = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVExternalIDs, error) {
		return &tmdb.TVExternalIDs{}, nil
	}

	show := &polochon.Show{ImdbID: "tt0133093"}
	if err := (&TmDB{}).GetDetails(context.Background(), show); err != nil {
		t.Fatal(err)
	}
	if show.TmdbID != 42 || show.ImdbID != "tt0133093" {
		t.Fatalf("show identifiers = (tmdb=%d, imdb=%q), want (tmdb=%d, imdb=%q)", show.TmdbID, show.ImdbID, 42, "tt0133093")
	}
}

func TestTmdbGetShowDetails(t *testing.T) {
	oldInfo, oldExternal := tmdbGetTVInfo, tmdbGetTVExternalIDs
	t.Cleanup(func() {
		tmdbGetTVInfo = oldInfo
		tmdbGetTVExternalIDs = oldExternal
	})

	tmdbGetTVInfo = func(_ *tmdb.Client, id int, _ map[string]string) (*tmdb.TVDetails, error) {
		if id != 42 {
			t.Fatalf("show TMDB ID = %d, want 42", id)
		}
		return &tmdb.TVDetails{
			ID:           42,
			Name:         "Example",
			Overview:     "A plot",
			FirstAirDate: "2020-01-02",
			VoteMetrics:  tmdb.VoteMetrics{VoteAverage: 8.5},
			Homepage:     "https://example.test",
			PosterPath:   "/poster.jpg",
			BackdropPath: "/backdrop.jpg",
		}, nil
	}
	tmdbGetTVExternalIDs = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVExternalIDs, error) {
		return &tmdb.TVExternalIDs{IMDbID: "tt-example", TVDBID: 84}, nil
	}

	show := &polochon.Show{TmdbID: 42, Episodes: []*polochon.ShowEpisode{{}}}
	if err := (&TmDB{}).GetDetails(context.Background(), show); err != nil {
		t.Fatal(err)
	}

	if show.TmdbID != 42 || show.ImdbID != "tt-example" || show.TvdbID != 84 || show.Year != 2020 || show.Title != "Example" {
		t.Fatalf("show IDs or basic metadata not mapped: %+v", show)
	}
	if show.FirstAired == nil || show.FirstAired.Format("2006-01-02") != "2020-01-02" {
		t.Fatalf("first air date not mapped: %v", show.FirstAired)
	}
	if show.URL != "https://example.test" || show.PosterURL != TmDBimageBaseURL+"/poster.jpg" || show.FanartURL != TmDBimageBaseURL+"/backdrop.jpg" || show.BannerURL != show.FanartURL {
		t.Fatalf("show artwork or homepage not mapped: %+v", show)
	}
	if len(show.Episodes) != 1 {
		t.Fatal("show detailing unexpectedly populated or replaced episodes")
	}
}

func TestTmdbShowDetailsLeavesAbsentArtworkEmpty(t *testing.T) {
	oldInfo, oldExternal := tmdbGetTVInfo, tmdbGetTVExternalIDs
	t.Cleanup(func() {
		tmdbGetTVInfo = oldInfo
		tmdbGetTVExternalIDs = oldExternal
	})

	tmdbGetTVInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVDetails, error) {
		return &tmdb.TVDetails{ID: 1, Name: "Example"}, nil
	}
	tmdbGetTVExternalIDs = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVExternalIDs, error) {
		return &tmdb.TVExternalIDs{}, nil
	}

	show := &polochon.Show{TmdbID: 1}
	if err := (&TmDB{}).GetDetails(context.Background(), show); err != nil {
		t.Fatal(err)
	}
	if show.PosterURL != "" || show.FanartURL != "" || show.BannerURL != "" {
		t.Fatalf("absent artwork produced URLs: %+v", show)
	}
}

func TestTmdbShowIdentityConflictDoesNotMutate(t *testing.T) {
	oldInfo, oldExternal := tmdbGetTVInfo, tmdbGetTVExternalIDs
	t.Cleanup(func() {
		tmdbGetTVInfo = oldInfo
		tmdbGetTVExternalIDs = oldExternal
	})

	tmdbGetTVInfo = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVDetails, error) {
		return &tmdb.TVDetails{ID: 1, Name: "resolved"}, nil
	}
	tmdbGetTVExternalIDs = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVExternalIDs, error) {
		return &tmdb.TVExternalIDs{IMDbID: "tt-other"}, nil
	}

	show := &polochon.Show{TmdbID: 1, ImdbID: "tt-original", Title: "original", Year: 1999}
	before := *show
	if err := (&TmDB{}).GetDetails(context.Background(), show); err != ErrIdentityConflict {
		t.Fatalf("error = %v, want %v", err, ErrIdentityConflict)
	}
	if !reflect.DeepEqual(*show, before) {
		t.Fatalf("show was mutated on conflict: got %+v, want %+v", show, &before)
	}
}

func TestTmdbGetEpisodeDetails(t *testing.T) {
	oldInfo, oldExternal, oldEpisode, oldEpisodeExternal := tmdbGetTVInfo, tmdbGetTVExternalIDs, tmdbGetTVEpisodeInfo, tmdbGetTVEpisodeExternalIDs
	t.Cleanup(func() {
		tmdbGetTVInfo = oldInfo
		tmdbGetTVExternalIDs = oldExternal
		tmdbGetTVEpisodeInfo = oldEpisode
		tmdbGetTVEpisodeExternalIDs = oldEpisodeExternal
	})

	tmdbGetTVInfo = func(_ *tmdb.Client, id int, _ map[string]string) (*tmdb.TVDetails, error) {
		return &tmdb.TVDetails{ID: int64(id), Name: "Example"}, nil
	}
	tmdbGetTVExternalIDs = func(_ *tmdb.Client, _ int, _ map[string]string) (*tmdb.TVExternalIDs, error) {
		return &tmdb.TVExternalIDs{IMDbID: "tt-show", TVDBID: 84}, nil
	}
	tmdbGetTVEpisodeInfo = func(_ *tmdb.Client, id, season, episode int, _ map[string]string) (*tmdb.TVEpisodeDetails, error) {
		if id != 42 || season != 2 || episode != 3 {
			t.Fatalf("episode lookup = (%d, %d, %d), want (42, 2, 3)", id, season, episode)
		}
		return &tmdb.TVEpisodeDetails{
			ID:            99,
			SeasonNumber:  2,
			EpisodeNumber: 3,
			Name:          "Episode",
			AirDate:       "2020-02-03",
			Overview:      "Episode plot",
			Runtime:       45,
			StillPath:     "/still.jpg",
			VoteMetrics:   tmdb.VoteMetrics{VoteAverage: 7.5},
		}, nil
	}
	tmdbGetTVEpisodeExternalIDs = func(_ *tmdb.Client, _, _, _ int) (*tmdb.TVEpisodeExternalIDs, error) {
		return &tmdb.TVEpisodeExternalIDs{IMDbID: "tt-episode", TVDBID: 99}, nil
	}

	episode := &polochon.ShowEpisode{
		Show:    &polochon.Show{TmdbID: 42},
		Season:  2,
		Episode: 3,
	}
	if err := (&TmDB{}).GetDetails(context.Background(), episode); err != nil {
		t.Fatal(err)
	}
	if episode.Title != "Episode" || episode.ShowTitle != "Example" || episode.ShowImdbID != "tt-show" || episode.ShowTvdbID != 84 || episode.EpisodeImdbID != "tt-episode" || episode.TvdbID != 99 {
		t.Fatalf("episode IDs or titles not mapped: %+v", episode)
	}
	if episode.Aired != "2020-02-03" || episode.Plot != "Episode plot" || episode.Runtime != 45 || episode.Rating != 7.5 || episode.ThumbURL != TmDBimageBaseURL+"/still.jpg" {
		t.Fatalf("episode metadata not mapped: %+v", episode)
	}
}
