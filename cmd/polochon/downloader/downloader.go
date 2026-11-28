package downloader

import (
	"context"
	"log/slog"
	"sync"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/robfig/cron/v3"
)

// AppName is the application name
const AppName = "downloader"

// videoLibrary is the part of the library used by the downloader, it exists
// so tests can replace the library with a fake
type videoLibrary interface {
	HasMovie(imdbID string) (bool, error)
	HasShowEpisode(imdbID string, season, episode int) (bool, error)
}

// Downloader represents the downloader
type Downloader struct {
	log     *slog.Logger
	config  *configuration.Config
	library videoLibrary
}

// New returns a new downloader
func New(config *configuration.Config, vs *library.Library, log *slog.Logger) *Downloader {
	return &Downloader{
		log:     log.With("app", AppName),
		config:  config,
		library: vs,
	}
}

// Run starts the downloader
func (d *Downloader) Run(ctx context.Context) error {
	d.log.Debug("downloader started")

	event := make(chan struct{}, 1)
	if d.config.Downloader.LaunchAtStartup {
		d.log.Debug("initial downloader launch")
		event <- struct{}{}
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		d.scheduler(ctx, event)
	})

	d.downloader(ctx, event)
	wg.Wait()

	d.log.Debug("downloader stopped")
	return nil
}

func (d *Downloader) scheduler(ctx context.Context, event chan<- struct{}) {
	c := cron.New()
	c.Schedule(d.config.Downloader.Schedule, cron.FuncJob(func() {
		d.log.Debug("downloader scheduler triggered")
		event <- struct{}{}
	}))
	c.Start()

	<-ctx.Done()
	d.log.Debug("downloader scheduler stopped")
	c.Stop()
}

func (d *Downloader) downloader(ctx context.Context, event <-chan struct{}) {
	for {
		select {
		case <-event:
			d.log.Debug("downloader event")
			d.downloadMissingVideos(ctx)
		case <-ctx.Done():
			d.log.Debug("downloader done handling events")
			return
		}
	}
}

func (d *Downloader) downloadMissingVideos(ctx context.Context) {
	// Fetch wishlist
	wl := polochon.NewWishlist(d.config.Wishlist, d.log)
	if err := wl.Fetch(ctx); err != nil {
		d.log.Error("got an error while fetching wishlist", "error", err)
		return
	}

	d.downloadMissingMovies(ctx, wl)
	d.downloadMissingShows(ctx, wl)
}

func (d *Downloader) downloadMissingMovies(ctx context.Context, wl *polochon.Wishlist) {
	log := d.log.With("function", "download_movies")

	for _, wantedMovie := range wl.Movies {
		log := log.With("imdb_id", wantedMovie.ImdbID)

		ok, err := d.library.HasMovie(wantedMovie.ImdbID)
		if err != nil {
			log.Error(err.Error())
			continue
		}

		if ok {
			log.Debug("movie already in the video store", "imdb_id", wantedMovie.ImdbID)
			continue
		}

		m := polochon.NewMovie(d.config.Movie)
		m.ImdbID = wantedMovie.ImdbID

		if err := polochon.GetDetails(ctx, m, log); err != nil {
			if err != polochon.ErrGettingDetails {
				log.Error(err.Error())
			}
			continue
		}

		log = log.With("title", m.Title)

		if err := polochon.GetTorrents(ctx, m); err != nil {
			if err == polochon.ErrTorrentNotFound {
				continue
			}

			log.Error(err.Error())
		}

		torrent := polochon.ChooseTorrentFromQualities(m.Torrents, wantedMovie.Qualities)
		if torrent == nil {
			log.Debug("no torrent found")
			continue
		}

		torrent.Type = polochon.TypeMovie
		torrent.ImdbID = m.ImdbID
		if err := d.config.Downloader.Client.Download(torrent); err != nil {
			log.Error(err.Error())
			continue
		}
	}
}

func (d *Downloader) downloadMissingShows(ctx context.Context, wl *polochon.Wishlist) {
	log := d.log.With("function", "download_shows")
	queued := d.queuedShowTorrents(log)

	for _, wishedShow := range wl.Shows {
		log := log.With("imdb_id", wishedShow.ImdbID)

		s := polochon.NewShow(d.config.Show)
		s.ImdbID = wishedShow.ImdbID

		if err := polochon.GetDetails(ctx, s, log); err != nil {
			if err != polochon.ErrGettingDetails {
				log.Error(err.Error())
			}

			continue
		}

		calendar, err := s.GetCalendar(ctx)
		if err != nil {
			if err == polochon.ErrCalendarNotFound {
				log.Info("calendar not found")
			} else {
				log.Error(err.Error())
			}
			continue
		}

		for _, season := range calendarSeasons(calendar) {
			d.downloadMissingSeason(ctx, log, s, wishedShow, season, queued)
		}
	}
}

// chooseEpisodeTorrent returns the best torrent for an episode, or nil
func (d *Downloader) chooseEpisodeTorrent(ctx context.Context, log *slog.Logger, s *polochon.Show, wishedShow *polochon.WishedShow, season, episode int) *polochon.Torrent {
	e := polochon.NewShowEpisode(d.config.Show)
	e.ShowImdbID = wishedShow.ImdbID
	e.ShowTitle = s.Title
	e.Season = season
	e.Episode = episode
	log = log.With(
		"show_imdb_id", e.ShowImdbID,
		"show_title", e.ShowTitle,
		"season", e.Season,
		"episode", e.Episode,
	)

	if err := polochon.GetTorrents(ctx, e); err != nil {
		if err != polochon.ErrTorrentNotFound {
			log.Error(err.Error())
		}
		return nil
	}

	torrent := polochon.ChooseTorrentFromQualities(e.Torrents, wishedShow.Qualities)
	if torrent == nil {
		log.Debug("no torrent found")
		return nil
	}

	torrent.Type = polochon.TypeEpisode
	torrent.ImdbID = e.ShowImdbID
	torrent.Season = e.Season
	torrent.Episode = e.Episode
	return torrent
}

func (d *Downloader) download(log *slog.Logger, torrent *polochon.Torrent) {
	if err := d.config.Downloader.Client.Download(torrent); err != nil {
		log.Error(err.Error())
	}
}
