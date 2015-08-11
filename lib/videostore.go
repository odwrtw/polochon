package polochon

import (
	"errors"
	"sync"

	"github.com/Sirupsen/logrus"
)

// Video errors
var (
	ErrSlugNotFound          = errors.New("videostore: no such file with this slug")
	ErrInvalidIndexVideoType = errors.New("videostore: invalid index video type")
)

// VideoStore represent a collection of videos
type VideoStore struct {
	movieIndex *MovieIndex
	showIndex  *ShowIndex
	config     *Config
	log        *logrus.Entry
}

// NewVideoStore returns a list of videos
func NewVideoStore(c *Config, log *logrus.Logger) *VideoStore {
	videoStoreLogger := log.WithField("function", "videoStore")
	return &VideoStore{
		config:     c,
		log:        videoStoreLogger,
		movieIndex: NewMovieIndex(c, videoStoreLogger),
		showIndex:  NewShowIndex(c, videoStoreLogger),
	}
}

// MovieIds returns the movie ids
func (vs *VideoStore) MovieIds() ([]string, error) {
	return vs.movieIndex.MovieIds()
}

// MovieSlugs returns the movie slugs
func (vs *VideoStore) MovieSlugs() ([]string, error) {
	return vs.movieIndex.MovieSlugs()
}

// HasMovie returns true if the movie is in the store
func (vs *VideoStore) HasMovie(imdbID string) (bool, error) {
	return vs.movieIndex.Has(imdbID)
}

// ShowIds returns the show ids, seasons and episodes
func (vs *VideoStore) ShowIds() (map[string]map[int]map[int]string, error) {
	return vs.showIndex.ShowIds()
}

// ShowSlugs returns the show slugs
func (vs *VideoStore) ShowSlugs() ([]string, error) {
	return vs.showIndex.ShowSlugs()
}

// HasShow returns true if the show is in the store
func (vs *VideoStore) HasShow(imdbID string, season, episode int) (bool, error) {
	return vs.showIndex.Has(imdbID, season, episode)
}

// SearchMovieBySlug returns the video by its slug
func (vs *VideoStore) SearchMovieBySlug(slug string) (Video, error) {
	return vs.movieIndex.SearchMovieBySlug(slug)
}

// SearchShowEpisodeBySlug search for a show episode by its slug
func (vs *VideoStore) SearchShowEpisodeBySlug(slug string) (Video, error) {
	return vs.showIndex.SearchShowEpisodeBySlug(slug)
}

// RebuildIndex rebuilds both the movie and show index
func (vs *VideoStore) RebuildIndex() error {
	// Create a goroutine for each index
	var wg sync.WaitGroup
	errc := make(chan error, 2)
	wg.Add(2)

	// Build the movie index
	go func() {
		defer wg.Done()
		if err := vs.movieIndex.Rebuild(); err != nil {
			errc <- err
		}
	}()

	// Build the show index
	go func() {
		defer wg.Done()
		if err := vs.showIndex.Rebuild(); err != nil {
			errc <- err
		}
	}()

	// Wait for them to be done
	wg.Wait()
	close(errc)

	// Return the first error found
	if len(errc) > 0 {
		return <-errc
	}

	return nil
}

// RebuildVideoIndex rebuilds the movie or show index
func (vs *VideoStore) RebuildVideoIndex(v Video) error {
	switch v.Type() {
	case MovieType:
		return vs.movieIndex.Rebuild()
	case ShowEpisodeType:
		return vs.showIndex.Rebuild()
	default:
		return ErrInvalidIndexVideoType
	}
}
