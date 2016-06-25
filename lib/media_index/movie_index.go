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

// MoviePath returns the movie path from its ID
func (mi *MovieIndex) MoviePath(imdbID string) (string, error) {
	mi.RLock()
	defer mi.RUnlock()

	// Check if the id is in the index and get the filePath
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
	if _, err := mi.MoviePath(m.ImdbID); err != nil {
		return err
	}

	mi.Lock()
	defer mi.Unlock()
	delete(mi.ids, m.ImdbID)

	return nil
}

// IDs returns the movie ids
func (mi *MovieIndex) IDs() []string {
	mi.RLock()
	defer mi.RUnlock()

	return extractAndSortStringMapKeys(mi.ids)
}

// Has searches the movie index for an ImdbID and returns true if the movie is
// indexed
func (mi *MovieIndex) Has(imdbID string) (bool, error) {
	mi.RLock()
	defer mi.RUnlock()

	_, err := mi.MoviePath(imdbID)
	switch err {
	case nil:
		return true, nil
	case ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}
