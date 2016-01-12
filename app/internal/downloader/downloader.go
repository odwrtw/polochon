package downloader

import (
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/app/internal/configuration"
	"github.com/odwrtw/polochon/lib"
)

// AppName is the application name
const AppName = "downloader"

// Downloader represents the downloader
type Downloader struct {
	config     *configuration.Config
	videoStore *polochon.VideoStore
	event      chan struct{}
	done       chan struct{}
}

// New returns a new downloader
func New(config *configuration.Config, vs *polochon.VideoStore) *Downloader {
	return &Downloader{
		config:     config,
		videoStore: vs,
		event:      make(chan struct{}),
		done:       make(chan struct{}),
	}
}

// Name returns the name of the app
func (d *Downloader) Name() string {
	return AppName
}

// Run starts the downloader
func (d *Downloader) Run(log *logrus.Entry) error {
	log = log.WithField("app", AppName)

	log.Debug("downloader started")

	var wg sync.WaitGroup

	// Start the ticker
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.ticker(log)
	}()

	// Start the downloader
	wg.Add(1)
	go func() {
		defer wg.Done()
		d.downloader(log)
	}()

	// Lauch the downloader at startup
	go func() {
		log.Debug("initial downloader launch")
		d.event <- struct{}{}
	}()

	wg.Wait()

	log.Debug("downloader stopped")

	return nil
}

// Stop stops the downloader
func (d *Downloader) Stop(log *logrus.Entry) {
	close(d.done)
}

func (d *Downloader) ticker(log *logrus.Entry) {
	tick := time.Tick(d.config.Downloader.Timer)
	for {
		select {
		case <-tick:
			log.Debug("downloader timer triggered")
			d.event <- struct{}{}
		case <-d.done:
			log.Debug("downloader timer stopped")
			return
		}
	}
}

func (d *Downloader) downloader(log *logrus.Entry) {
	for {
		select {
		case <-d.event:
			log.Debug("downloader event")
			d.downloadMissingVideos(log)
		case <-d.done:
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

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		d.downloadMissingMovies(wl, log)
	}()
	go func() {
		defer wg.Done()
		d.downloadMissingShows(wl, log)
	}()
	wg.Wait()
}

func (d *Downloader) downloadMissingMovies(wl *polochon.Wishlist, log *logrus.Entry) {
	log = log.WithField("function", "download_movies")

	for _, wantedMovie := range wl.Movies {
		ok, err := d.videoStore.HasMovie(wantedMovie.ImdbID)
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
		log = log.WithField("imdb_id", m.ImdbID)

		if err := m.GetDetails(log); err != nil {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				continue
			}
		}

		log = log.WithField("title", m.Title)

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

		if err := d.config.Downloader.Client.Download(torrentURL, log); err != nil {
			log.Error(err)
			continue
		}
	}
}

func (d *Downloader) downloadMissingShows(wl *polochon.Wishlist, log *logrus.Entry) {
	log = log.WithField("function", "download_shows")

	for _, wishedShow := range wl.Shows {
		s := polochon.NewShow(d.config.Show)
		s.ImdbID = wishedShow.ImdbID
		log = log.WithField("imdb_id", s.ImdbID)

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

			if err := d.config.Downloader.Client.Download(torrentURL, log); err != nil {
				log.Error(err)
				continue
			}
		}
	}
}
