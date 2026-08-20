package library

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
)

// RebuildIndex rebuilds both the movie and show index
func (l *Library) RebuildIndex() error {
	var wg sync.WaitGroup
	errc := make(chan error, 2)
	wg.Add(2)

	// Build the movie index
	go func() {
		defer wg.Done()
		if err := l.buildMovieIndex(); err != nil {
			errc <- err
		}
	}()

	// Build the show index
	go func() {
		defer wg.Done()
		if err := l.buildShowIndex(); err != nil {
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

func (l *Library) buildMovieIndex() error {
	start := time.Now()
	defer func() {
		l.log.Info("movie index built", "duration", time.Since(start))
	}()
	l.movieIndex.Clear()

	root, err := os.Open(l.MovieDir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	dirs, err := root.Readdirnames(-1)
	if err != nil {
		return err
	}

	reg := regexp.MustCompile(`.*\(\d{4}\)$`)

	for _, d := range dirs {
		if !reg.MatchString(d) {
			l.log.Warn("invalid movie dir", "dir", d)
			continue
		}

		if err := l.buildFromMovieDir(d); err != nil {
			l.log.Error("failed to build movie index entry", "dir", d, "error", err)
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
	defer func() { _ = dir.Close() }()

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

func (l *Library) buildShowIndex() error {
	start := time.Now()
	defer func() {
		l.log.Info("show index built", "duration", time.Since(start))
	}()

	l.showIndex.Clear()

	root, err := os.Open(l.ShowDir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()

	dirs, err := root.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, d := range dirs {
		showDir := filepath.Join(l.ShowDir, d)
		nfoPath := l.showNFOPath(showDir)

		show, err := l.newShowFromPath(nfoPath)
		if err != nil {
			l.log.Error("failed to read tv show NFO", "error", err)
			continue
		}
		if err := l.buildFromShowDir(show, showDir); err != nil {
			l.log.Error("failed to build show index entry", "dir", d, "error", err)
		}
	}

	return nil
}

func (l *Library) buildFromShowDir(show *polochon.Show, showDir string) error {
	dir, err := os.Open(showDir)
	if err != nil {
		return fmt.Errorf("failed to read movie dir %w", err)
	}
	defer func() { _ = dir.Close() }()

	files, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !strings.Contains(file, "Season") {
			continue
		}

		seasonDir := filepath.Join(showDir, file)
		if err := l.buildFromShowSeasonDir(show, seasonDir); err != nil {
			l.log.Error("failed to build show season index entry", "path", seasonDir, "error", err)
			continue
		}
	}

	return nil
}

func (l *Library) buildFromShowSeasonDir(show *polochon.Show, seasonDir string) error {
	dir, err := os.Open(seasonDir)
	if err != nil {
		return fmt.Errorf("failed to read movie dir %w", err)
	}
	defer func() { _ = dir.Close() }()

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
			l.log.Error("failed to read episode NFO", "error", err)
			continue
		}

		episode.ShowImdbID = show.ImdbID
		episode.Show = show
		episode.ShowConfig = l.showConfig
		if err := l.showIndex.Add(episode); err != nil {
			l.log.Error("failed to add episode to the library", "error", err)
			continue
		}
	}

	return nil
}
