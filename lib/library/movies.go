package library

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
	"github.com/sirupsen/logrus"
)

// MovieIDs returns the movie ids
func (l *Library) MovieIDs() []string {
	return l.movieIndex.IDs()
}

// MovieIndex returns the movie ids
func (l *Library) MovieIndex() map[string]*index.Movie {
	return l.movieIndex.Index()
}

// HasMovie returns true if the movie is in the store
func (l *Library) HasMovie(imdbID string) (bool, error) {
	return l.movieIndex.Has(imdbID)
}

func (l *Library) getMovieDir(movie *polochon.Movie) string {
	title := movie.Title
	if movie.Year != 0 {
		title += fmt.Sprintf(" (%d)", movie.Year)
	}
	title = strings.ReplaceAll(title, "/", "-")

	return filepath.Join(l.MovieDir, title)
}

// AddMovie adds a movie to the store
func (l *Library) AddMovie(movie *polochon.Movie, logEntry *logrus.Entry) error {
	log := logEntry.WithFields(logrus.Fields{
		"type":    "movie",
		"title":   movie.Title,
		"imdb_id": movie.ImdbID,
	})

	log.Debug("adding movie to the library")

	if movie.Path == "" {
		return ErrMissingMovieFilePath
	}

	// Check if the movie is already in the library
	ok, err := l.HasMovie(movie.ImdbID)
	if err != nil {
		return err
	}

	if ok {
		// Get the old movie path from the index
		oldMovie, err := l.GetMovie(movie.ImdbID)
		if err != nil {
			return err
		}

		// Delete it
		if err := l.DeleteMovie(oldMovie, log); err != nil {
			return err
		}
	}

	storePath := l.getMovieDir(movie)

	// If the movie already in the right dir there is nothing to do
	if path.Dir(movie.Path) == storePath {
		log.Debug("movie already in the destination folder")
		return nil
	}

	// Remove movie dir if it exisits
	if ok := exists(storePath); ok {
		log.Debug("movie folder exists, remove it")
		if err := os.RemoveAll(storePath); err != nil {
			return err
		}
	}

	// Create the folder
	if err := os.Mkdir(storePath, os.ModePerm); err != nil {
		return err
	}

	// Move the movie into the folder
	newPath := filepath.Join(storePath, path.Base(movie.Path))

	// Save the old path
	oldPath := movie.Path

	log.WithFields(logrus.Fields{
		"old_path": movie.Path,
		"new_path": newPath,
	}).Debugf("moving movie file")
	if err := MoveFile(movie.Path, newPath); err != nil {
		return err
	}

	// Set the new movie path
	movie.Path = newPath

	// Create a symlink between the new and the old location
	// Only if the downloader is enabled
	if l.downloaderConfig.Enabled {
		log.Debugf("creating symlink with the old path")
		if err := os.Symlink(movie.Path, oldPath); err != nil {
			log.Warnf("error while making symlink between %s and %s : %+v", oldPath, movie.Path, err)
		}
	}

	// Write NFO into the file
	if err := writeNFOFile(movie.NfoPath(), movie); err != nil {
		return err
	}

	// Save the subtitles here
	if err := l.SaveSubtitles(movie, log); err != nil {
		return err
	}

	if movie.Fanart == "" || movie.Thumb == "" {
		return ErrMissingMovieImageURL
	}

	// Download images
	for _, img := range []struct {
		name     string
		url      string
		savePath string
	}{
		{
			name:     "fanart",
			url:      movie.Fanart,
			savePath: movie.MovieFanartPath(),
		},
		{
			name:     "thumb",
			url:      movie.Thumb,
			savePath: movie.MovieThumbPath(),
		},
	} {
		log.Debug("downloading " + img.name)
		if err := download(img.url, img.savePath); err != nil {
			return err
		}
	}

	// At this point the video is stored
	if err := l.movieIndex.Add(movie); err != nil {
		return err
	}

	return nil
}

// GetMovie returns the video by its imdb ID
func (l *Library) GetMovie(imdbID string) (*polochon.Movie, error) {
	movieIndex, err := l.movieIndex.Movie(imdbID)
	if err != nil {
		return nil, err
	}
	return l.newMovieFromPath(movieIndex.Path)
}

// DeleteMovie will delete the movie
func (l *Library) DeleteMovie(m *polochon.Movie, log *logrus.Entry) error {
	// Delete the movie
	d := filepath.Dir(m.Path)
	log.Infof("removing movie")

	if err := os.RemoveAll(d); err != nil {
		return err
	}
	// Remove the movie from the index
	return l.movieIndex.Remove(m, log)
}

// NewMovieFromPath returns a new Movie from its path
func (l *Library) newMovieFromPath(path string) (*polochon.Movie, error) {
	file := polochon.NewFile(path)
	m := polochon.NewMovieFromFile(l.movieConfig, *file)

	if err := readNFOFile(file.NfoPath(), m); err != nil {
		return nil, err
	}

	l.UpdateSubtitles(m)
	return m, nil
}

// GetIndexedMovie returns a Movie index from its id
func (l *Library) GetIndexedMovie(id string) (*index.Movie, error) {
	m, err := l.movieIndex.Movie(id)
	if err != nil {
		return nil, err
	}

	return m, nil
}
