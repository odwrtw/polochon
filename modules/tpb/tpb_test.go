package tpb

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/odwrtw/tpb"
	"github.com/odwrtw/whatsthis"

	polochon "github.com/odwrtw/polochon/lib"
)

func newTestTPB(t *testing.T, torrents []*tpb.Torrent) (*TPB, <-chan string) {
	t.Helper()
	requests := make(chan string, 16)
	// TPB's wire format uses Unix timestamps, unlike Torrent.MarshalJSON.
	results := make([]map[string]any, 0, len(torrents))
	for _, torrent := range torrents {
		results = append(results, map[string]any{
			"name": torrent.Name, "username": torrent.User, "imdb": torrent.ImdbID,
			"seeders": torrent.Seeders, "leechers": torrent.Leechers, "size": torrent.Size,
			"magnet_link": torrent.MagnetLink,
		})
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.URL.RequestURI()
		if err := json.NewEncoder(w).Encode(results); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	t.Cleanup(server.Close)
	module := &TPB{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if err := module.InitWithParams(&Params{URLs: []string{server.URL}, MovieUsers: []string{"trusted"}, ShowUsers: []string{"trusted"}}); err != nil {
		t.Fatal(err)
	}
	return module, requests
}

func TestSearchTorrentsQuality(t *testing.T) {
	for _, tc := range []struct {
		name    string
		quality polochon.Quality
	}{
		{"Movie.2024.480P.x264-GROUP", polochon.Quality480p},
		{"Movie.2024.m720p.x264-GROUP", polochon.Quality720p},
		{"Movie.2024.720pBrRip.x264-GROUP", polochon.Quality720p},
		{"Movie (2024) 1080P WEB-DL x265-GROUP", polochon.Quality1080p},
		{"Movie.2024.4K.x265-GROUP", polochon.Quality2160p},
		{"Show Season 1 Complete 2160p x265-GROUP", polochon.Quality2160p},
		{"Show.S01-S04.2160p.x265-GROUP", polochon.Quality2160p},
		{"Movie.2024.720p.1080p.x264-GROUP", polochon.Quality1080p},
		{"Movie.2024.[3D].x264-GROUP", polochon.Quality3D},
		{"Movie.2024.[3D].1080p.x264-GROUP", polochon.Quality1080p},
		{"Saw.3D.x264-GROUP", "unknown"},
		{"Movie.2024.x264-3D", "unknown"},
		{"Movie.2024.x264-3DTEAM", "unknown"},
		{"Movie.2024.1440p.x264-GROUP", "unknown"},
		{"Movie.2024.x264-GROUP", "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module, requests := newTestTPB(t, []*tpb.Torrent{{Name: tc.name, User: "any-user", Seeders: 7, Leechers: 2, Size: 123, MagnetLink: "magnet:?xt=test"}})
			got, err := module.SearchTorrents("test query")
			if err != nil || len(got) != 1 {
				t.Fatalf("SearchTorrents = %+v, %v", got, err)
			}
			want := &polochon.Torrent{Quality: tc.quality, Result: &polochon.TorrentResult{
				Name: tc.name, URL: "magnet:?xt=test", Seeders: 7, Leechers: 2, Size: 123, Source: moduleName, UploadUser: "any-user",
			}}
			if !reflect.DeepEqual(got[0], want) {
				t.Fatalf("torrent = %+v, want %+v", got[0], want)
			}
			if path := <-requests; path != "/q.php?cat=200&q=test+query" {
				t.Fatalf("unexpected query: %s", path)
			}
		})
	}
}

func TestGetMovieTorrents(t *testing.T) {
	module, _ := newTestTPB(t, []*tpb.Torrent{
		{Name: "Example (2024) 4K WEB-DL x265-GROUP", User: "trusted", ImdbID: "tt1", Seeders: 3, MagnetLink: "magnet:?xt=less"},
		{Name: "Example (2024) 4K WEB-DL x265-GROUP", User: "trusted", ImdbID: "tt1", Seeders: 20, MagnetLink: "magnet:?xt=best"},
		{Name: "EXAMPLE.1080p.x264-GROUP", User: "trusted", Seeders: 4}, // Missing year is acceptable.
		{Name: "Example.2024.x264-GROUP", User: "trusted", Seeders: 5},  // Default movie quality.
		{Name: "Example.2023.480p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Other.2024.480p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.2024.S01.480p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.2024.1440p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.2024.480p.x264-GROUP", User: "untrusted", Seeders: 100},
		{Name: "Example.2024.480p.x264-GROUP", User: "trusted", ImdbID: "tt2", Seeders: 100},
	})
	movie := &polochon.Movie{Title: "Example", Year: 2024, ImdbID: "tt1"}
	if err := module.GetTorrents(context.Background(), movie); err != nil {
		t.Fatal(err)
	}
	if len(movie.Torrents) != 3 {
		t.Fatalf("torrents = %+v, want three qualities", movie.Torrents)
	}
	seeders := map[polochon.Quality]int{polochon.Quality720p: 5, polochon.Quality1080p: 4, polochon.Quality2160p: 20}
	for _, torrent := range movie.Torrents {
		if torrent.Result.Seeders != seeders[torrent.Quality] {
			t.Fatalf("invalid candidate was selected: %+v", torrent)
		}
		if torrent.Type != polochon.TypeMovie || torrent.ImdbID != movie.ImdbID || torrent.Season != 0 || torrent.Episode != 0 || torrent.Result.UploadUser != "trusted" {
			t.Fatalf("unexpected identity: %+v", torrent)
		}
	}
	best := polochon.ChooseTorrentFromQualities(movie.Torrents, []polochon.Quality{polochon.Quality2160p})
	if best == nil || best.Result.URL != "magnet:?xt=best" || best.Result.Seeders != 20 {
		t.Fatalf("unexpected UHD selection: %+v", best)
	}
}

func TestGetEpisodeTorrents(t *testing.T) {
	module, requests := newTestTPB(t, []*tpb.Torrent{
		{Name: "EXAMPLE.2024.S01E03.4K.x265-GROUP", User: "trusted", Seeders: 10},
		{Name: "Example S01E03 x264-GROUP", User: "trusted", Seeders: 5},
		{Name: "Example.2023.S01E03.720p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.2024.S01E04.720p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.2024.S02E03.720p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.2024.Season.1.Complete.720p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.S01E01-08.720p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.S01-S04.720p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.S01.720p.x264-GROUP.mkv", User: "trusted", Seeders: 100},
		{Name: "Other.S01E03.720p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.S01E03.1440p.x264-GROUP", User: "trusted", Seeders: 100},
		{Name: "Example.S01E03.720p.x264-GROUP", User: "untrusted", Seeders: 100},
		{Name: "Example.S01E03.720p.x264-GROUP", User: "trusted", ImdbID: "tt2", Seeders: 100},
	})
	episode := &polochon.ShowEpisode{ShowTitle: "Example", ShowImdbID: "tt1", Season: 1, Episode: 3, Show: &polochon.Show{Year: 2024}}
	if err := module.GetTorrents(context.Background(), episode); err != nil {
		t.Fatal(err)
	}
	if len(episode.Torrents) != 2 {
		t.Fatalf("torrents = %+v, want UHD and default quality", episode.Torrents)
	}
	for _, torrent := range episode.Torrents {
		if torrent.Type != polochon.TypeEpisode || torrent.ImdbID != "tt1" || torrent.Season != 1 || torrent.Episode != 3 {
			t.Fatalf("unexpected identity: %+v", torrent)
		}
	}
	if episode.Torrents[0].Quality != polochon.Quality2160p || episode.Torrents[1].Quality != polochon.Quality480p || episode.Torrents[1].Result.Seeders != 5 {
		t.Fatalf("unexpected qualities: %+v", episode.Torrents)
	}
	if path := <-requests; path != "/q.php?cat=200&q=Example+S01E03" {
		t.Fatalf("unexpected episode query: %s", path)
	}
}

func TestEpisodeLookupRejectsPacks(t *testing.T) {
	module, _ := newTestTPB(t, []*tpb.Torrent{
		{Name: "Example.S01E01-08.720p.x264-GROUP", User: "trusted"},
		{Name: "Example.S01E01-E08.720p.x264-GROUP", User: "trusted"},
		{Name: "Example.Season.1.Complete.720p.x264-GROUP", User: "trusted"},
		{Name: "Example.S01-S04.720p.x264-GROUP", User: "trusted"},
	})
	episode := &polochon.ShowEpisode{ShowTitle: "Example", ShowImdbID: "tt1", Season: 1, Episode: 1}
	if err := module.GetTorrents(context.Background(), episode); err != nil {
		t.Fatal(err)
	}
	if len(episode.Torrents) != 0 {
		t.Fatalf("packs must not be stamped as episode 1: %+v", episode.Torrents)
	}
}

func TestSearcherOptionalYears(t *testing.T) {
	movie := &movieSearcher{Movie: &polochon.Movie{Title: "Example"}}
	episode := &showSearcher{Episode: &polochon.ShowEpisode{ShowTitle: "Example", Season: 1, Episode: 3}}
	if !movie.isValidGuess(whatsthis.File("Example.2024.720p.x264-GROUP")) {
		t.Fatal("unknown movie year should not reject a supplied year")
	}
	if !episode.isValidGuess(whatsthis.File("Example.2024.S01E03.720p.x264-GROUP")) {
		t.Fatal("missing detailed show should not reject a supplied year")
	}
}

func TestExplorerTypesAndQuality(t *testing.T) {
	module, requests := newTestTPB(t, []*tpb.Torrent{
		{Name: "Movie - Part 2 (2024) 4K x265-GROUP", ImdbID: "tt1"},
		{Name: "Default.2024.x264-GROUP", ImdbID: "tt2"},
		{Name: "Unsupported.2024.1440p.x264-GROUP"},
		{Name: "Example.2024.S01E03.720p.x264-GROUP", ImdbID: "tt3"},
		{Name: "Example.2024.Season.1.Complete.1080p.x265-GROUP", ImdbID: "tt3"},
		{Name: "Example.2024.S01-S04.1080p.x265-GROUP", ImdbID: "tt3"},
	})
	for _, option := range module.AvailableMovieOptions() {
		movies, err := module.GetMovieList(context.Background(), option)
		if err != nil || len(movies) != 2 {
			t.Fatalf("GetMovieList = %+v, %v", movies, err)
		}
		if movies[0].Title != "Movie Part 2" || movies[0].Year != 2024 || movies[0].ImdbID != "tt1" || movies[0].Torrents[0].Quality != polochon.Quality2160p || movies[1].Torrents[0].Quality != polochon.Quality720p {
			t.Fatalf("unexpected movies: %+v, %+v", movies[0], movies[1])
		}
		wantPath := "/precompiled/data_top100_201.json"
		if option == "top100 HD" {
			wantPath = "/precompiled/data_top100_207.json"
		}
		if path := <-requests; path != wantPath {
			t.Fatalf("unexpected movie category: %s", path)
		}
	}
	for _, option := range module.AvailableShowOptions() {
		shows, err := module.GetShowList(context.Background(), option)
		if err != nil || len(shows) != 3 {
			t.Fatalf("GetShowList = %+v, %v", shows, err)
		}
		for _, show := range shows {
			if show.Title != "Example" || show.Year != 2024 || show.ImdbID != "tt3" || len(show.Episodes) != 0 {
				t.Fatalf("unexpected show discovery: %+v", show)
			}
		}
		wantPath := "/precompiled/data_top100_205.json"
		if option == "top100 HD" {
			wantPath = "/precompiled/data_top100_208.json"
		}
		if path := <-requests; path != wantPath {
			t.Fatalf("unexpected show category: %s", path)
		}
	}
	if _, err := module.GetMovieList(context.Background(), "invalid"); err != ErrInvalidArgument {
		t.Fatalf("unexpected movie option error: %v", err)
	}
	if _, err := module.GetShowList(context.Background(), "invalid"); err != ErrInvalidArgument {
		t.Fatalf("unexpected show option error: %v", err)
	}
}
