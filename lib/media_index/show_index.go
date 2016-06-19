package index

import (
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

// ShowIndex is an index for the shows
type ShowIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// shows represents the index of the show
	shows map[string]IndexedShow
}

// IndexedSeason represents an indexed season
type IndexedSeason struct {
	Path     string
	Episodes map[int]string
}

// IndexedShow represents an indexed show
type IndexedShow struct {
	Path    string
	Seasons map[int]IndexedSeason
}

// NewShowIndex returns a new show index
func NewShowIndex() *ShowIndex {
	return &ShowIndex{
		shows: map[string]IndexedShow{},
	}
}

// Clear clears the show index
func (si *ShowIndex) Clear() {
	si.Lock()
	defer si.Unlock()
	si.shows = map[string]IndexedShow{}
}

// IDs returns the show ids
func (si *ShowIndex) IDs() (map[string]IndexedShow, error) {
	si.RLock()
	defer si.RUnlock()
	return si.shows, nil
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
	_, err := si.EpisodePath(imdbID, season, episode)
	switch err {
	case nil:
		return true, nil
	case ErrNotFound:
		return false, nil
	default:
		return false, err
	}
}

// EpisodePath returns the episode path from the index
func (si *ShowIndex) EpisodePath(imdbID string, sNum, eNum int) (string, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return "", ErrNotFound
	}

	season, ok := show.Seasons[sNum]
	if !ok {
		return "", ErrNotFound
	}

	filePath, ok := season.Episodes[eNum]
	if !ok {
		return "", ErrNotFound
	}

	return filePath, nil
}

// SeasonPath returns the season path from the index
func (si *ShowIndex) SeasonPath(imdbID string, sNum int) (string, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return "", ErrNotFound
	}

	season, ok := show.Seasons[sNum]
	if !ok {
		return "", ErrNotFound
	}

	return season.Path, nil
}

// ShowPath returns the show path from the index
func (si *ShowIndex) ShowPath(imdbID string) (string, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return "", ErrNotFound
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
		si.Lock()
		defer si.Unlock()

		// Add a whole new show
		si.shows[episode.ShowImdbID] = IndexedShow{
			Path: showPath,
			Seasons: map[int]IndexedSeason{
				episode.Season: {
					Path:     seasonPath,
					Episodes: map[int]string{episode.Episode: episode.Path},
				},
			},
		}
		return nil
	}

	// Check if the season is in the index
	hasSeason, err := si.HasSeason(episode.ShowImdbID, episode.Season)
	if err != nil {
		return err
	}
	if !hasSeason {
		si.Lock()
		defer si.Unlock()

		// Add a whole new season
		si.shows[episode.ShowImdbID].Seasons[episode.Season] = IndexedSeason{
			Path:     seasonPath,
			Episodes: map[int]string{episode.Episode: episode.Path},
		}
		return nil
	}

	// The show and the season are already indexed
	si.shows[episode.ShowImdbID].Seasons[episode.Season].Episodes[episode.Episode] = episode.Path

	return nil
}

// IsShowEmpty returns true if the episode is the only episode in the
// whole show
func (si *ShowIndex) IsShowEmpty(imdbID string) (bool, error) {
	si.RLock()
	defer si.RUnlock()

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
	if _, err := si.EpisodePath(id, sNum, eNum); err != nil {
		return err
	}

	// Delete the episode from the index
	si.Lock()
	defer si.Unlock()
	delete(si.shows[id].Seasons[sNum].Episodes, eNum)

	return nil
}
