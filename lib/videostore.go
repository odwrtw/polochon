package polochon

import (
	"os"
	"path"
	"path/filepath"

	"github.com/Sirupsen/logrus"
)

// VideoStore represent a collection of videos
type VideoStore struct {
	config *Config
	log    *logrus.Entry
}

// NewVideoStore returns a list of videos
func NewVideoStore(c *Config) *VideoStore {
	return &VideoStore{
		config: c,
		log:    c.Log.WithField("function", "videoStore"),
	}
}

// ScanMovies scans the movie folder and returns a list of video
func (vs *VideoStore) ScanMovies() ([]*Movie, error) {
	movies := []*Movie{}

	// Walk movies
	err := filepath.Walk(vs.config.Movie.Dir, func(filePath string, file os.FileInfo, err error) error {
		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// Only check nfo files
		if ext := path.Ext(filePath); ext != ".nfo" {
			return nil
		}

		// Read the nfo
		nfoFile, err := os.Open(filePath)
		if err != nil {
			vs.log.Errorf("video store: failed to open file %q", filePath)
			return nil
		}

		movie, err := readMovieNFO(nfoFile)
		if err != nil {
			vs.log.Errorf("video store: failed to read movie NFO: %q", err)
			return nil
		}

		var movieFile File
		basePath := RemoveExt(filePath)
		//Get related movie file path
		for _, ext := range vs.config.Video.VideoExtentions {
			if _, err := os.Stat(basePath + ext); err == nil {
				movieFile = File{Path: basePath + ext, config: vs.config}
			}
		}

		if movieFile.Path == "" {
			vs.log.Errorf("video store: can't find movie file for NFO: %q", filePath)
			return nil
		}
		movie.File = &movieFile

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
	err := filepath.Walk(vs.config.Show.Dir, func(filePath string, file os.FileInfo, err error) error {
		if file.Name() != "tvshow.nfo" {
			return nil
		}

		nfoFile, err := os.Open(filePath)
		if err != nil {
			vs.log.Errorf("video store: failed to open tv show NFO: %q", err)
			return nil
		}

		show, err := readShowNFO(nfoFile)
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

		// Only check nfo files
		if ext := path.Ext(filePath); ext != ".nfo" {
			return nil
		}

		if file.Name() == "tvshow.nfo" {
			return nil
		}

		nfoFile, err := os.Open(filePath)
		if err != nil {
			vs.log.Errorf("video store: failed to open tv show episode NFO: %q", err)
			return nil
		}

		showEpisode, err := readShowEpisodeNFO(nfoFile)
		if err != nil {
			vs.log.Errorf("video store: failed to read tv show episode NFO: %q", err)
			return nil
		}

		showEpisodes = append(showEpisodes, showEpisode)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return showEpisodes, nil
}
