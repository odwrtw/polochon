package downloader

import (
	"context"
	"log/slog"

	"github.com/odwrtw/polochon/app/subapp"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/robfig/cron/v3"
)

// AppName is the application name
const AppName = "downloader"

// Downloader represents the downloader
type Downloader struct {
	*subapp.Base

	log     *slog.Logger
	config  *configuration.Config
	library *library.Library
	event   chan struct{}
}

// New returns a new downloader
func New(config *configuration.Config, vs *library.Library, log *slog.Logger) *Downloader {
	l := log.With("app", AppName)
	return &Downloader{
		Base:    subapp.NewBase(AppName, l),
		log:     l,
		config:  config,
		library: vs,
	}
}

// Name returns the name of the app
func (d *Downloader) Name() string {
	return AppName
}

// Run starts the downloader
func (d *Downloader) Run(ctx context.Context) error {
	// Init the app
	d.InitStart()

	d.log.Debug("downloader started")
	d.event = make(chan struct{}, 1)

	if d.config.Downloader.LaunchAtStartup {
		d.log.Debug("initial downloader launch")
		d.event <- struct{}{}
	}

	// Start the scheduler
	d.Wg.Go(func() {
		d.scheduler()
	})

	// Start the downloader
	var err error
	d.Wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err = subapp.ErrPanicRecovered
				d.Stop()
			}

			d.Wg.Done()
		}()
		d.downloader(ctx)
	}()

	defer d.log.Debug("downloader stopped")

	d.Wg.Wait()

	return err
}

func (d *Downloader) scheduler() {
	c := cron.New()
	c.Schedule(d.config.Downloader.Schedule, cron.FuncJob(func() {
		d.log.Debug("downloader scheduler triggered")
		d.event <- struct{}{}
	}))
	c.Start()

	<-d.Done
	d.log.Debug("downloader scheduler stopped")
	c.Stop()
}

func (d *Downloader) downloader(ctx context.Context) {
	for {
		select {
		case <-d.event:
			d.log.Debug("downloader event")
			d.downloadMissingVideos(ctx)
		case <-d.Done:
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

		for _, calEpisode := range calendar.Episodes {
			if calEpisode.Season == 0 {
				// Skip the show "Specials" episodes
				continue
			}

			// Check if the episode should be downloaded
			if calEpisode.IsOlder(wishedShow) {
				continue
			}

			// Check if the episode has already been downloaded
			ok, err := d.library.HasShowEpisode(wishedShow.ImdbID, calEpisode.Season, calEpisode.Episode)
			if err != nil {
				log.Error(err.Error())
				continue
			}

			if ok {
				continue
			}

			// Setup the episode
			e := polochon.NewShowEpisode(d.config.Show)
			e.ShowImdbID = wishedShow.ImdbID
			e.ShowTitle = s.Title
			e.Season = calEpisode.Season
			e.Episode = calEpisode.Episode
			log = log.With(
				"show_imdb_id", e.ShowImdbID,
				"show_title", e.ShowTitle,
				"season", e.Season,
				"episode", e.Episode,
			)

			err = polochon.GetTorrents(ctx, e)
			if err != nil {
				if err != polochon.ErrTorrentNotFound {
					log.Error(err.Error())
				}

				continue
			}

			torrent := polochon.ChooseTorrentFromQualities(e.Torrents, wishedShow.Qualities)
			if torrent == nil {
				log.Debug("no torrent found")
				continue
			}

			torrent.Type = polochon.TypeEpisode
			torrent.ImdbID = e.ShowImdbID
			torrent.Season = e.Season
			torrent.Episode = e.Episode
			if err := d.config.Downloader.Client.Download(torrent); err != nil {
				log.Error(err.Error())
				continue
			}
		}
	}
}
