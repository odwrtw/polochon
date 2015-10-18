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

// ShowIndex is an index for the shows
type ShowIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.Mutex
	// Build the index only once, use resync to add the Reset capability
	once resync.Once
	// Config
	config *Config
	// Logger
	log *logrus.Entry
	// ids keep the path of the show indexed by id, season and episode
	ids map[string]map[int]map[int]string
	// paths keep the path of the shows
	paths map[string]string
	// slugs keep the episode index by slug
	slugs map[string]string
}

// NewShowIndex returns a new show index
func NewShowIndex(config *Config, log *logrus.Entry) *ShowIndex {
	return &ShowIndex{
		config: config,
		log:    log.WithField("function", "showIndex"),
		ids:    map[string]map[int]map[int]string{},
		slugs:  map[string]string{},
	}
}

// ShowIds returns the show ids
func (si *ShowIndex) ShowIds() (map[string]map[int]map[int]string, error) {
	if err := si.index(); err != nil {
		return map[string]map[int]map[int]string{}, err
	}

	return si.ids, nil
}

// ShowSlugs returns the show slugs
func (si *ShowIndex) ShowSlugs() ([]string, error) {
	if err := si.index(); err != nil {
		return []string{}, err
	}

	si.Lock()
	defer si.Unlock()

	return extractMapKeys(si.slugs)
}

// Has searches for a show episode by id, season and episode and returns true
// if this episode is indexed
func (si *ShowIndex) Has(imdbID string, season, episode int) (bool, error) {
	if err := si.index(); err != nil {
		return false, err
	}

	si.Lock()
	defer si.Unlock()

	// Search for the show
	if _, ok := si.ids[imdbID]; !ok {
		return false, nil
	}

	// Search for the show
	_, ok := si.ids[imdbID][season]
	if !ok {
		return false, nil
	}

	// Search for the episode
	_, ok = si.ids[imdbID][season][episode]
	if !ok {
		return false, nil
	}

	return true, nil
}

// Function to be overwritten during the tests
var buildShowIndex = func(si *ShowIndex) error {
	return buildShowEpisodeIndex(si)
}

// Rebuild rebuilds the show index
func (si *ShowIndex) Rebuild() error {
	si.once.Reset()
	return si.index()
}

// index builds the show index only once
func (si *ShowIndex) index() error {
	var err error
	si.once.Do(func() {
		err = buildShowIndex(si)
	})
	return err
}

// SearchShowEpisodeBySlug returns a show from a slug
func (si *ShowIndex) SearchShowEpisodeBySlug(slug string) (Video, error) {
	if err := si.index(); err != nil {
		return nil, err
	}

	// Check if the slug is in the index
	filePath, err := si.searchShowEpisodeBySlug(slug)
	if err != nil {
		return nil, err
	}

	// Create a File from the path
	file := NewFileWithConfig(filePath, si.config.File)

	// Open the NFO
	nfoFile, err := os.Open(file.NfoPath())
	if err != nil {
		return nil, err
	}
	defer nfoFile.Close()

	// Unmarshal the NFO into an episode
	episode, err := readShowEpisodeNFO(nfoFile, si.config.Video.Show)
	if err != nil {
		return nil, err
	}

	episode.SetFile(file)
	// Set logger
	episode.log = si.log.WithFields(logrus.Fields{
		"type": "show_episode",
	})
	episode.Show = NewShowFromEpisode(episode)

	return episode, nil
}

// SearchShowEpisodeByImdbID returns a show from a slug
func (si *ShowIndex) SearchShowEpisodeByImdbID(imdbID string, sNum, eNum int) (Video, error) {
	if err := si.index(); err != nil {
		return nil, err
	}

	// Check if the slug is in the index
	filePath, err := si.searchShowEpisodeByImdbID(imdbID, sNum, eNum)
	if err != nil {
		return nil, err
	}

	ep, err := NewShowEpisodeFromPath(si.config.Video.Show, si.config.File, si.log, filePath)
	if err != nil {
		return nil, err
	}

	ep.Show = NewShowFromEpisode(ep)

	return ep, nil
}

// scanShow returns a show with the path for its episodes
func buildShowEpisodeIndex(si *ShowIndex) error {
	// Keep track of the time to build the index
	start := time.Now()
	si.log.Info("Building show index")

	// Reset the previous values
	si.slugs = map[string]string{}

	// used to catch if the first root folder have been walked
	var rootWalked bool
	// Get only the parent folders
	err := filepath.Walk(si.config.Video.Show.Dir, func(filePath string, file os.FileInfo, err error) error {
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
		nfoFile, err := os.Open(nfoPath)
		if err != nil {
			si.log.Errorf("video store: failed to open tv show NFO: %q", err)
			return nil
		}
		defer nfoFile.Close()

		show, err := readShowNFO(nfoFile, si.config.Video.Show)
		if err != nil {
			si.log.Errorf("video store: failed to read tv show NFO: %q", err)
			return nil
		}

		// Scan the path for the episodes
		err = si.scanEpisodes(show.ImdbID, filePath)
		if err != nil {
			return err
		}

		// No need to go deeper, the tvshow.nfo is on the second root folder
		return filepath.SkipDir
	})
	if err != nil {
		return err
	}

	si.log.Infof("Index built in %s", time.Since(start))

	return nil
}

