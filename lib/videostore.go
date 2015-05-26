package polochon

import (
	"log"
	"os"
	"path"
	"path/filepath"
)

// VideoStore represent a collection of videos
type VideoStore struct {
	Videos []Video
	config *Config
}

// NewVideoStore returns a list of videos
func NewVideoStore(c *Config) *VideoStore {
	return &VideoStore{
		config: c,
	}
}

// Scan scans movies and shows dir and update the video store
func (vs *VideoStore) Scan() error {
	// Reset video store
	vs.Videos = []Video{}

	// Scan movies
	movies, err := vs.scanMovies()
	if err != nil {
		return err
	}
	vs.Videos = append(vs.Videos, movies...)

	return nil
}

func (vs *VideoStore) scanMovies() ([]Video, error) {
	movies := []Video{}

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
			log.Printf("video store: failed to open file %q", filePath)
			return nil
		}

		video, err := readMovieNFO(nfoFile)
		if err != nil {
			log.Printf("video store: failed to read movie NFO: %q", err)
			return nil
		}

		movies = append(movies, video)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return movies, nil
}
