package library

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/errors"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/media_index"
)

// Custom errors
var (
	ErrInvalidIndexVideoType      = errors.New("library: invalid index video type")
	ErrMissingMovieFilePath       = errors.New("library: movie has no file path")
	ErrMissingMovieImageURL       = errors.New("library: missing movie images URL")
	ErrMissingShowImageURL        = errors.New("library: missing URL to download show images")
	ErrMissingShowEpisodeFilePath = errors.New("library: missing file path")
)

// Config represents configuration for the library
type Config struct {
	MovieDir string
	ShowDir  string
}

// Library represents a collection of videos
type Library struct {
	Config
	movieIndex  *index.MovieIndex
	showIndex   *index.ShowIndex
	showConfig  polochon.ShowConfig
	movieConfig polochon.MovieConfig
	fileConfig  polochon.FileConfig
}

// New returns a list of videos
func New(fileConfig polochon.FileConfig, movieConfig polochon.MovieConfig, showConfig polochon.ShowConfig, vsConfig Config) *Library {
	return &Library{
		movieIndex:  index.NewMovieIndex(),
		showIndex:   index.NewShowIndex(),
		showConfig:  showConfig,
		movieConfig: movieConfig,
		fileConfig:  fileConfig,
		Config:      vsConfig,
	}
}

