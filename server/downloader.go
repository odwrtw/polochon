package main

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/lib"
)

type downloader struct {
	config     *Config
	videoStore *polochon.VideoStore
	event      chan struct{}
	done       chan struct{}
	stop       chan struct{}
	errc       chan error
	wg         *sync.WaitGroup
	log        *logrus.Entry
}

func (d *downloader) downloadDaemon() {
	d.log.Info("Starting downloader")

	// Start the ticker
	go d.ticker()

	// Start the downloader
	go d.downloader()

	// Lauch the downloader at startup
	go func() {
		d.log.Debug("initial downloader launch")
		d.event <- struct{}{}
	}()
}

func (d *downloader) ticker() {
	d.wg.Add(1)
	defer d.wg.Done()
	tick := time.Tick(d.config.Downloader.Timer)
	for {
		select {
		case <-tick:
			d.log.Debug("Downloader timer triggered")
			d.event <- struct{}{}
		case <-d.done:
			return
		}
	}
}

func (d *downloader) downloader() {
	d.wg.Add(1)
	defer d.wg.Done()
	for {
		select {
		case <-d.event:
			d.log.Info("Downloader event !")
			d.downloadMissingVideos()
		case <-d.done:
			d.log.Info("Downloader done")
			return
		}
	}
}

func (d *downloader) downloadMissingVideos() {
	// Fetch wishlist
	wl := polochon.NewWishlist(d.config.Wishlist, d.log)
	if err := wl.Fetch(); err != nil {
		d.log.Errorf("Got an error while fetching wishlist: %q", err)
		return
	}

	d.downloadMissingMovies(wl)
	d.downloadMissingShows(wl)
}

func (d *downloader) downloadMissingMovies(wl *polochon.Wishlist) {
	for _, wantedMovie := range wl.Movies {
		ok, err := d.videoStore.HasMovie(wantedMovie.ImdbID)
		if err != nil {
			d.errc <- err
			continue
		}

		if ok {
			d.log.Debugf("Movie %q already in the video store", wantedMovie.ImdbID)
			continue
		}

		m := polochon.NewMovie(d.config.Movie)
		m.ImdbID = wantedMovie.ImdbID
		log := d.log.WithField("imdbID", m.ImdbID)

		if err := m.GetDetails(log); err != nil {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				continue
			}
		}

		log = d.log.WithField("title", m.Title)

		if err := m.GetTorrents(log); err != nil && err != polochon.ErrMovieTorrentNotFound {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				continue
			}
		}

		// Keep the torrent URL
		var torrentURL string
	quality_loop:
		for _, q := range wantedMovie.Qualities {
			for _, t := range m.Torrents {
				if t.Quality == q {
					torrentURL = t.URL
					break quality_loop
				}
			}
		}

		if torrentURL == "" {
			log.Debug("no torrent found")
			continue
		}

		if err := d.config.Downloader.Client.Download(torrentURL, d.log); err != nil {
			log.Error(err)
			continue
		}
	}
}

func (d *downloader) downloadMissingShows(wl *polochon.Wishlist) {
	for _, wishedShow := range wl.Shows {
		s := polochon.NewShow(d.config.Show)
		s.ImdbID = wishedShow.ImdbID
		log := d.log.WithField("imdbID", s.ImdbID)

		if err := s.GetDetails(log); err != nil {
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
			// Check if the episode should be downloaded
			if calEpisode.IsOlder(wishedShow) {
				continue
			}

			// Check if the episode has already been downloaded
			ok, err := d.videoStore.HasShowEpisode(wishedShow.ImdbID, calEpisode.Season, calEpisode.Episode)
			if err != nil {
				d.errc <- err
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
			log := d.log.WithFields(logrus.Fields{
				"show_imdb_id": e.ShowImdbID,
				"show_title":   e.ShowTitle,
				"season":       e.Season,
				"episode":      e.Episode,
			})

			if err := e.GetTorrents(log); err != nil && err != polochon.ErrShowEpisodeTorrentNotFound {
				errors.LogErrors(log, err)
				if errors.IsFatal(err) {
					continue
				}
			}

			// Keep the torrent URL
			var torrentURL string

		quality_loop:
			for _, q := range wishedShow.Qualities {
				for _, t := range e.Torrents {
					if t.Quality == q {
						torrentURL = t.URL
						break quality_loop
					}
				}
			}

			if torrentURL == "" {
				log.Debug("no torrent found")
				continue
			}

			if err := d.config.Downloader.Client.Download(torrentURL, d.log); err != nil {
				d.errc <- err
				continue
			}
		}
	}
}

// Run launches the downloader
func (app *App) startDownloader() {
	app.downloader = &downloader{
		config:     app.config,
		videoStore: app.videoStore,
		event:      make(chan struct{}),
		done:       app.done,
		stop:       app.stop,
		errc:       app.errc,
		wg:         &app.wg,
		log:        app.logger.WithField("function", "downloader"),
	}

	app.downloader.downloadDaemon()
}
