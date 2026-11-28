package downloader

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/modules/mock"
)

const packShowID = "tt1234567"

var errPackTest = errors.New("pack test error")

// packModule is the detailer, calendar and download client of the tests
type packModule struct {
	mock.Mock
	calendar  []*polochon.ShowCalendarEpisode
	single    polochon.Quality
	searches  []string
	downloads []*polochon.Torrent
	failAdd   bool
	queued    []*polochon.Torrent
}

func (m *packModule) List() ([]*polochon.Torrent, error) {
	return m.queued, nil
}

func (m *packModule) GetDetails(_ context.Context, v any) error {
	s := v.(*polochon.Show)
	s.Title = "Show"
	s.Year = 2026
	return nil
}

func (m *packModule) GetShowCalendar(_ context.Context, s *polochon.Show) (*polochon.ShowCalendar, error) {
	return &polochon.ShowCalendar{ImdbID: s.ImdbID, Episodes: m.calendar}, nil
}

func (m *packModule) GetTorrents(_ context.Context, v any) error {
	e, ok := v.(*polochon.ShowEpisode)
	if !ok {
		return errPackTest
	}
	m.searches = append(m.searches, fmt.Sprintf("S%02dE%02d", e.Season, e.Episode))
	if m.single == "" {
		return polochon.ErrTorrentNotFound
	}
	e.Torrents = []*polochon.Torrent{{
		Quality: m.single,
		Result:  &polochon.TorrentResult{Name: "single", URL: "single", Seeders: 1},
	}}
	return nil
}

func (m *packModule) Download(t *polochon.Torrent) error {
	m.downloads = append(m.downloads, t)
	if m.failAdd {
		return errPackTest
	}
	return nil
}

// packTorrenter returns its results for any season
type packTorrenter struct {
	mock.Mock
	results []*polochon.Torrent
	seasons []*polochon.ShowSeason
}

func (p *packTorrenter) GetTorrents(_ context.Context, v any) error {
	ss := v.(*polochon.ShowSeason)
	p.seasons = append(p.seasons, ss)
	ss.Torrents = p.results
	return nil
}

type packLibrary struct {
	have map[int]bool
}

func (l *packLibrary) HasMovie(string) (bool, error) { return false, nil }

func (l *packLibrary) HasShowEpisode(_ string, _, episode int) (bool, error) {
	return l.have[episode], nil
}

func episodes(season, from, to int, aired *time.Time) []*polochon.ShowCalendarEpisode {
	var out []*polochon.ShowCalendarEpisode
	for e := from; e <= to; e++ {
		out = append(out, &polochon.ShowCalendarEpisode{Season: season, Episode: e, AiredDate: aired})
	}
	return out
}

func result(name string, seeders int, q polochon.Quality) *polochon.Torrent {
	return &polochon.Torrent{
		Quality: q,
		Result:  &polochon.TorrentResult{Name: name, URL: "magnet:" + name, Seeders: seeders},
	}
}

// describe summarizes the downloads, a pack being "<name> <quality>"
func describe(downloads []*polochon.Torrent) []string {
	var out []string
	for _, t := range downloads {
		if t.Type == polochon.TypeSeason {
			out = append(out, fmt.Sprintf("%s %s", t.Result.Name, t.Quality))
			continue
		}
		out = append(out, fmt.Sprintf("S%02dE%02d", t.Season, t.Episode))
	}
	return out
}

func singles(season, from, to int) []string {
	var out []string
	for e := from; e <= to; e++ {
		out = append(out, fmt.Sprintf("S%02dE%02d", season, e))
	}
	return out
}

