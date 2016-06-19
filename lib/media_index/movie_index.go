package index

import (
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

// MovieIndex is an index for the movies
type MovieIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// ids keep the imdb ids and their associated paths
	ids map[string]string
}

// NewMovieIndex returns a new movie index
func NewMovieIndex() *MovieIndex {
	return &MovieIndex{
		ids: map[string]string{},
	}
}

// Clear clears the movie index
func (mi *MovieIndex) Clear() {
	mi.Lock()
	defer mi.Unlock()

	mi.ids = map[string]string{}
}

// SearchByImdbID searches for a movie from its IMDB ID
func (mi *MovieIndex) SearchByImdbID(imdbID string) (string, error) {
	mi.RLock()
	defer mi.RUnlock()

	// Check if the id is in the index and get the filePath
	filePath, err := mi.searchMovieByImdbID(imdbID)
	if err != nil {
		return "", err
	}

	return filePath, nil
}

func (mi *MovieIndex) searchMovieByImdbID(imdbID string) (string, error) {
	filePath, ok := mi.ids[imdbID]
	if !ok {
		return "", ErrNotFound
	}

	return filePath, nil
}

// Add adds a movie to an index
func (mi *MovieIndex) Add(movie *polochon.Movie) error {
	mi.Lock()
	defer mi.Unlock()

	mi.ids[movie.ImdbID] = movie.Path

	return nil
}

// Remove will delete the movie from the index
func (mi *MovieIndex) Remove(m *polochon.Movie, log *logrus.Entry) error {
	mi.Lock()
	defer mi.Unlock()

	if _, ok := mi.ids[m.ImdbID]; !ok {
		log.Errorf("Movie not in ids index, WEIRD")
		return ErrNotFound
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
