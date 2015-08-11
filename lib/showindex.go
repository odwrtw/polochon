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

	si.Lock()
	defer si.Unlock()
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

	return false, nil
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
		err = si.buildShowIndex()
	})
	return err
}

// SearchShowEpisodeBySlug returns a show from a slug
func (si *ShowIndex) SearchShowEpisodeBySlug(slug string) (Video, error) {
	if err := si.index(); err != nil {
		return nil, err
	}

	// Check if the slug is in the index
	si.Lock()
	filePath, ok := si.slugs[slug]
	if !ok {
		return nil, ErrSlugNotFound
	}
	si.Unlock()

	// Create a File from the path
	file := NewFileWithConfig(filePath, si.config)

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

	return episode, nil
}

// scanShow returns a show with the path for its episodes
func (si *ShowIndex) buildShowIndex() error {
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
		i, err := si.scanEpisodes(show.ImdbID, filePath)
		if err != nil {
			return err
		}
		si.ids[show.ImdbID] = i

		// No need to go deeper, the tvshow.nfo is on the second root folder
		return filepath.SkipDir
	})
	if err != nil {
		return err
	}

	si.log.Infof("Index built in %s", time.Since(start))

	return nil
}

// scanEpisodes returns the show episodes in a path
func (si *ShowIndex) scanEpisodes(imdbID, showRootPath string) (map[int]map[int]string, error) {
	showEpisodesIndex := map[int]map[int]string{}

	// Walk the files of a show
	err := filepath.Walk(showRootPath, func(filePath string, file os.FileInfo, err error) error {
		// Nothing to do on dir
		if file.IsDir() {
			return nil
		}

		// search for show type
		ext := path.Ext(filePath)

		var f *File
		for _, mext := range si.config.File.VideoExtentions {
			if ext == mext {
				f = NewFileWithConfig(filePath, si.config)
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

		// Create the season index if needed
		if _, ok := showEpisodesIndex[episode.Season]; !ok {
			showEpisodesIndex[episode.Season] = map[int]string{}
		}

		// Add the episode to the index
		showEpisodesIndex[episode.Season][episode.Episode] = filePath
		si.slugs[episode.Slug()] = filePath

		return nil
	})
	if err != nil {
		return nil, err
	}

	return showEpisodesIndex, nil
}
