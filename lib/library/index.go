package library

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// RebuildIndex rebuilds both the movie and show index
func (l *Library) RebuildIndex(log *logrus.Entry) error {
	var wg sync.WaitGroup
	errc := make(chan error, 2)
	wg.Add(2)

	// Build the movie index
	go func() {
		defer wg.Done()
		if err := l.buildMovieIndex(log); err != nil {
			errc <- err
		}
	}()

	// Build the show index
	go func() {
		defer wg.Done()
		if err := l.buildShowIndex(log); err != nil {
			errc <- err
		}
	}()

	// Wait for them to be done
	wg.Wait()
	close(errc)

	// Return the first error found
	err, ok := <-errc
	if ok {
		return err
	}

	return nil
}

func (l *Library) buildMovieIndex(log *logrus.Entry) error {
	start := time.Now()
	defer func() {
		log.Infof("movie index built in %s", time.Since(start))
	}()
	l.movieIndex.Clear()

	root, err := os.Open(l.MovieDir)
	if err != nil {
		return err
	}
	defer root.Close()

	dirs, err := root.Readdirnames(-1)
	if err != nil {
		return err
	}

	reg := regexp.MustCompile(`.*\(\d{4}\)$`)

	for _, d := range dirs {
		if !reg.MatchString(d) {
			log.WithField("dir", d).Warn("invalid movie dir")
			continue
		}

		if err := l.buildFromMovieDir(d); err != nil {
			log.WithField("dir", d).Error(err)
		}
	}

	return nil
}

func (l *Library) buildFromMovieDir(d string) error {
	movieDir := filepath.Join(l.MovieDir, d)

	dir, err := os.Open(movieDir)
	if err != nil {
		return fmt.Errorf("failed to read movie dir %w", err)
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	var moviePath string
	for _, file := range files {
		if l.fileConfig.IsVideo(file) {
			moviePath = filepath.Join(movieDir, file)
			break
		}
	}

	if moviePath == "" {
		return fmt.Errorf("no video file found")
	}

	// Read the movie informations
	movie, err := l.newMovieFromPath(moviePath)
	if err != nil {
		return fmt.Errorf("library: failed to read movie NFO: %w", err)
	}

	return l.movieIndex.Add(movie)
}

func (l *Library) buildShowIndex(log *logrus.Entry) error {
	start := time.Now()
	defer func() {
		log.Infof("show index built in %s", time.Since(start))
	}()

	l.showIndex.Clear()

	root, err := os.Open(l.ShowDir)
	if err != nil {
		return err
	}
	defer root.Close()

	dirs, err := root.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, d := range dirs {
		showDir := filepath.Join(l.ShowDir, d)
		nfoPath := l.showNFOPath(showDir)

		show, err := l.newShowFromPath(nfoPath)
		if err != nil {
			log.Errorf("library: failed to read tv show NFO: %q", err)
			continue
		}

		if err := l.buildFromShowDir(show.ImdbID, showDir, log); err != nil {
			log.WithField("dir", d).Error(err)
		}
	}

	return nil
}

func (l *Library) buildFromShowDir(imdbID, showDir string, log *logrus.Entry) error {
	dir, err := os.Open(showDir)
	if err != nil {
		return fmt.Errorf("failed to read movie dir %w", err)
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !strings.Contains(file, "Season") {
			continue
		}

		seasonDir := filepath.Join(showDir, file)
		if err := l.buildFromShowSeasonDir(imdbID, seasonDir, log); err != nil {
			log.WithField("path", seasonDir).Error(err)
			continue
		}
	}

	return nil
}

func (l *Library) buildFromShowSeasonDir(imdbID, seasonDir string, log *logrus.Entry) error {
	dir, err := os.Open(seasonDir)
	if err != nil {
		return fmt.Errorf("failed to read movie dir %w", err)
	}
	defer dir.Close()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !l.fileConfig.IsVideo(file) {
			continue
		}

		episodePath := filepath.Join(seasonDir, file)

		episode, err := l.newEpisodeFromPath(episodePath)
		if err != nil {
			log.Errorf("library: failed to read episode NFO: %q", err)
			continue
		}

		episode.ShowImdbID = imdbID
		episode.ShowConfig = l.showConfig
		err = l.showIndex.Add(episode)
		if err != nil {
			log.Errorf("library: failed to add episode to the library: %q", err)
			continue
		}
	}

	return nil
}
