package polochon

import (
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/resync"
)

// MovieIndex is an index for the movies
type MovieIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.Mutex
	// Build the index only once, use resync to add the Reset capability
	once resync.Once
	// Config
	config *Config
	// Logger
	log *logrus.Entry
	// ids keep the imdb ids and their associated paths
	ids map[string]string
	// slugs keep the movie index by slug
	slugs map[string]string
}

// NewMovieIndex returns a new movie index
func NewMovieIndex(config *Config, log *logrus.Entry) *MovieIndex {
	return &MovieIndex{
		config: config,
		log:    log.WithField("function", "movieIndex"),
		ids:    map[string]string{},
		slugs:  map[string]string{},
	}
}

// SearchMovieBySlug searches for a movie from its slug
func (mi *MovieIndex) SearchMovieBySlug(slug string) (Video, error) {
	if err := mi.index(); err != nil {
		return nil, err
	}

	// Check if the slug is in the index and get the filePath
	filePath, err := mi.searchMovieBySlug(slug)
	if err != nil {
		return nil, err
	}

	// Create a File from the path
	file := NewFileWithConfig(filePath, mi.config)

	// Open the NFO
	nfoFile, err := os.Open(file.NfoPath())
	if err != nil {
		return nil, err
	}
	defer nfoFile.Close()

	// Unmarshal the NFO into an episode
	movie, err := readMovieNFO(nfoFile, mi.config.Video.Movie)
	if err != nil {
		return nil, err
	}

	movie.SetFile(file)
	// Set logger
	movie.log = mi.log.WithFields(logrus.Fields{
		"type": "movie",
	})

	return movie, nil
}

// index builds the movie index only once
func (mi *MovieIndex) index() error {
	var err error
	mi.once.Do(func() {
		err = buildMovieIndex(mi)
	})
	return err
}

// Rebuild rebuilds the movie index
func (mi *MovieIndex) Rebuild() error {
	mi.once.Reset()
	return mi.index()
}

// Has searches the movie index for an ImdbID and returns true if the movie is
// indexed
func (mi *MovieIndex) Has(imdbID string) (bool, error) {
	if err := mi.index(); err != nil {
		return false, err
	}

	filePath, err := mi.searchMovieByImdbID(imdbID)
	if filePath != "" && err == nil {
		return true, nil
	}

	return false, nil
}

// Function to be overwritten during the tests
var buildMovieIndex = func(mi *MovieIndex) error {
	return buildIndex(mi)
}

// buildMovieIndex is the function to populate the movie index
func buildIndex(mi *MovieIndex) error {
	// Keep track of the time to build the index
	start := time.Now()
	mi.log.Info("Building movie index")

	// Reset the previous values
	mi.ids = map[string]string{}
	mi.slugs = map[string]string{}

	// Walk movies
	err := filepath.Walk(mi.config.Video.Movie.Dir, func(filePath string, file os.FileInfo, err error) error {
		// Check err
		if err != nil {
			mi.log.Errorf("video store: failed to walk %q", err)
			return nil
		}

		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for movie type
		ext := path.Ext(filePath)

		var movieFile *File
		for _, mext := range mi.config.File.VideoExtentions {
			if ext == mext {
				movieFile = NewFileWithConfig(filePath, mi.config)
				break
			}
		}

		if movieFile == nil {
			return nil
		}

		// load nfo
		nfoFile, err := os.Open(movieFile.NfoPath())
		if err != nil {
			mi.log.Errorf("video store: failed to open file %q", filePath)
			return nil
		}
		defer nfoFile.Close()

		// Read the movie informations
		movie, err := readMovieNFO(nfoFile, mi.config.Video.Movie)
		if err != nil {
			mi.log.Errorf("video store: failed to read movie NFO: %q", err)
			return nil
		}
		movie.SetFile(movieFile)

		// Add the movie to the index
		mi.AddToIndex(movie)

		return nil
	})
	if err != nil {
		// If an error occurs we should be able to rebuild the index
		mi.once.Reset()
		return err
	}

	mi.log.Infof("Index built in %s", time.Since(start))

	return nil
}

// AddToIndex adds a movie to an index
func (mi *MovieIndex) AddToIndex(movie *Movie) error {
	mi.Lock()
	defer mi.Unlock()

	mi.slugs[movie.Slug()] = movie.Path
	mi.ids[movie.ImdbID] = movie.Path

	return nil
}

// RemoveFromIndex will delete the movie from the index
func (mi *MovieIndex) RemoveFromIndex(m *Movie) error {
	mi.Lock()
	defer mi.Unlock()

	slug := m.Slug()

	if _, ok := mi.slugs[slug]; !ok {
		mi.log.Errorf("Movie not in slug index, WEIRD")
		return ErrSlugNotFound
	}
	delete(mi.slugs, slug)

	if _, ok := mi.ids[m.ImdbID]; !ok {
		mi.log.Errorf("Movie not in ids index, WEIRD")
		return ErrSlugNotFound
	}
	delete(mi.ids, m.ImdbID)

	return nil
}

// MovieIds returns the movie ids
func (mi *MovieIndex) MovieIds() ([]string, error) {
	if err := mi.index(); err != nil {
		return []string{}, err
	}

	mi.Lock()
	defer mi.Unlock()

	return extractMapKeys(mi.ids)
}

// MovieSlugs returns the movie slugs
func (mi *MovieIndex) MovieSlugs() ([]string, error) {
	if err := mi.index(); err != nil {
		return []string{}, err
	}

	mi.Lock()
	defer mi.Unlock()

	return extractMapKeys(mi.slugs)
}

// searchMovieBySlug searches for a movie from its slug
func (mi *MovieIndex) searchMovieBySlug(slug string) (string, error) {
	mi.Lock()
	defer mi.Unlock()

	filePath, ok := mi.slugs[slug]
	if !ok {
		return "", ErrSlugNotFound
	}

	return filePath, nil
}

// searchMovieByImdbID searches for a movie from its imdbId
func (mi *MovieIndex) searchMovieByImdbID(imdbID string) (string, error) {
	mi.Lock()
	defer mi.Unlock()

	filePath, ok := mi.ids[imdbID]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	return filePath, nil
}
