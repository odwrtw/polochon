package index

import (
	"sync"

	polochon "github.com/odwrtw/polochon/lib"
)

// MovieIndex is an index for the movies
type MovieIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// ids keep the imdb ids and their associated movie
	ids map[string]*polochon.Movie
}

// NewMovieIndex returns a new movie index
func NewMovieIndex() *MovieIndex {
	return &MovieIndex{
		ids: map[string]*polochon.Movie{},
	}
}

// Clear clears the movie index
func (mi *MovieIndex) Clear() {
	mi.Lock()
	defer mi.Unlock()

	mi.ids = map[string]*polochon.Movie{}
}

// Movie returns the movie from its ID
func (mi *MovieIndex) Movie(imdbID string) (*polochon.Movie, error) {
	mi.RLock()
	defer mi.RUnlock()

	movie, ok := mi.ids[imdbID]
	if !ok {
		return nil, ErrNotFound
	}

	return movie, nil
}

// Add adds a movie to the index. The movie must already have its sidecar file
// fields (FanartFile, ThumbFile, NFOFile) populated before calling Add.
func (mi *MovieIndex) Add(movie *polochon.Movie) error {
	if movie.Name == "" {
		movie.Name = movie.Filename()
	}

	mi.Lock()
	mi.ids[movie.ImdbID] = movie
	mi.Unlock()

	return nil
}

// UpsertSubtitle updates or inserts a subtitle in the index
func (mi *MovieIndex) UpsertSubtitle(m *polochon.Movie, s *polochon.Subtitle) error {
	movie, err := mi.Movie(m.ImdbID)
	if err != nil {
		return err
	}

	mi.Lock()
	movie.Subtitles = upsertSubtitle(movie.Subtitles, s)
	mi.Unlock()

	return nil
}

// Remove deletes the movie from the index
func (mi *MovieIndex) Remove(m *polochon.Movie) error {
	if _, err := mi.Movie(m.ImdbID); err != nil {
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

// Index returns the movie index
func (mi *MovieIndex) Index() map[string]*polochon.Movie {
	mi.RLock()
	defer mi.RUnlock()

	return mi.ids
}

// Has searches the movie index for an ImdbID and returns true if the movie is indexed
func (mi *MovieIndex) Has(imdbID string) (bool, error) {
	mi.RLock()
	defer mi.RUnlock()

	_, err := mi.Movie(imdbID)
	switch err {
	case nil:
		return true, nil
	case ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

// HasSubtitle searches the movie index for a subtitle in language lang
func (mi *MovieIndex) HasSubtitle(imdbID string, sub *polochon.Subtitle) (bool, error) {
	movie, err := mi.Movie(imdbID)
	if err != nil {
		return false, err
	}

	for _, s := range movie.Subtitles {
		if sub.Lang == s.Lang {
			return true, nil
		}
	}

	return false, nil
}