func TestDownloadMissingSeason(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-24*time.Hour), now.Add(24*time.Hour)
	q720, q1080, q2160 := polochon.Quality720p, polochon.Quality1080p, polochon.Quality2160p
	plain := []*polochon.Torrent{result("Show S01 720p", 10, q720)}

	for _, tc := range []struct {
		name       string
		calendar   []*polochon.ShowCalendarEpisode
		from       int
		have       []int
		qualities  []polochon.Quality
		single     polochon.Quality
		failAdd    bool
		queued     []*polochon.Torrent
		torrenters []*packTorrenter
		want       []string
		wantQuery  bool
	}{
		{
			name:     "exactly half missing",
			calendar: episodes(1, 1, 8, &past), have: []int{1, 2, 3, 4},
			torrenters: []*packTorrenter{{results: plain}},
			want:       singles(1, 5, 8),
		},
		{
			name:     "more than half missing",
			calendar: episodes(1, 1, 8, &past), have: []int{1, 2, 3},
			torrenters: []*packTorrenter{{results: plain}},
			want:       []string{"Show S01 720p 720p"}, wantQuery: true,
		},
		{
			name:       "unaired episodes count in the threshold",
			calendar:   append(episodes(1, 1, 4, &past), episodes(1, 5, 8, &future)...),
			torrenters: []*packTorrenter{{results: plain}},
			want:       singles(1, 1, 4),
		},
		{
			name:       "duplicates and specials are ignored",
			calendar:   slices.Concat(episodes(0, 1, 9, &past), episodes(1, 1, 4, &past), episodes(1, 1, 4, &past)),
			have:       []int{1, 2},
			torrenters: []*packTorrenter{{results: plain}},
			want:       singles(1, 3, 4),
		},
		{
			name:       "plain pack with an unknown air date",
			calendar:   append(episodes(1, 1, 3, &past), episodes(1, 4, 4, nil)...),
			torrenters: []*packTorrenter{{results: plain}},
			want:       singles(1, 1, 4), wantQuery: true,
		},
		{
			name:       "plain pack with a future episode",
			calendar:   append(episodes(1, 1, 7, &past), episodes(1, 8, 8, &future)...),
			torrenters: []*packTorrenter{{results: plain}},
			want:       singles(1, 1, 7), wantQuery: true,
		},
		{
			name:     "plain pack for a partially tracked season",
			calendar: episodes(1, 1, 8, &past), from: 3,
			torrenters: []*packTorrenter{{results: plain}},
			want:       singles(1, 3, 8), wantQuery: true,
		},
		{
			name:     "range matching the tracked episodes",
			calendar: episodes(1, 1, 8, &past), from: 3,
			torrenters: []*packTorrenter{{results: []*polochon.Torrent{result("Show.S01E03-08.720p", 10, "")}}},
			want:       []string{"Show.S01E03-08.720p 720p"}, wantQuery: true,
		},
		{
			name:     "range not matching the tracked episodes",
			calendar: episodes(1, 1, 8, &past), from: 3,
			torrenters: []*packTorrenter{{results: []*polochon.Torrent{
				result("Show.S01E01-08.720p", 10, q720),
				result("Show.S01E03-07.720p", 10, q720),
			}}},
			want: singles(1, 3, 8), wantQuery: true,
		},
		{
			name:     "names that are not this season pack",
			calendar: episodes(1, 1, 4, &past),
			torrenters: []*packTorrenter{{results: []*polochon.Torrent{
				result("Other S01 720p", 10, q720),
				result("Show (2019) S01 720p", 10, q720),
				result("Show S02 720p", 10, q720),
				result("Show S01-S02 720p", 10, q720),
				result("Show S01E03 720p", 10, q720),
				result("Show S01 no seeders 720p", 0, q720),
				nil,
				{Quality: q720},
			}}},
			want: singles(1, 1, 4), wantQuery: true,
		},
		{
			name:       "year and complete marker",
			calendar:   episodes(1, 1, 4, &past),
			torrenters: []*packTorrenter{{results: []*polochon.Torrent{result("SHOW (2026) Season 1 Complete 720p", 10, q720)}}},
			want:       []string{"SHOW (2026) Season 1 Complete 720p 720p"}, wantQuery: true,
		},
		{
			name:     "next torrenter when the first has no pack",
			calendar: episodes(1, 1, 4, &past),
			torrenters: []*packTorrenter{
				{},
				{results: []*polochon.Torrent{result("Show S01 720p second", 10, q720)}},
				{results: plain},
			},
			want: []string{"Show S01 720p second 720p"}, wantQuery: true,
		},
		{
			name:     "quality from the name",
			calendar: episodes(1, 1, 4, &past), qualities: []polochon.Quality{q2160},
			torrenters: []*packTorrenter{{results: []*polochon.Torrent{result("Show S01 4K", 10, "")}}},
			want:       []string{"Show S01 4K 2160p"}, wantQuery: true,
		},
		{
			name:     "better single episodes win",
			calendar: episodes(1, 1, 4, &past), qualities: []polochon.Quality{q1080, q720}, single: q1080,
			torrenters: []*packTorrenter{{results: plain}},
			want:       singles(1, 1, 4), wantQuery: true,
		},
		{
			name:     "equal single episodes lose",
			calendar: episodes(1, 1, 4, &past), qualities: []polochon.Quality{q1080, q720},
			torrenters: []*packTorrenter{{results: plain}},
			want:       []string{"Show S01 720p 720p"}, wantQuery: true,
		},
		{
			name:     "failed pack add",
			calendar: episodes(1, 1, 4, &past), failAdd: true,
			torrenters: []*packTorrenter{{results: plain}},
			want:       []string{"Show S01 720p 720p"}, wantQuery: true,
		},
		{
			name:     "pack already queued",
			calendar: episodes(1, 1, 8, &past),
			queued: []*polochon.Torrent{
				{ImdbID: "tt7654321", Type: polochon.TypeEpisode, Season: 1, Episode: 1},
				{ImdbID: packShowID, Type: polochon.TypeSeason, Season: 1},
			},
			torrenters: []*packTorrenter{{results: plain}},
		},
		{
			name:     "queued episodes are not missing",
			calendar: episodes(1, 1, 8, &past),
			queued: []*polochon.Torrent{
				{ImdbID: packShowID, Type: polochon.TypeSeason, Season: 2},
				{ImdbID: packShowID, Type: polochon.TypeEpisode, Season: 1, Episode: 1},
				{ImdbID: packShowID, Type: polochon.TypeEpisode, Season: 1, Episode: 2},
				{ImdbID: packShowID, Type: polochon.TypeEpisode, Season: 1, Episode: 3},
				{ImdbID: packShowID, Type: polochon.TypeEpisode, Season: 1, Episode: 4},
			},
			torrenters: []*packTorrenter{{results: plain}},
			want:       singles(1, 5, 8),
		},
		{
			name:       "no pack found",
			calendar:   episodes(1, 1, 4, &past),
			torrenters: []*packTorrenter{{}},
			want:       singles(1, 1, 4), wantQuery: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module := &packModule{calendar: tc.calendar, single: q720, failAdd: tc.failAdd, queued: tc.queued}
			if tc.single != "" {
				module.single = tc.single
			}
			qualities := tc.qualities
			if qualities == nil {
				qualities = []polochon.Quality{q720}
			}
			from := max(tc.from, 1)

			torrenters := []polochon.Torrenter{module}
			for _, p := range tc.torrenters {
				torrenters = append(torrenters, p)
			}
			lib := &packLibrary{have: map[int]bool{}}
			for _, e := range tc.have {
				lib.have[e] = true
			}

			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			d := &Downloader{
				log: log,
				config: &configuration.Config{
					Show: polochon.ShowConfig{
						Detailers:  []polochon.Detailer{module},
						Calendar:   module,
						Torrenters: torrenters,
					},
					Downloader: configuration.DownloaderConfig{Client: module},
				},
				library: lib,
			}
			d.downloadMissingShows(context.Background(), &polochon.Wishlist{
				Shows: []*polochon.WishedShow{{ImdbID: packShowID, Season: 1, Episode: from, Qualities: qualities}},
			})

			if got := describe(module.downloads); !slices.Equal(got, tc.want) {
				t.Fatalf("downloads = %q, want %q", got, tc.want)
			}

			queried := false
			for _, p := range tc.torrenters {
				queried = queried || len(p.seasons) > 0
			}
			if queried != tc.wantQuery {
				t.Fatalf("pack searched = %t, want %t", queried, tc.wantQuery)
			}
		})
	}
}