// AddToIndex adds a show episode to the index
func (si *ShowIndex) AddToIndex(episode *ShowEpisode) error {
	si.Lock()
	defer si.Unlock()

	// Add the episode to the index
	// first by id
	if _, ok := si.ids[episode.ShowImdbID][episode.Season]; !ok {
		if _, ok := si.ids[episode.ShowImdbID]; !ok {
			si.ids[episode.ShowImdbID] = map[int]map[int]string{}
		}
		si.ids[episode.ShowImdbID][episode.Season] = map[int]string{}
	}
	si.ids[episode.ShowImdbID][episode.Season][episode.Episode] = episode.Path
	// then by slug
	si.slugs[episode.Slug()] = episode.Path

	return nil
}

// isShowEmpty returns true if the episode is the only episode in the
// whole show
func (si *ShowIndex) isShowEmpty(imdbID string) (bool, error) {
	si.Lock()
	defer si.Unlock()

	// Check if there is something in the show index
	if len(si.ids[imdbID]) != 0 {
		return false, nil
	}

	return true, nil
}

// isSeasonEmpty returns true if the season index is empty
func (si *ShowIndex) isSeasonEmpty(imdbID string, season int) (bool, error) {
	si.Lock()
	defer si.Unlock()

	// More than one season
	if len(si.ids[imdbID][season]) != 0 {
		return false, nil
	}

	return true, nil
}

// RemoveSeasonFromIndex removes the season from the index
func (si *ShowIndex) RemoveSeasonFromIndex(show *Show, season int) error {
	si.log.Infof("Deleting whole season %d of %s from index", season, show.ImdbID)

	for _, ep := range show.Episodes {
		if ep.Season == season {
			si.RemoveFromIndex(ep)
		}
	}

	return nil
}

// RemoveShowFromIndex removes the show from the index
func (si *ShowIndex) RemoveShowFromIndex(show *Show) error {
	si.log.Infof("Deleting whole show %s from index", show.ImdbID)

	for _, ep := range show.Episodes {
		si.RemoveFromIndex(ep)
	}
	delete(si.ids, show.ImdbID)

	return nil
}

// RemoveFromIndex removes the show episode from the index
func (si *ShowIndex) RemoveFromIndex(episode *ShowEpisode) error {
	slug := episode.Slug()
	imdbID := episode.ShowImdbID
	season := episode.Season

	// Delete from the slug index
	// Check if the slug is in the index
	_, err := si.searchShowEpisodeBySlug(slug)
	if err != nil {
		si.log.Errorf("Show not in slug index, WEIRD")
		return err
	}

	si.Lock()
	// Delete the episode from the index
	delete(si.slugs, slug)
	delete(si.ids[imdbID][season], episode.Episode)
	si.Unlock()

	return nil
}

// scanEpisodes returns the show episodes in a path
func (si *ShowIndex) scanEpisodes(imdbID, showRootPath string) error {
	// Walk the files of a show
	err := filepath.Walk(showRootPath, func(filePath string, file os.FileInfo, err error) error {
		// Check err
		if err != nil {
			si.log.Errorf("video store: failed to walk %q", err)
			return nil
		}

		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for show type
		ext := path.Ext(filePath)

		var f *File
		for _, mext := range si.config.File.VideoExtentions {
			if ext == mext {
				f = NewFileWithConfig(filePath, si.config.File)
				break
			}
		}

		// No file with an allowed extention found
		if f == nil {
			return nil
		}

		// Open the nfo file
		nfoFile, err := os.Open(f.NfoPath())
		if err != nil {
			si.log.Errorf("video store: failed to open file %q", filePath)
			return nil
		}
		defer nfoFile.Close()

		// Read the nfo file
		episode, err := readShowEpisodeNFO(nfoFile, si.config.Video.Show)
		if err != nil {
			si.log.Errorf("video store: failed to read episode NFO: %q", err)
			return nil
		}

		episode.SetFile(f)
		episode.ShowImdbID = imdbID
		episode.ShowConfig = si.config.Video.Show
		si.AddToIndex(episode)

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

// searchShowEpisodeBySlug returns a show from a slug
func (si *ShowIndex) searchShowEpisodeBySlug(slug string) (string, error) {
	si.Lock()
	defer si.Unlock()

	filePath, ok := si.slugs[slug]
	if !ok {
		return "", ErrSlugNotFound
	}

	return filePath, nil
}

// searchShowEpisodeByImdbID searches for a show from its imdbId
func (si *ShowIndex) searchShowEpisodeByImdbID(imdbID string, sNum, eNum int) (string, error) {
	si.Lock()
	defer si.Unlock()

	show, ok := si.ids[imdbID]
	if !ok {
		return "", ErrImdbIDNotFound
	}
	season, ok := show[sNum]
	if !ok {
		return "", ErrImdbIDNotFound
	}
	filePath, ok := season[eNum]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	return filePath, nil
}
