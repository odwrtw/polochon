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

	// Check if the slug is in the index
	mi.Lock()
	filePath, ok := mi.slugs[slug]
	if !ok {
		return nil, ErrSlugNotFound
	}
	mi.Unlock()

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

	return movie, nil
}

// index builds the movie index only once
func (mi *MovieIndex) index() error {
	var err error
	mi.once.Do(func() {
		err = mi.buildMovieIndex()
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

	mi.Lock()
	defer mi.Unlock()

	if _, ok := mi.ids[imdbID]; ok {
		return true, nil
	}

	return false, nil
}

// buildMovieIndex is the function to populate the movie index
func (mi *MovieIndex) buildMovieIndex() error {
	// Keep track of the time to build the index
	start := time.Now()
	mi.log.Info("Building movie index")

	// Reset the previous values
	mi.ids = map[string]string{}
	mi.slugs = map[string]string{}

	// Walk movies
	err := filepath.Walk(mi.config.Video.Movie.Dir, func(filePath string, file os.FileInfo, err error) error {
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

		// Add the movie to the index
		mi.slugs[movie.Slug()] = filePath
		mi.ids[movie.ImdbID] = filePath

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
