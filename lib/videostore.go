package polochon

import (
	"errors"
	"sync"

	"github.com/Sirupsen/logrus"
)

// Video errors
var (
	ErrSlugNotFound          = errors.New("videostore: no such file with this slug")
	ErrImdbIDNotFound        = errors.New("videostore: no such file with this imdbID")
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

// HasVideo returns true if the video is in the store
func (vs *VideoStore) HasVideo(video Video) (bool, error) {
	switch v := video.(type) {
	case *Movie:
		return vs.HasMovie(v.ImdbID)
	case *ShowEpisode:
		return vs.HasShow(v.ShowImdbID, v.Season, v.Episode)
	default:
		return false, ErrInvalidIndexVideoType
	}
}

// SearchBySlug search the video by its slug
func (vs *VideoStore) SearchBySlug(video Video) (Video, error) {
	switch v := video.(type) {
	case *Movie:
		return vs.SearchMovieBySlug(v.Slug())
	case *ShowEpisode:
		return vs.SearchShowEpisodeBySlug(v.Slug())
	default:
		return nil, ErrInvalidIndexVideoType
	}
}

// Delete will delete the video
func (vs *VideoStore) Delete(video Video) error {
	switch v := video.(type) {
	case *Movie:
		return vs.DeleteMovie(v)
	case *ShowEpisode:
		return vs.DeleteShowEpisode(v)
	default:
		return ErrInvalidIndexVideoType
	}
}

// DeleteMovie will delete the movie
func (vs *VideoStore) DeleteMovie(m *Movie) error {
	// Delete the movie
	if err := m.Delete(); err != nil {
		vs.log.Errorf("Error while deleting movie :%q", err)
		return err
	}
	// Remove the movie from the index
	if err := vs.movieIndex.RemoveFromIndex(m); err != nil {
		vs.log.Errorf("Error while deleting movie from index :%q", err)
		return err
	}

	return nil
}

// DeleteShowEpisode will delete the showEpisode
func (vs *VideoStore) DeleteShowEpisode(se *ShowEpisode) error {
	// Delete the episode
	if err := se.Delete(); err != nil {
		vs.log.Errorf("Error while deleting episode :%q", err)
		return err
	}
	// Remove the episode from the index
	if err := vs.showIndex.RemoveFromIndex(se); err != nil {
		vs.log.Errorf("Error while deleting episode from index :%q", err)
		return err
	}

	// Season is empty, delete the whole season
	ok, err := vs.showIndex.isSeasonEmpty(se.ShowImdbID, se.Season)
	if err != nil {
		return err
	}
	if ok {
		// Delete the whole season
		if err := se.DeleteSeason(); err != nil {
			vs.log.Errorf("Error while deleting season :%q", err)
			return err
		}
		// Remove the season from the index
		if err := vs.showIndex.RemoveSeasonFromIndex(se.Show, se.Season); err != nil {
			vs.log.Errorf("Error while deleting season from index :%q", err)
			return err
		}
	}

	// Show is empty, delete the whole show from the index
	ok, err = vs.showIndex.isShowEmpty(se.ShowImdbID)
	if err != nil {
		return err
	}
	if ok {
		// Delete the whole Show
		if err := se.Show.Delete(); err != nil {
			vs.log.Errorf("Error while deleting show :%q", err)
			return err
		}
		// Remove the show from the index
		if err := vs.showIndex.RemoveShowFromIndex(se.Show); err != nil {
			vs.log.Errorf("Error while deleting show from index :%q", err)
			return err
		}
	}

	return nil
}

// AddToIndex adds a video to the index
func (vs *VideoStore) AddToIndex(video Video) error {
	switch v := video.(type) {
	case *Movie:
		return vs.movieIndex.AddToIndex(v)
	case *ShowEpisode:
		return vs.showIndex.AddToIndex(v)
	default:
		return ErrInvalidIndexVideoType
	}
}

// HasMovie returns true if the movie is in the store
func (vs *VideoStore) HasMovie(imdbID string) (bool, error) {
	return vs.movieIndex.Has(imdbID)
}

// HasShow returns true if the show is in the store
func (vs *VideoStore) HasShow(imdbID string, season, episode int) (bool, error) {
	return vs.showIndex.Has(imdbID, season, episode)
}

// ShowIds returns the show ids, seasons and episodes
func (vs *VideoStore) ShowIds() (map[string]map[int]map[int]string, error) {
	return vs.showIndex.ShowIds()
}

// ShowSlugs returns the show slugs
func (vs *VideoStore) ShowSlugs() ([]string, error) {
	return vs.showIndex.ShowSlugs()
}

// HasShowEpisode returns true if the show episode is in the store
func (vs *VideoStore) HasShowEpisode(imdbID string, season, episode int) (bool, error) {
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

// SearchMovieByImdbID returns the video by its imdb ID
func (vs *VideoStore) SearchMovieByImdbID(imdbID string) (Video, error) {
	return vs.movieIndex.SearchMovieByImdbID(imdbID)
}

// SearchShowEpisodeByImdbID search for a show episode by its imdb ID
func (vs *VideoStore) SearchShowEpisodeByImdbID(imdbID string) (Video, error) {
	return vs.showIndex.SearchShowEpisodeByImdbID(imdbID)
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
	err, ok := <-errc
	if ok {
		return err
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
