package index

import (
	"fmt"
	"sync"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// MovieIndex is an index for the movies
type MovieIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// ids keep the imdb ids and their associated infos
	ids map[string]*Movie
}

// Movie represents a Movie in the index
type Movie struct {
	polochon.VideoMetadata
	Path      string      `json:"-"`
	Filename  string      `json:"filename"`
	Title     string      `json:"title"`
	Year      int         `json:"year"`
	Size      int64       `json:"size"`
	Subtitles []*Subtitle `json:"subtitles"`
}

// NewMovieIndex returns a new movie index
func NewMovieIndex() *MovieIndex {
	return &MovieIndex{
		ids: map[string]*Movie{},
	}
}

// Clear clears the movie index
func (mi *MovieIndex) Clear() {
	mi.Lock()
	defer mi.Unlock()

	mi.ids = map[string]*Movie{}
}

// Movie returns the movie index from its ID
func (mi *MovieIndex) Movie(imdbID string) (*Movie, error) {
	mi.RLock()
	defer mi.RUnlock()

	// Check if the id is in the index and get the filePath
	movie, ok := mi.ids[imdbID]
	if !ok {
		return nil, ErrNotFound
	}

	return movie, nil
}

// Add adds a movie to an index
func (mi *MovieIndex) Add(movie *polochon.Movie) error {
	mi.Lock()
	defer mi.Unlock()

	mi.ids[movie.ImdbID] = &Movie{
		Path:          movie.Path,
		Filename:      movie.Filename(),
		Title:         movie.Title,
		Year:          movie.Year,
		Size:          movie.Size,
		VideoMetadata: movie.VideoMetadata,
	}

	return nil
}

// AddSubtitle adds a movie subtitle to an index
func (mi *MovieIndex) AddSubtitle(movie *polochon.Movie, sub *polochon.Subtitle) error {
	// Check that we have the movie
	has, err := mi.Has(movie.ImdbID)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("failed to add subtitle : movie %s not indexed", movie.ImdbID)
	}

	mi.Lock()
	defer mi.Unlock()

	// Append the subtitle to the index
	mi.ids[movie.ImdbID].Subtitles = append(
		mi.ids[movie.ImdbID].Subtitles,
		NewSubtitle(sub),
	)
	return nil
}

// Remove will delete the movie from the index
func (mi *MovieIndex) Remove(m *polochon.Movie, log *logrus.Entry) error {
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

// Index returns the movie index to be rendered
func (mi *MovieIndex) Index() map[string]*Movie {
	mi.RLock()
	defer mi.RUnlock()

	return mi.ids
}

// Has searches the movie index for an ImdbID and returns true if the movie is
// indexed
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

// HasSubtitle searches the movie index for a subtitle in language lang and
// ImdbID and returns true if the subtitle is present
func (mi *MovieIndex) HasSubtitle(imdbID string, sub *polochon.Subtitle) (bool, error) {
	movie, err := mi.Movie(imdbID)
	if err != nil {
		if err == ErrNotFound {
			err = nil
		}
		return false, err
	}

	for _, s := range movie.Subtitles {
		if sub.Lang == s.Lang {
			return true, nil
		}
	}

	return false, nil
}