func TestSeasonPackSubmission(t *testing.T) {
	aired := time.Now().Add(-time.Hour)
	module := &packModule{calendar: episodes(1, 1, 4, &aired), single: polochon.Quality720p}
	first := &packTorrenter{results: []*polochon.Torrent{
		result("Show S01 720p", 5, polochon.Quality720p),
		result("Show S01 720p", 20, polochon.Quality720p),
		result("Show S01 720p other", 10, polochon.Quality720p),
	}}
	first.results[1].ImdbID = "tt0000001"
	first.results[1].Type = polochon.TypeEpisode
	first.results[1].Season = 1
	first.results[1].Episode = 2
	second := &packTorrenter{}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	d := &Downloader{
		log: log,
		config: &configuration.Config{
			Show: polochon.ShowConfig{
				Detailers:  []polochon.Detailer{module},
				Calendar:   module,
				Torrenters: []polochon.Torrenter{module, first, second},
			},
			Downloader: configuration.DownloaderConfig{Client: module},
		},
		library: &packLibrary{},
	}
	d.downloadMissingShows(context.Background(), &polochon.Wishlist{
		Shows: []*polochon.WishedShow{{ImdbID: packShowID, Season: 1, Episode: 1, Qualities: []polochon.Quality{polochon.Quality720p}}},
	})

	if len(first.seasons) != 1 {
		t.Fatalf("seasons = %+v", first.seasons)
	}
	if ss := first.seasons[0]; ss.ShowImdbID != packShowID || ss.ShowTitle != "Show" || ss.Season != 1 {
		t.Fatalf("season = %+v", ss)
	}
	if len(second.seasons) != 0 {
		t.Fatal("searched the next torrenter after a pack was found")
	}
	if len(module.searches) != 0 {
		t.Fatalf("searched single episodes for a pack in the preferred quality: %q", module.searches)
	}
	if len(module.downloads) != 1 {
		t.Fatalf("downloads = %q", describe(module.downloads))
	}

	got := module.downloads[0]
	if got.ImdbID != packShowID || got.Type != polochon.TypeSeason || got.Season != 1 || got.Episode != 0 || got.Status != nil {
		t.Fatalf("pack torrent is not labelled as a season: %+v", got)
	}
	if got.Result.Name != "Show S01 720p" || got.Result.Seeders != 20 {
		t.Fatalf("pack = %+v, want the duplicate with the most seeders", got.Result)
	}
}
