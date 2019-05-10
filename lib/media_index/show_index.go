package index

import (
	"fmt"
	"path/filepath"
	"sync"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// ShowIndex is an index for the shows
type ShowIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// shows represents the index of the show
	shows map[string]*Show
}

// Show represents an indexed show
type Show struct {
	Path    string
	Seasons map[int]*Season
	Title   string
}

// Season represents an indexed season
type Season struct {
	Path     string           `json:"-"`
	Episodes map[int]*Episode `json:"episodes"`
}

// Episode represents an indexed episode
type Episode struct {
	Path      string                 `json:"-"`
	Metadata  polochon.VideoMetadata `json:"metadata"`
	Subtitles []polochon.Language    `json:"subtitles"`
}

// SeasonList returns the season numbers of the indexed show
func (si *Show) SeasonList() []int {
	return extractAndSortIndexedSeasonsMapKeys(si.Seasons)
}

// NewShowIndex returns a new show index
func NewShowIndex() *ShowIndex {
	return &ShowIndex{
		shows: map[string]*Show{},
	}
}

// Clear clears the show index
func (si *ShowIndex) Clear() {
	si.Lock()
	defer si.Unlock()
	si.shows = map[string]*Show{}
}

// Index returns the showIndex
func (si *ShowIndex) Index() map[string]*Show {
	si.RLock()
	defer si.RUnlock()
	return si.shows
}