var exists = func(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

var download = func(URL, savePath string) error {
	resp, err := http.Get(URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	file, err := os.Create(savePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write from the net to the file
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// MovieIDs returns the movie ids
func (l *Library) MovieIDs() ([]string, error) {
	return l.movieIndex.IDs()
}

// HasVideo checks if the video is in the library
func (l *Library) HasVideo(video polochon.Video) (bool, error) {
	switch v := video.(type) {
	case *polochon.Movie:
		return l.HasMovie(v.ImdbID)
	case *polochon.ShowEpisode:
		return l.HasShowEpisode(v.ShowImdbID, v.Season, v.Episode)
	default:
		return false, ErrInvalidIndexVideoType
	}
}

// HasMovie returns true if the movie is in the store
func (l *Library) HasMovie(imdbID string) (bool, error) {
	return l.movieIndex.Has(imdbID)
}

// HasShowEpisode returns true if the show is in the store
func (l *Library) HasShowEpisode(imdbID string, season, episode int) (bool, error) {
	return l.showIndex.HasEpisode(imdbID, season, episode)
}

// Add video
func (l *Library) Add(video polochon.Video, log *logrus.Entry) error {
	ok, err := l.HasVideo(video)
	if err != nil {
		return err
	}
	switch v := video.(type) {
	case *polochon.Movie:
		if ok {
			err := l.DeleteMovie(v, log)
			if err != nil {
				return err
			}
		}

		if err := l.AddMovie(v, log); err != nil {
			return err
		}
	case *polochon.ShowEpisode:
		if ok {
			err := l.DeleteShowEpisode(v, log)
			if err != nil {
				return err
			}
		}

		if err := l.AddShowEpisode(v, log); err != nil {
			return err
		}
	default:
		return ErrInvalidIndexVideoType
	}

	return l.addToIndex(video)
}

func (l *Library) getMovieDir(movie *polochon.Movie) string {
	if movie.Year != 0 {
		return filepath.Join(l.MovieDir, fmt.Sprintf("%s (%d)", movie.Title, movie.Year))
	}
	return filepath.Join(l.MovieDir, movie.Title)
}

// AddMovie adds a movie to the store
func (l *Library) AddMovie(movie *polochon.Movie, log *logrus.Entry) error {
	if movie.Path == "" {
		return ErrMissingMovieFilePath
	}

	storePath := l.getMovieDir(movie)

	// If the movie already in the right dir there is nothing to do
	if path.Dir(movie.Path) == storePath {
		log.Debug("Movie already in the destination folder")
		return nil
	}

	// Remove movie dir if it exisits
	if ok := exists(storePath); ok {
		log.Debug("Movie folder exists, remove it")
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

	log.Debugf("Old path: %q, new path %q", movie.Path, newPath)
	if err := os.Rename(movie.Path, newPath); err != nil {
		return err
	}

	// Set the new movie path
	movie.Path = newPath

	// Create a symlink between the new and the old location
	if err := os.Symlink(movie.Path, oldPath); err != nil {
		log.Warnf("Error while making symlink between %s and %s : %+v", oldPath, movie.Path, err)
	}

	// Write NFO into the file
	if err := l.WriteNFOFile(movie.NfoPath(), movie); err != nil {
		return err
	}

	if movie.Fanart == "" || movie.Thumb == "" {
		return ErrMissingMovieImageURL
	}

	// Download images
	for _, img := range []struct {
		url      string
		savePath string
	}{
		{
			url:      movie.Fanart,
			savePath: movie.MovieFanartPath(),
		},
		{
			url:      movie.Thumb,
			savePath: movie.MovieThumbPath(),
		},
	} {
		if err := download(img.url, img.savePath); err != nil {
			return err
		}
	}

	return nil
}

func (l *Library) getShowDir(ep *polochon.ShowEpisode) string {
	return filepath.Join(l.ShowDir, ep.ShowTitle)
}

func (l *Library) getSeasonDir(ep *polochon.ShowEpisode) string {
	return filepath.Join(l.ShowDir, ep.ShowTitle, fmt.Sprintf("Season %d", ep.Season))
}

func (l *Library) showNFOPath(showDir string) string {
	return filepath.Join(showDir, "tvshow.nfo")
}

// AddShowEpisode adds an episode to the store
func (l *Library) AddShowEpisode(ep *polochon.ShowEpisode, log *logrus.Entry) error {
	if ep.Path == "" {
		return ErrMissingShowEpisodeFilePath
	}

	// If the show nfo does not exist yet, create it
	showDir := l.getShowDir(ep)
	showNFOPath := l.showNFOPath(showDir)

	if !exists(showNFOPath) {

		show := polochon.NewShowFromEpisode(ep)
		if err := show.GetDetails(log); err != nil {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				return err
			}
		}

		// Create show dir if necessary
		if !exists(showDir) {
			if err := os.Mkdir(showDir, os.ModePerm); err != nil {
				return err
			}
		}

		// Write NFO into the file
		if err := l.WriteNFOFile(showNFOPath, show); err != nil {
			return err
		}

		// Download show images
		if show.Fanart == "" || show.Banner == "" || show.Poster == "" {
			return ErrMissingShowImageURL
		}

		// download images
		images := map[string]string{
			show.Fanart: filepath.Join(showDir, "banner.jpg"),
			show.Banner: filepath.Join(showDir, "fanart.jpg"),
			show.Poster: filepath.Join(showDir, "poster.jpg"),
		}
		for url, savepath := range images {
			if err := download(url, savepath); err != nil {
				return err
			}
		}
	}

	// Create show season dir if necessary
	seasonDir := l.getSeasonDir(ep)
	if !exists(seasonDir) {
		if err := os.Mkdir(seasonDir, os.ModePerm); err != nil {
			return err
		}
	}

	// Move the file
	// If the show episode already in the right dir there is nothing to do
	if path.Dir(ep.Path) == seasonDir {
		log.Debug("show episode already in the destination folder")
		return nil
	}

	// Save the old path
	oldPath := ep.Path

	// Move the episode into the folder
	newPath := filepath.Join(seasonDir, path.Base(ep.Path))
	log.Debugf("Moving episode to folder Old path: %q, New path: %q", ep.Path, newPath)
	if err := os.Rename(ep.Path, newPath); err != nil {
		return err
	}

	// Set the new movie path
	ep.Path = newPath

	// Create a symlink between the new and the old location
	if err := os.Symlink(ep.Path, oldPath); err != nil {
		log.Warnf("Error while making symlink between %s and %s : %+v", oldPath, ep.Path, err)
	}

	// Create show NFO if necessary
	if err := l.WriteNFOFile(ep.NfoPath(), ep); err != nil {
		return err
	}

	return nil
}

// Delete will delete the video
func (l *Library) Delete(video polochon.Video, log *logrus.Entry) error {
	switch v := video.(type) {
	case *polochon.Movie:
		return l.DeleteMovie(v, log)
	case *polochon.ShowEpisode:
		return l.DeleteShowEpisode(v, log)
	default:
		return ErrInvalidIndexVideoType
	}
}

// DeleteMovie will delete the movie
func (l *Library) DeleteMovie(m *polochon.Movie, log *logrus.Entry) error {
	// Delete the movie
	d := filepath.Dir(m.Path)
	log.Infof("Removing Movie %s", d)

	if err := os.RemoveAll(d); err != nil {
		return err
	}
	// Remove the movie from the index
	if err := l.movieIndex.Remove(m, log); err != nil {
		return err
	}

	return nil
}

// DeleteShowEpisode will delete the showEpisode
func (l *Library) DeleteShowEpisode(se *polochon.ShowEpisode, log *logrus.Entry) error {
	// Delete the episode
	log.Infof("Removing ShowEpisode %q", se.Path)
	// Remove the episode
	if err := os.RemoveAll(se.Path); err != nil {
		return err
	}
	pathWithoutExt := se.PathWithoutExt()
	// Remove also the .nfo and .srt files
	for _, ext := range []string{"nfo", "srt"} {
		fileToDelete := fmt.Sprintf("%s.%s", pathWithoutExt, ext)
		log.Debugf("Removing %q", fileToDelete)
		// Remove file
		if err := os.RemoveAll(fileToDelete); err != nil {
			return err
		}
	}

	// Remove the episode from the index
	if err := l.showIndex.RemoveEpisode(se, log); err != nil {
		return err
	}

	// Season is empty, delete the whole season
	ok, err := l.showIndex.IsSeasonEmpty(se.ShowImdbID, se.Season)
	if err != nil {
		return err
	}
	if ok {
		// Delete the whole season
		seasonDir := l.getSeasonDir(se)
		if err := os.RemoveAll(seasonDir); err != nil {
			return err
		}
		// Remove the season from the index
		show := &polochon.Show{ImdbID: se.ShowImdbID}
		if err := l.showIndex.RemoveSeason(show, se.Season, log); err != nil {
			return err
		}
	}

	// Show is empty, delete the whole show from the index
	ok, err = l.showIndex.IsShowEmpty(se.ShowImdbID)
	if err != nil {
		return err
	}
	if ok {
		// Delete the whole Show
		showDir := l.getShowDir(se)
		if err := os.RemoveAll(showDir); err != nil {
			return err
		}
		// Remove the show from the index
		show := &polochon.Show{ImdbID: se.ShowImdbID}
		if err := l.showIndex.RemoveShow(show, log); err != nil {
			return err
		}
	}

	return nil
}

func (l *Library) addToIndex(video polochon.Video) error {
	switch v := video.(type) {
	case *polochon.Movie:
		return l.movieIndex.Add(v)
	case *polochon.ShowEpisode:
		return l.showIndex.Add(v)
	default:
		return ErrInvalidIndexVideoType
	}
}

// ShowIDs returns the show ids, seasons and episodes
func (l *Library) ShowIDs() (map[string]index.IndexedShow, error) {
	return l.showIndex.IDs()
}

// GetMovie returns the video by its imdb ID
func (l *Library) GetMovie(imdbID string) (*polochon.Movie, error) {
	path, err := l.movieIndex.MoviePath(imdbID)
	if err != nil {
		return nil, err
	}
	return l.newMovieFromPath(path)
}

// GetEpisode returns an episode if present in the index
func (l *Library) GetEpisode(imdbID string, season, episode int) (*polochon.ShowEpisode, error) {
	path, err := l.showIndex.EpisodePath(imdbID, season, episode)
	if err != nil {
		return nil, err
	}
	return l.newEpisodeFromPath(path)
}

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
		// Check err
		if err != nil {
			log.Errorf("video store: failed to walk %q", err)
			return nil
		}

		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for movie type
		ext := path.Ext(filePath)

		var moviePath string
		for _, mext := range l.fileConfig.VideoExtentions {
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
			log.Errorf("video store: failed to read movie NFO: %q", err)
			return nil
		}

		// Add the movie to the index
		l.addToIndex(movie)

		return nil
	})

	log.Infof("Index built in %s", time.Since(start))

	return err
}

func (l *Library) buildShowIndex(log *logrus.Entry) error {
	start := time.Now()

	// used to catch if the first root folder has been walked
	var rootWalked bool
	// Get only the parent folders
	err := filepath.Walk(l.ShowDir, func(filePath string, file os.FileInfo, err error) error {
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
			log.Errorf("video store: failed to read tv show NFO: %q", err)
			return nil
		}

		// Scan the path for the episodes
		err = l.scanEpisodes(show.ImdbID, filePath, log)
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
		// Check err
		if err != nil {
			log.Errorf("video store: failed to walk %q", err)
			return nil
		}

		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for show type
		ext := path.Ext(filePath)

		var epPath string
		for _, mext := range l.fileConfig.VideoExtentions {
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
			log.Errorf("video store: failed to read episode NFO: %q", err)
			return nil
		}

		episode.ShowImdbID = imdbID
		episode.ShowConfig = l.showConfig
		l.addToIndex(episode)

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// ReadNFOFile reads the NFO file
func (l *Library) ReadNFOFile(filePath string, i interface{}) error {
	// Open the file
	nfoFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer nfoFile.Close()

	return polochon.ReadNFO(nfoFile, i)
}

// WriteNFOFile write the NFO into a file
func (l *Library) WriteNFOFile(filePath string, i interface{}) error {
	return writeNFOFile(filePath, i, l)
}

// Fuction to be overwritten during the tests
var writeNFOFile = func(filePath string, i interface{}, l *Library) error {
	// Open the file
	nfoFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer nfoFile.Close()

	return polochon.WriteNFO(nfoFile, i)
}

// NewShowFromID returns a new Show from its id
func (l *Library) NewShowFromID(id string) (*polochon.Show, error) {
	path, err := l.showIndex.ShowPath(id)
	if err != nil {
		return nil, err
	}
	nfoPath := l.showNFOPath(path)

	s := &polochon.Show{}
	if err := l.ReadNFOFile(nfoPath, s); err != nil {
		return nil, err
	}

	return s, nil
}

// newShowFromPath returns a new Show from its path
func (l *Library) newShowFromPath(path string) (*polochon.Show, error) {
	s := &polochon.Show{}
	if err := l.ReadNFOFile(path, s); err != nil {
		return nil, err
	}

	return s, nil
}

// NewShowEpisodeFromPath returns a new ShowEpisode from its path
func (l *Library) newEpisodeFromPath(path string) (*polochon.ShowEpisode, error) {
	file := polochon.NewFile(path)
	se := polochon.NewShowEpisodeFromFile(l.showConfig, *file)

	if err := l.ReadNFOFile(file.NfoPath(), se); err != nil {
		return nil, err
	}

	return se, nil
}

// NewMovieFromPath returns a new Movie from its path
func (l *Library) newMovieFromPath(path string) (*polochon.Movie, error) {
	file := polochon.NewFile(path)
	m := polochon.NewMovieFromFile(l.movieConfig, *file)

	if err := l.ReadNFOFile(file.NfoPath(), m); err != nil {
		return nil, err
	}

	return m, nil
}
