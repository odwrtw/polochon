package polochon

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
)

// Video errors
var (
	ErrImdbIDNotFound             = errors.New("polochon: no such file with this imdbID")
	ErrInvalidIndexVideoType      = errors.New("polochon: invalid index video type")
	ErrMissingMovieFilePath       = errors.New("polochon: movie has no file path")
	ErrMissingMovieImageURL       = errors.New("polochon: missing movie images URL")
	ErrMissingShowImageURL        = errors.New("polochon: missing URL to download show images")
	ErrMissingShowEpisodeFilePath = errors.New("polochon: missing file path")
)

// VideoStoreConfig represents configuration for VideoStore
type VideoStoreConfig struct {
	MovieDir string
	ShowDir  string
}

// VideoStore represent a collection of videos
type VideoStore struct {
	VideoStoreConfig
	movieIndex  *MovieIndex
	showIndex   *ShowIndex
	showConfig  ShowConfig
	movieConfig MovieConfig
	fileConfig  FileConfig
}

// NewVideoStore returns a list of videos
func NewVideoStore(fileConfig FileConfig, movieConfig MovieConfig, showConfig ShowConfig, vsConfig VideoStoreConfig) *VideoStore {
	return &VideoStore{
		movieIndex:       NewMovieIndex(),
		showIndex:        NewShowIndex(),
		showConfig:       showConfig,
		movieConfig:      movieConfig,
		fileConfig:       fileConfig,
		VideoStoreConfig: vsConfig,
	}
}

var exists = func(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}

var mkdir = func(path string) error {
	return os.Mkdir(path, os.ModePerm)
}

var move = func(from string, to string) error {
	return os.Rename(from, to)
}

