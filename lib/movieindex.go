package polochon

import (
	"sync"

	"github.com/Sirupsen/logrus"
)

// MovieIndex is an index for the movies
type MovieIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// ids keep the imdb ids and their associated paths
	ids map[string]string
	// slugs keep the movie index by slug
	slugs map[string]string
}

// NewMovieIndex returns a new movie index
func NewMovieIndex() *MovieIndex {
	return &MovieIndex{
		ids:   map[string]string{},
		slugs: map[string]string{},
	}
}

// Clear clears the movie index
func (mi *MovieIndex) Clear() {
	mi.Lock()
	defer mi.Unlock()

	mi.ids = map[string]string{}
	mi.slugs = map[string]string{}
}

// SearchBySlug searches for a movie from its slug
func (mi *MovieIndex) SearchBySlug(slug string) (string, error) {
	mi.RLock()
	defer mi.RUnlock()

	// Check if the slug is in the index and get the filePath
	filePath, err := mi.searchMovieBySlug(slug)
	if err != nil {
		return "", err
	}
	return filePath, nil

}

// SearchByImdbID searches for a movie from its slug
func (mi *MovieIndex) SearchByImdbID(imdbID string) (string, error) {
	mi.RLock()
	defer mi.RUnlock()

	// Check if the slug is in the index and get the filePath
	filePath, err := mi.searchMovieByImdbID(imdbID)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func (mi *MovieIndex) searchMovieBySlug(slug string) (string, error) {
	filePath, ok := mi.slugs[slug]
	if !ok {
		return "", ErrSlugNotFound
	}

	return filePath, nil
}

func (mi *MovieIndex) searchMovieByImdbID(imdbID string) (string, error) {
	filePath, ok := mi.ids[imdbID]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	return filePath, nil
}

// Add adds a movie to an index
func (mi *MovieIndex) Add(movie *Movie) error {
	mi.Lock()
	defer mi.Unlock()

	mi.slugs[movie.Slug()] = movie.Path
	mi.ids[movie.ImdbID] = movie.Path

	return nil
}

// Remove will delete the movie from the index
func (mi *MovieIndex) Remove(m *Movie, log *logrus.Entry) error {
	mi.Lock()
	defer mi.Unlock()

	slug := m.Slug()

	if _, ok := mi.slugs[slug]; !ok {
		log.Errorf("Movie not in slug index, WEIRD")
		return ErrSlugNotFound
	}
	delete(mi.slugs, slug)

	if _, ok := mi.ids[m.ImdbID]; !ok {
		log.Errorf("Movie not in ids index, WEIRD")
		return ErrSlugNotFound
	}
	delete(mi.ids, m.ImdbID)

	return nil
}

// IDs returns the movie ids
func (mi *MovieIndex) IDs() ([]string, error) {
	mi.RLock()
	defer mi.RUnlock()

	return extractMapKeys(mi.ids)
}

// Slugs returns the movie slugs
func (mi *MovieIndex) Slugs() ([]string, error) {
	mi.RLock()
	defer mi.RUnlock()

	return extractMapKeys(mi.slugs)
}

// Has searches the movie index for an ImdbID and returns true if the movie is
// indexed
func (mi *MovieIndex) Has(imdbID string) (bool, error) {
	mi.RLock()
	defer mi.RUnlock()
	filePath, err := mi.searchMovieByImdbID(imdbID)
	if filePath != "" && err == nil {
		return true, nil
	}

	return false, nil
}
