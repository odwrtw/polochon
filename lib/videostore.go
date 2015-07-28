package polochon

import (
	"errors"
	"os"
	"path"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

// Video errors
var (
	ErrSlugNotFound = errors.New("videostore: no such file with this slug")
)

// VideoStore represent a collection of videos
type VideoStore struct {
	config *Config
	log    *logrus.Entry
}

// NewVideoStore returns a list of videos
func NewVideoStore(c *Config, log *logrus.Logger) *VideoStore {
	return &VideoStore{
		config: c,
		log:    log.WithField("function", "videoStore"),
	}
}

// ScanMovies scans the movie folder and returns a list of video
func (vs *VideoStore) ScanMovies() ([]*Movie, error) {
	movies := []*Movie{}

	// Walk movies
	err := filepath.Walk(vs.config.Video.Movie.Dir, func(filePath string, file os.FileInfo, err error) error {
		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for movie type
		ext := path.Ext(filePath)

		var movieFile *File
		for _, mext := range vs.config.File.VideoExtentions {
			if ext == mext {
				movieFile = NewFileWithConfig(filePath, vs.config)
				break
			}
		}

		if movieFile == nil {
			return nil
		}

		// load nfo
		nfoFile, err := os.Open(movieFile.NfoPath())
		if err != nil {
			vs.log.Errorf("video store: failed to open file %q", filePath)
			return nil
		}
		defer nfoFile.Close()

		movie, err := readMovieNFO(nfoFile, vs.config.Video.Movie)
		if err != nil {
			vs.log.Errorf("video store: failed to read movie NFO: %q", err)
			return nil
		}

		movie.SetFile(movieFile)
		movies = append(movies, movie)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return movies, nil
}

// ScanShows scans the show folder and returns a list of show with the linked episodes
func (vs *VideoStore) ScanShows() ([]*Show, error) {
	showStore := []*Show{}

	shows, err := vs.scanShows()
	if err != nil {
		return showStore, err
	}

	for showPath, show := range shows {
		episodes, err := vs.scanEpisodes(showPath)
		if err != nil {
			return showStore, err
		}

		show.Episodes = episodes

		showStore = append(showStore, show)
	}

	return showStore, nil
}

// scanShow returns a show with the path for its episodes
func (vs *VideoStore) scanShows() (map[string]*Show, error) {
	showPath := make(map[string]*Show)

	// Walk movies
	err := filepath.Walk(vs.config.Video.Show.Dir, func(filePath string, file os.FileInfo, err error) error {
		if file.Name() != "tvshow.nfo" {
			return nil
		}

		nfoFile, err := os.Open(filePath)
		if err != nil {
			vs.log.Errorf("video store: failed to open tv show NFO: %q", err)
			return nil
		}
		defer nfoFile.Close()

		show, err := readShowNFO(nfoFile, vs.config.Video.Show)
		if err != nil {
			vs.log.Errorf("video store: failed to read tv show NFO: %q", err)
			return nil
		}

		// Add the show to the map
		p, _ := filepath.Split(filePath)
		showPath[p] = show

		return nil
	})
	if err != nil {
		return nil, err
	}

	return showPath, nil
}

// scanEpisodes returns the show episodes in a path
func (vs *VideoStore) scanEpisodes(showPath string) ([]*ShowEpisode, error) {
	showEpisodes := []*ShowEpisode{}

	// Walk movies
	err := filepath.Walk(showPath, func(filePath string, file os.FileInfo, err error) error {
		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for movie type
		ext := path.Ext(filePath)

		var epFile *File
		for _, mext := range vs.config.File.VideoExtentions {
			if ext == mext {
				epFile = NewFileWithConfig(filePath, vs.config)
				break
			}
		}

		if epFile == nil {
			return nil
		}

		// load nfo
		nfoFile, err := os.Open(epFile.NfoPath())
		if err != nil {
			vs.log.Errorf("video store: failed to open file %q", filePath)
			return nil
		}
		defer nfoFile.Close()

		episode, err := readShowEpisodeNFO(nfoFile, vs.config.Video.Show)
		if err != nil {
			vs.log.Errorf("video store: failed to read episode NFO: %q", err)
			return nil
		}

		episode.SetFile(epFile)
		showEpisodes = append(showEpisodes, episode)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return showEpisodes, nil
}

// SearchMovieBySlug returns the video by its slug
func (vs *VideoStore) SearchMovieBySlug(slug string) (Video, error) {
	// Get all the movies
	movies, err := vs.ScanMovies()
	if err != nil {
		return nil, err
	}

	// Look for movies with this Slug
	for _, m := range movies {
		if m.Slug() == slug {
			return m, nil
		}
	}

	return nil, ErrSlugNotFound
}

// SearchShowEpisodeBySlug search for a show episode by its slug
func (vs *VideoStore) SearchShowEpisodeBySlug(slug string) (Video, error) {
	shows, err := vs.ScanShows()
	if err != nil {
		return nil, err
	}

	// Look for shows with this Slug
	for _, s := range shows {
		for _, e := range s.Episodes {
			if e.Slug() == slug {
				return e, nil
			}
		}
	}

	return nil, ErrSlugNotFound
}