var remove = func(path string) error {
	return os.RemoveAll(path)
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

// MovieIds returns the movie ids
func (vs *VideoStore) MovieIds() ([]string, error) {
	return vs.movieIndex.IDs()
}

// Has returns true if the video is in the store
func (vs *VideoStore) Has(video Video) (bool, error) {
	switch v := video.(type) {
	case *Movie:
		return vs.HasMovie(v.ImdbID)
	case *ShowEpisode:
		return vs.HasShowEpisode(v.ShowImdbID, v.Season, v.Episode)
	default:
		return false, ErrInvalidIndexVideoType
	}
}

// HasMovie returns true if the movie is in the store
func (vs *VideoStore) HasMovie(imdbID string) (bool, error) {
	return vs.movieIndex.Has(imdbID)
}

// HasShowEpisode returns true if the show is in the store
func (vs *VideoStore) HasShowEpisode(imdbID string, season, episode int) (bool, error) {
	return vs.showIndex.Has(imdbID, season, episode)
}

// Add video
func (vs *VideoStore) Add(video Video, log *logrus.Entry) error {
	ok, err := vs.Has(video)
	if err != nil {
		return err
	}
	switch v := video.(type) {
	case *Movie:
		if ok {
			err := vs.DeleteMovie(v, log)
			if err != nil {
				return err
			}
		}

		if err := vs.AddMovie(v, log); err != nil {
			return err
		}
	case *ShowEpisode:
		if ok {
			err := vs.DeleteShowEpisode(v, log)
			if err != nil {
				return err
			}
		}

		if err := vs.AddShowEpisode(v, log); err != nil {
			return err
		}
	default:
		return ErrInvalidIndexVideoType
	}

	return vs.addToIndex(video)
}

func (vs *VideoStore) getMovieDir(movie *Movie) string {
	if movie.Year != 0 {
		return filepath.Join(vs.MovieDir, fmt.Sprintf("%s (%d)", movie.Title, movie.Year))
	}
	return filepath.Join(vs.MovieDir, movie.Title)
}

// AddMovie adds a movie to the store
func (vs *VideoStore) AddMovie(movie *Movie, log *logrus.Entry) error {
	if movie.Path == "" {
		return ErrMissingMovieFilePath
	}

	storePath := vs.getMovieDir(movie)

	// If the movie already in the right dir there is nothing to do
	if path.Dir(movie.Path) == storePath {
		log.Debug("Movie already in the destination folder")
		return nil
	}

	// Remove movie dir if it exisits
	if ok := exists(storePath); ok {
		log.Debug("Movie folder exists, remove it")
		if err := remove(storePath); err != nil {
			return err
		}
	}

	// Create the folder
	if err := mkdir(storePath); err != nil {
		return err
	}

	// Move the movie into the folder
	newPath := filepath.Join(storePath, path.Base(movie.Path))

	// Save the old path
	oldPath := movie.Path

	log.Debugf("Old path: %q, new path %q", movie.Path, newPath)
	if err := move(movie.Path, newPath); err != nil {
		return err
	}

	// Set the new movie path
	movie.Path = newPath

	// Create a symlink between the new and the old location
	if err := os.Symlink(movie.Path, oldPath); err != nil {
		log.Warnf("Error while making symlink between %s and %s : %+v", oldPath, movie.Path, err)
	}

	// Write NFO into the file
	if err := vs.WriteNFOFile(movie.NfoPath(), movie); err != nil {
		return err
	}

	if movie.Fanart == "" || movie.Thumb == "" {
		return ErrMissingMovieImageURL
	}

	// Download images
	images := map[string]string{
		movie.Fanart: movie.File.filePathWithoutExt() + "-fanart.jpg",
		movie.Thumb:  filepath.Join(path.Dir(movie.File.Path), "/poster.jpg"),
	}
	for URL, savePath := range images {
		if err := download(URL, savePath); err != nil {
			return err
		}
	}
	return nil
}

func (vs *VideoStore) getShowDir(ep *ShowEpisode) string {
	return filepath.Join(vs.ShowDir, ep.ShowTitle)
}

func (vs *VideoStore) getSeasonDir(ep *ShowEpisode) string {
	return filepath.Join(vs.ShowDir, ep.ShowTitle, fmt.Sprintf("Season %d", ep.Season))
}

// AddShowEpisode adds an episode to the store
func (vs *VideoStore) AddShowEpisode(ep *ShowEpisode, log *logrus.Entry) error {
	if ep.Path == "" {
		return ErrMissingShowEpisodeFilePath
	}

	// If the show nfo does not exist yet, create it
	showDir := vs.getShowDir(ep)
	showNFOPath := filepath.Join(showDir, "tvshow.nfo")

	if !exists(showNFOPath) {

		show := NewShowFromEpisode(ep)
		if err := show.GetDetails(log); err != nil {
			errors.LogErrors(log, err)
			if errors.IsFatal(err) {
				return err
			}
		}

		// Create show dir if necessary
		if !exists(showDir) {
			if err := mkdir(showDir); err != nil {
				return err
			}
		}

		// Write NFO into the file
		if err := vs.WriteNFOFile(showNFOPath, show); err != nil {
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
	seasonDir := vs.getSeasonDir(ep)
	if !exists(seasonDir) {
		if err := mkdir(seasonDir); err != nil {
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
	if err := move(ep.Path, newPath); err != nil {
		return err
	}

	// Set the new movie path
	ep.Path = newPath

	// Create a symlink between the new and the old location
	if err := os.Symlink(ep.Path, oldPath); err != nil {
		log.Warnf("Error while making symlink between %s and %s : %+v", oldPath, ep.Path, err)
	}

	// Create show NFO if necessary
	if err := vs.WriteNFOFile(ep.NfoPath(), ep); err != nil {
		return err
	}

	return nil
}

// Delete will delete the video
func (vs *VideoStore) Delete(video Video, log *logrus.Entry) error {
	switch v := video.(type) {
	case *Movie:
		return vs.DeleteMovie(v, log)
	case *ShowEpisode:
		return vs.DeleteShowEpisode(v, log)
	default:
		return ErrInvalidIndexVideoType
	}
}

// DeleteMovie will delete the movie
func (vs *VideoStore) DeleteMovie(m *Movie, log *logrus.Entry) error {
	// Delete the movie
	d := filepath.Dir(m.Path)
	log.Infof("Removing Movie %s", d)

	if err := os.RemoveAll(d); err != nil {
		return err
	}
	// Remove the movie from the index
	if err := vs.movieIndex.Remove(m, log); err != nil {
		return err
	}

	return nil
}

// DeleteShowEpisode will delete the showEpisode
func (vs *VideoStore) DeleteShowEpisode(se *ShowEpisode, log *logrus.Entry) error {
	// Delete the episode
	log.Infof("Removing ShowEpisode %q", se.Path)
	// Remove the episode
	if err := os.RemoveAll(se.Path); err != nil {
		return err
	}
	pathWithoutExt := se.filePathWithoutExt()
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
	if err := vs.showIndex.Remove(se, log); err != nil {
		return err
	}

	// Season is empty, delete the whole season
	ok, err := vs.showIndex.IsSeasonEmpty(se.ShowImdbID, se.Season)
	if err != nil {
		return err
	}
	if ok {
		// Delete the whole season
		seasonDir := vs.getSeasonDir(se)
		if err := os.RemoveAll(seasonDir); err != nil {
			return err
		}
		// Remove the season from the index
		if err := vs.showIndex.RemoveSeason(se.Show, se.Season, log); err != nil {
			return err
		}
	}

	// Show is empty, delete the whole show from the index
	ok, err = vs.showIndex.IsShowEmpty(se.ShowImdbID)
	if err != nil {
		return err
	}
	if ok {
		// Delete the whole Show
		showDir := vs.getShowDir(se)
		if err := os.RemoveAll(showDir); err != nil {
			return err
		}
		// Remove the show from the index
		if err := vs.showIndex.RemoveShow(se.Show, log); err != nil {
			return err
		}
	}

	return nil
}

func (vs *VideoStore) addToIndex(video Video) error {
	switch v := video.(type) {
	case *Movie:
		return vs.movieIndex.Add(v)
	case *ShowEpisode:
		return vs.showIndex.Add(v)
	default:
		return ErrInvalidIndexVideoType
	}
}

// ShowIds returns the show ids, seasons and episodes
func (vs *VideoStore) ShowIds() (map[string]map[int]map[int]string, error) {
	return vs.showIndex.IDs()
}

// SearchMovieByImdbID returns the video by its imdb ID
func (vs *VideoStore) SearchMovieByImdbID(imdbID string) (Video, error) {
	path, err := vs.movieIndex.SearchByImdbID(imdbID)
	if err != nil {
		return nil, err
	}
	return vs.NewMovieFromPath(path)
}

// SearchShowEpisodeByImdbID search for a show episode by its imdb ID
func (vs *VideoStore) SearchShowEpisodeByImdbID(imdbID string, sNum, eNum int) (Video, error) {
	path, err := vs.showIndex.SearchByImdbID(imdbID, sNum, eNum)
	if err != nil {
		return nil, err
	}
	return vs.NewShowEpisodeFromPath(path)
}

// RebuildIndex rebuilds both the movie and show index
func (vs *VideoStore) RebuildIndex(log *logrus.Entry) error {
	// Create a goroutine for each index
	var wg sync.WaitGroup
	errc := make(chan error, 2)
	wg.Add(2)

	// Build the movie index
	vs.movieIndex.Clear()
	go func() {
		defer wg.Done()
		if err := vs.buildMovieIndex(log); err != nil {
			errc <- err
		}
	}()

	// Build the show index
	vs.showIndex.Clear()
	go func() {
		defer wg.Done()
		if err := vs.buildShowIndex(log); err != nil {
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

func (vs *VideoStore) buildMovieIndex(log *logrus.Entry) error {
	start := time.Now()
	err := filepath.Walk(vs.MovieDir, func(filePath string, file os.FileInfo, err error) error {
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
		for _, mext := range vs.fileConfig.VideoExtentions {
			if ext == mext {
				moviePath = filePath
				break
			}
		}

		if moviePath == "" {
			return nil
		}

		// Read the movie informations
		movie, err := vs.NewMovieFromPath(moviePath)
		if err != nil {
			log.Errorf("video store: failed to read movie NFO: %q", err)
			return nil
		}

		// Add the movie to the index
		vs.addToIndex(movie)

		return nil
	})

	log.Infof("Index built in %s", time.Since(start))

	return err
}

func (vs *VideoStore) buildShowIndex(log *logrus.Entry) error {
	start := time.Now()

	// used to catch if the first root folder have been walked
	var rootWalked bool
	// Get only the parent folders
	err := filepath.Walk(vs.ShowDir, func(filePath string, file os.FileInfo, err error) error {
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
		nfoPath := filepath.Join(filePath, "tvshow.nfo")
		show, err := vs.NewShowFromPath(nfoPath)
		if err != nil {
			log.Errorf("video store: failed to read tv show NFO: %q", err)
			return nil
		}

		// Scan the path for the episodes
		err = vs.scanEpisodes(show.ImdbID, filePath, log)
		if err != nil {
			return err
		}

		// No need to go deeper, the tvshow.nfo is on the second root folder
		return filepath.SkipDir
	})
	if err != nil {
		return err
	}

	log.Infof("Index built in %s", time.Since(start))

	return nil

}

func (vs *VideoStore) scanEpisodes(imdbID, showRootPath string, log *logrus.Entry) error {
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
		for _, mext := range vs.fileConfig.VideoExtentions {
			if ext == mext {
				epPath = filePath
				break
			}
		}

		if epPath == "" {
			return nil
		}

		// Read the nfo file
		episode, err := vs.NewShowEpisodeFromPath(epPath)
		if err != nil {
			log.Errorf("video store: failed to read episode NFO: %q", err)
			return nil
		}

		episode.ShowImdbID = imdbID
		episode.ShowConfig = vs.showConfig
		vs.addToIndex(episode)

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// ReadNFOFile reads the NFO file
func (vs *VideoStore) ReadNFOFile(filePath string, i interface{}) error {
	// Open the file
	nfoFile, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer nfoFile.Close()

	return ReadNFO(nfoFile, i)
}

// WriteNFOFile write the NFO into a file
func (vs *VideoStore) WriteNFOFile(filePath string, i interface{}) error {
	return writeNFOFile(filePath, i, vs)
}

// Fuction to be overwritten during the tests
var writeNFOFile = func(filePath string, i interface{}, vs *VideoStore) error {
	// Open the file
	nfoFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer nfoFile.Close()

	return WriteNFO(nfoFile, i)
}

// NewShowFromPath returns a new Show from its path
func (vs *VideoStore) NewShowFromPath(path string) (*Show, error) {
	s := &Show{}
	if err := vs.ReadNFOFile(path, s); err != nil {
		return nil, err
	}

	return s, nil
}

// NewShowEpisodeFromPath returns a new ShowEpisode from its path
func (vs *VideoStore) NewShowEpisodeFromPath(path string) (*ShowEpisode, error) {
	file := NewFileWithConfig(path, vs.fileConfig)
	se := NewShowEpisodeFromFile(vs.showConfig, *file)

	if err := vs.ReadNFOFile(file.NfoPath(), se); err != nil {
		return nil, err
	}

	return se, nil
}

// NewMovieFromPath returns a new Movie from its path
func (vs *VideoStore) NewMovieFromPath(path string) (*Movie, error) {
	file := NewFileWithConfig(path, vs.fileConfig)
	m := NewMovieFromFile(vs.movieConfig, *file)

	if err := vs.ReadNFOFile(file.NfoPath(), m); err != nil {
		return nil, err
	}

	return m, nil
}