// HasShow returns true if the show is already in the index
func (si *ShowIndex) HasShow(imdbID string) (bool, error) {
	_, err := si.ShowPath(imdbID)
	switch err {
	case nil:
		return true, nil
	case ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

// HasSeason returns true if the show is already in the index
func (si *ShowIndex) HasSeason(imdbID string, season int) (bool, error) {
	_, err := si.SeasonPath(imdbID, season)
	switch err {
	case nil:
		return true, nil
	case ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

// HasEpisode searches for a show episode by id, season and episode and returns true
// if this episode is indexed
func (si *ShowIndex) HasEpisode(imdbID string, season, episode int) (bool, error) {
	_, err := si.Episode(imdbID, season, episode)
	switch err {
	case nil:
		return true, nil
	case ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

// HasEpisodeSubtitle searches for a show episode by id, season and episode and
// returns true if this episode has a subtitle indexed
func (si *ShowIndex) HasEpisodeSubtitle(imdbID string, season, episode int, lang polochon.Language) (bool, error) {
	e, err := si.Episode(imdbID, season, episode)
	if err != nil {
		return false, err
	}
	for _, l := range e.Subtitles {
		if l == lang {
			return true, nil
		}
	}
	return false, nil
}

// Episode returns the episode path from the index
func (si *ShowIndex) Episode(imdbID string, sNum, eNum int) (*Episode, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return nil, ErrNotFound
	}

	season, ok := show.Seasons[sNum]
	if !ok {
		return nil, ErrNotFound
	}

	episode, ok := season.Episodes[eNum]
	if !ok {
		return nil, ErrNotFound
	}

	return episode, nil
}

// IndexedSeason returns the indexed season from the index
func (si *ShowIndex) IndexedSeason(imdbID string, sNum int) (*Season, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return nil, ErrNotFound
	}

	season, ok := show.Seasons[sNum]
	if !ok {
		return nil, ErrNotFound
	}

	return season, nil
}

// SeasonPath returns the season path from the index
func (si *ShowIndex) SeasonPath(imdbID string, sNum int) (string, error) {
	season, err := si.IndexedSeason(imdbID, sNum)
	if err != nil {
		return "", err
	}

	return season.Path, nil
}

// IndexedShow returns the indexed show from the index
func (si *ShowIndex) IndexedShow(imdbID string) (*Show, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return nil, ErrNotFound
	}

	return show, nil
}

// ShowPath returns the show path from the index
func (si *ShowIndex) ShowPath(imdbID string) (string, error) {
	show, err := si.IndexedShow(imdbID)
	if err != nil {
		return "", err
	}

	return show.Path, nil
}

// Add adds a show episode to the index
func (si *ShowIndex) Add(episode *polochon.ShowEpisode) error {
	// Get the parent paths
	seasonPath := filepath.Dir(episode.Path)
	showPath := filepath.Dir(seasonPath)

	// Check if the show is in the index
	hasShow, err := si.HasShow(episode.ShowImdbID)
	if err != nil {
		return err
	}
	if !hasShow {
		// Add a whole new show
		si.Lock()
		si.shows[episode.ShowImdbID] = &Show{
			Title:   episode.ShowTitle,
			Path:    showPath,
			Seasons: map[int]*Season{},
		}
		si.Unlock()
	}

	// Check if the season is in the index
	hasSeason, err := si.HasSeason(episode.ShowImdbID, episode.Season)
	if err != nil {
		return err
	}
	if !hasSeason {
		// Add a whole new season
		si.Lock()
		si.shows[episode.ShowImdbID].Seasons[episode.Season] = &Season{
			Path:     seasonPath,
			Episodes: map[int]*Episode{},
		}
		si.Unlock()
	}

	// Add the episode
	si.Lock()
	si.shows[episode.ShowImdbID].Seasons[episode.Season].Episodes[episode.Episode] = &Episode{
		Path:     episode.Path,
		Metadata: episode.VideoMetadata,
	}
	si.Unlock()

	return nil
}

// IsShowEmpty returns true if the episode is the only episode in the
// whole show
func (si *ShowIndex) IsShowEmpty(imdbID string) (bool, error) {
	si.RLock()
	defer si.RUnlock()

	if _, ok := si.shows[imdbID]; !ok {
		return true, nil
	}

	// Check if there is something in the show index
	if len(si.shows[imdbID].Seasons) != 0 {
		return false, nil
	}

	return true, nil
}

// IsSeasonEmpty returns true if the season index is empty
func (si *ShowIndex) IsSeasonEmpty(imdbID string, season int) (bool, error) {
	si.RLock()
	defer si.RUnlock()

	if _, ok := si.shows[imdbID]; !ok {
		return true, nil
	}

	if _, ok := si.shows[imdbID].Seasons[season]; !ok {
		return true, nil
	}

	// More than one season
	if len(si.shows[imdbID].Seasons[season].Episodes) != 0 {
		return false, nil
	}

	return true, nil
}

// RemoveSeason removes the season from the index
func (si *ShowIndex) RemoveSeason(show *polochon.Show, season int, log *logrus.Entry) error {
	log.Infof("Deleting whole season %d of %s from index", season, show.ImdbID)

	delete(si.shows[show.ImdbID].Seasons, season)

	return nil
}

// RemoveShow removes the show from the index
func (si *ShowIndex) RemoveShow(show *polochon.Show, log *logrus.Entry) error {
	log.Infof("Deleting whole show %s from index", show.ImdbID)

	si.Lock()
	defer si.Unlock()
	delete(si.shows, show.ImdbID)

	return nil
}

// RemoveEpisode removes the show episode from the index
func (si *ShowIndex) RemoveEpisode(episode *polochon.ShowEpisode, log *logrus.Entry) error {
	id := episode.ShowImdbID
	sNum := episode.Season
	eNum := episode.Episode

	// Check if the episode is in the index
	if _, err := si.Episode(id, sNum, eNum); err != nil {
		return err
	}

	// Delete the episode from the index
	si.Lock()
	defer si.Unlock()
	delete(si.shows[id].Seasons[sNum].Episodes, eNum)

	return nil
}

// AddSubtitle adds a movie subtitle to an index
func (si *ShowIndex) AddSubtitle(episode *polochon.ShowEpisode, lang polochon.Language) error {
	// Check that we have the show
	has, err := si.HasEpisode(episode.ShowImdbID, episode.Season, episode.Episode)
	if err != nil {
		return err
	}
	if !has {
		return fmt.Errorf("failed to add subtitle : show %s S%02dE%02d not indexed", episode.ShowImdbID, episode.Season, episode.Episode)
	}

	si.Lock()
	defer si.Unlock()

	// Append the subtitle to the index
	si.shows[episode.ShowImdbID].Seasons[episode.Season].Episodes[episode.Episode].Subtitles = append(
		si.shows[episode.ShowImdbID].Seasons[episode.Season].Episodes[episode.Episode].Subtitles,
		lang,
	)
	return nil
}
