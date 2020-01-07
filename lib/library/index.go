package library

import (
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// RebuildIndex rebuilds both the movie and show index
func (l *Library) RebuildIndex(log *logrus.Entry) error {
	// Create a goroutine for each index
	var wg sync.WaitGroup
	errc := make(chan error, 2)
	wg.Add(2)

	// Build the movie index
	l.movieIndex.Clear()
	go func() {
		defer wg.Done()
		if err := l.buildMovieIndex(log); err != nil {
			errc <- err
		}
	}()

	// Build the show index
	l.showIndex.Clear()
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
	err := filepath.Walk(l.MovieDir, func(filePath string, file os.FileInfo, err error) error {
		walkLog := log.WithField("path", filePath)
		// Check err
		if err != nil {
			walkLog.Errorf("library: failed to walk %q", err)
			return nil
		}

		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for movie type
		ext := path.Ext(filePath)

		var moviePath string
		for _, mext := range l.fileConfig.VideoExtensions {
			if ext == mext {
				moviePath = filePath
				break
			}
		}

		if moviePath == "" {
			return nil
		}

		// Read the movie informations
		movie, err := l.newMovieFromPath(moviePath)
		if err != nil {
			walkLog.Errorf("library: failed to read movie NFO: %q", err)
			return nil
		}

		// Add the movie to the index
		err = l.movieIndex.Add(movie)
		if err != nil {
			walkLog.Errorf("library: failed to add movie to the Library: %q", err)
			return nil
		}

		// Check for subtitles in the same folder
		for _, subLang := range l.SubtitleLanguages {
			if !l.HasSubtitle(movie, subLang) {
				continue
			}
			if err = l.movieIndex.AddSubtitle(movie, subLang); err != nil {
				walkLog.Warnf("library: failed to add subtitles %s : %q", subLang, err)
				continue
			}
		}

		return nil
	})

	log.Infof("Index built in %s", time.Since(start))

	return err
}

// HasSubtitle returns true if the subtitle exists on the disk
func (l *Library) HasSubtitle(v polochon.Video, lang polochon.Language) bool {
	if _, err := os.Stat(v.SubtitlePath(lang)); err == nil {
		// There is no such file
		return true
	}
	return false
}

func (l *Library) buildShowIndex(log *logrus.Entry) error {
	start := time.Now()

	// used to catch if the first root folder has been walked
	var rootWalked bool
	// Get only the parent folders
	err := filepath.Walk(l.ShowDir, func(filePath string, file os.FileInfo, err error) error {
		walkLog := log.WithField("path", filePath)
		if err != nil {
			walkLog.Errorf("library: failed to access path: %q", filePath)
			return err
		}

		// Only check directories
		if !file.IsDir() {
			return nil
		}

		// The root folder is only walk once
		if !rootWalked {
			rootWalked = true
			return nil
		}

		// Check if we can find the tvshow.nfo file
		nfoPath := l.showNFOPath(filePath)
		show, err := l.newShowFromPath(nfoPath)
		if err != nil {
			walkLog.Errorf("library: failed to read tv show NFO: %q", err)
			return nil
		}

		// Scan the path for the episodes
		err = l.scanEpisodes(show.ImdbID, filePath, walkLog)
		if err != nil {
			return err
		}

		// No need to go deeper, the tvshow.nfo is in the second root folder
		return filepath.SkipDir
	})
	if err != nil {
		return err
	}

	log.Infof("Index built in %s", time.Since(start))

	return nil

}

func (l *Library) scanEpisodes(imdbID, showRootPath string, log *logrus.Entry) error {
	// Walk the files of a show
	err := filepath.Walk(showRootPath, func(filePath string, file os.FileInfo, err error) error {
		walkLog := log.WithField("path", filePath)
		// Check err
		if err != nil {
			walkLog.Errorf("library: failed to walk %q", err)
			return nil
		}

		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for show type
		ext := path.Ext(filePath)

		var epPath string
		for _, mext := range l.fileConfig.VideoExtensions {
			if ext == mext {
				epPath = filePath
				break
			}
		}

		if epPath == "" {
			return nil
		}

		// Read the nfo file
		episode, err := l.newEpisodeFromPath(epPath)
		if err != nil {
			walkLog.Errorf("library: failed to read episode NFO: %q", err)
			return nil
		}

		episode.ShowImdbID = imdbID
		episode.ShowConfig = l.showConfig
		err = l.showIndex.Add(episode)
		if err != nil {
			walkLog.Errorf("library: failed to add movie to the Library: %q", err)
			return nil
		}

		// Check for subtitles in the same folder
		for _, subLang := range l.SubtitleLanguages {
			if !l.HasSubtitle(episode, subLang) {
				continue
			}
			if err = l.showIndex.AddSubtitle(episode, subLang); err != nil {
				walkLog.Warnf("library: failed to add subtitles %s : %q", subLang, err)
				continue
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
