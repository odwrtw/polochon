package downloader

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/odwrtw/polochon/modules/mock"
)

type calendarModule struct {
	mock.Mock
	episodes  []*polochon.ShowCalendarEpisode
	searches  int
	downloads int
}

func (m *calendarModule) GetShowCalendar(_ context.Context, show *polochon.Show) (*polochon.ShowCalendar, error) {
	return &polochon.ShowCalendar{ImdbID: show.ImdbID, Episodes: m.episodes}, nil
}

func (m *calendarModule) GetTorrents(_ context.Context, value any) error {
	m.searches++
	episode := value.(*polochon.ShowEpisode)
	episode.Torrents = []*polochon.Torrent{{Quality: polochon.Quality720p}}
	return nil
}

func (m *calendarModule) Download(_ *polochon.Torrent) error {
	m.downloads++
	return nil
}

func TestDownloadMissingShowsAvailability(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-24*time.Hour), now.Add(24*time.Hour)
	for _, tc := range []struct {
		name      string
		airedDate *time.Time
		want      int
	}{
		{name: "aired", airedDate: &past, want: 1},
		{name: "future", airedDate: &future, want: 0},
		{name: "unknown air date", want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			module := &calendarModule{
				episodes: []*polochon.ShowCalendarEpisode{{Season: 1, Episode: 1, AiredDate: tc.airedDate}},
			}
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			config := &configuration.Config{
				Logger: log,
				Show: polochon.ShowConfig{
					Detailers:  []polochon.Detailer{module},
					Calendar:   module,
					Torrenters: []polochon.Torrenter{module},
				},
				Downloader: configuration.DownloaderConfig{Client: module},
			}
			wishlist := &polochon.Wishlist{
				Shows: []*polochon.WishedShow{{
					ImdbID: "tt1234567", Season: 1, Episode: 1,
					Qualities: []polochon.Quality{polochon.Quality720p},
				}},
			}
			d := New(config, library.New(config), log)
			d.downloadMissingShows(context.Background(), wishlist)

			if module.searches != tc.want {
				t.Errorf("torrent searches = %d, want %d", module.searches, tc.want)
			}
			if module.downloads != tc.want {
				t.Errorf("downloads = %d, want %d", module.downloads, tc.want)
			}
		})
	}
}
