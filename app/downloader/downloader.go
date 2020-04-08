package downloader

import (
	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/app/subapp"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/odwrtw/polochon/lib/library"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

// AppName is the application name
const AppName = "downloader"

// Downloader represents the downloader
type Downloader struct {
	*subapp.Base

	config  *configuration.Config
	library *library.Library
	event   chan struct{}
}

// New returns a new downloader
func New(config *configuration.Config, vs *library.Library) *Downloader {
	return &Downloader{
		Base:    subapp.NewBase(AppName),
		config:  config,
		library: vs,
	}
}

// Name returns the name of the app
func (d *Downloader) Name() string {
	return AppName
}

// Run starts the downloader
func (d *Downloader) Run(log *logrus.Entry) error {
	log = log.WithField("app", AppName)

	// Init the app
	d.InitStart(log)

	log.Debug("downloader started")
	d.event = make(chan struct{}, 1)

	if d.config.Downloader.LaunchAtStartup {
		log.Debug("initial downloader launch")
		d.event <- struct{}{}
	}

	// Start the scheduler
	d.Wg.Add(1)
	go func() {
		defer d.Wg.Done()
		d.scheduler(log)
	}()

	// Start the downloader
	var err error
	d.Wg.Add(1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err = errors.New("panic recovered").Fatal().AddContext(errors.Context{
					"sub_app": AppName,
				})
				d.Stop(log)
			}

			d.Wg.Done()
		}()
		d.downloader(log)
	}()

	defer log.Debug("downloader stopped")

	d.Wg.Wait()

	return err
}

func (d *Downloader) scheduler(log *logrus.Entry) {
	c := cron.New()
	c.Schedule(d.config.Downloader.Schedule, cron.FuncJob(func() {
		log.Debug("downloader scheduler triggered")
		d.event <- struct{}{}
	}))
	c.Start()

	<-d.Done
	log.Debug("downloader scheduler stopped")
	c.Stop()
}

func (d *Downloader) downloader(log *logrus.Entry) {
	for {
		select {
		case <-d.event:
			log.Debug("downloader event")
			d.downloadMissingVideos(log)
		case <-d.Done:
			log.Debug("downloader done handling events")
			return
		}
	}
}

func (d *Downloader) downloadMissingVideos(log *logrus.Entry) {
	// Fetch wishlist
	wl := polochon.NewWishlist(d.config.Wishlist, log)
	if err := wl.Fetch(); err != nil {
		log.Errorf("got an error while fetching wishlist: %q", err)
		return
	}

	d.downloadMissingMovies(wl, log)
	d.downloadMissingShows(wl, log)
}

func (d *Downloader) downloadMissingMovies(wl *polochon.Wishlist, log *logrus.Entry) {
	logger := log.WithField("function", "download_movies")

	for _, wantedMovie := range wl.Movies {
		log := logger.WithField("imdb_id", wantedMovie.ImdbID)

		ok, err := d.library.HasMovie(wantedMovie.ImdbID)
		if err != nil {
			log.Error(err)
			continue
		}

		if ok {
			log.Debugf("movie %q already in the video store", wantedMovie.ImdbID)
			continue
		}

		m := polochon.NewMovie(d.config.Movie)
		m.ImdbID = wantedMovie.ImdbID

		if err := polochon.GetDetails(m, log); err != nil {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				continue
			}
		}

		log = log.WithField("title", m.Title)

		if err := polochon.GetTorrents(m, log); err != nil && err != polochon.ErrMovieTorrentNotFound {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				continue
			}
		}

		var torrent *polochon.Torrent
	quality_loop:
		for _, q := range wantedMovie.Qualities {
			for _, t := range m.Torrents {
				if t.Quality == q {
					torrent = &t
					break quality_loop
				}
			}
		}

		if torrent == nil {
			log.Debug("no torrent found")
			continue
		}

		metadata := &polochon.DownloadableMetadata{
			Type:    "movie",
			ImdbID:  m.ImdbID,
			Quality: torrent.Quality,
		}

		if err := d.config.Downloader.Client.Download(torrent.URL, metadata, log); err != nil {
			log.Error(err)
			continue
		}
	}
}

func (d *Downloader) downloadMissingShows(wl *polochon.Wishlist, log *logrus.Entry) {
	logger := log.WithField("function", "download_shows")

	for _, wishedShow := range wl.Shows {
		log := logger.WithField("imdb_id", wishedShow.ImdbID)

		s := polochon.NewShow(d.config.Show)
		s.ImdbID = wishedShow.ImdbID

		if err := polochon.GetDetails(s, log); err != nil {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				continue
			}
		}

		calendar, err := s.GetCalendar(log)
		if err != nil {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				continue
			}
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
				log.Error(err)
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
			log = log.WithFields(logrus.Fields{
				"show_imdb_id": e.ShowImdbID,
				"show_title":   e.ShowTitle,
				"season":       e.Season,
				"episode":      e.Episode,
			})

			if err := polochon.GetTorrents(e, log); err != nil && err != polochon.ErrShowEpisodeTorrentNotFound {
				errors.LogErrors(log, err)
				if errors.IsFatal(err) {
					continue
				}
			}

			var torrent *polochon.Torrent
		quality_loop:
			for _, q := range wishedShow.Qualities {
				for _, t := range e.Torrents {
					if t.Quality == q {
						torrent = &t
						break quality_loop
					}
				}
			}

			if torrent == nil {
				log.Debug("no torrent found")
				continue
			}

			metadata := &polochon.DownloadableMetadata{
				Type:    "episode",
				ImdbID:  e.ShowImdbID,
				Quality: torrent.Quality,
				Season:  e.Season,
				Episode: e.Episode,
			}

			if err := d.config.Downloader.Client.Download(torrent.URL, metadata, log); err != nil {
				log.Error(err)
				continue
			}
		}
	}
}
