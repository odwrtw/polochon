package polochon

import (
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
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

// Has searches for a show episode by id, season and episode and returns true
// if this episode is indexed
func (si *ShowIndex) Has(imdbID string, season, episode int) (bool, error) {
	si.RLock()
	defer si.RUnlock()

	// Search for the show
	if _, ok := si.shows[imdbID]; !ok {
		return false, nil
	}

	// Search for the show
	_, ok := si.shows[imdbID].Seasons[season]
	if !ok {
		return false, nil
	}

	// Search for the episode
	_, ok = si.shows[imdbID].Seasons[season].Episodes[episode]
	if !ok {
		return false, nil
	}

	return true, nil
}

// EpisodePath returns the episode path from the index
func (si *ShowIndex) EpisodePath(imdbID string, sNum, eNum int) (string, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	season, ok := show.Seasons[sNum]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	filePath, ok := season.Episodes[eNum]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	return filePath, nil
}

// SeasonPath returns the season path from the index
func (si *ShowIndex) SeasonPath(imdbID string, sNum int) (string, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	season, ok := show.Seasons[sNum]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	return season.Path, nil
}

// ShowPath returns the show path from the index
func (si *ShowIndex) ShowPath(imdbID string) (string, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return "", ErrImdbIDNotFound
	}

	return show.Path, nil
}

// Add adds a show episode to the index
func (si *ShowIndex) Add(episode *ShowEpisode) error {
	si.Lock()
	defer si.Unlock()

	// Get the parent paths
	seasonPath := filepath.Dir(episode.Path)
	showPath := filepath.Dir(seasonPath)

	// The show is not yet indexed
	if _, ok := si.shows[episode.ShowImdbID]; !ok {
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

	// The season is not yet indexed
	if _, ok := si.shows[episode.ShowImdbID].Seasons[episode.Season]; !ok {
		si.shows[episode.ShowImdbID].Seasons[episode.Season] = IndexedSeason{
			Path:     "mama",
			Episodes: map[int]string{episode.Episode: episode.Path},
		}
		return nil
	}

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
func (si *ShowIndex) RemoveSeason(show *Show, season int, log *logrus.Entry) error {
	log.Infof("Deleting whole season %d of %s from index", season, show.ImdbID)

	delete(si.shows[show.ImdbID].Seasons, season)

	return nil
}

// RemoveShow removes the show from the index
func (si *ShowIndex) RemoveShow(show *Show, log *logrus.Entry) error {
	log.Infof("Deleting whole show %s from index", show.ImdbID)

	for _, ep := range show.Episodes {
		si.Remove(ep, log)
	}
	si.Lock()
	defer si.Unlock()
	delete(si.shows, show.ImdbID)

	return nil
}

// Remove removes the show episode from the index
func (si *ShowIndex) Remove(episode *ShowEpisode, log *logrus.Entry) error {
	si.Lock()
	defer si.Unlock()

	id := episode.ShowImdbID
	sNum := episode.Season
	eNum := episode.Episode

	// Delete from the index
	_, err := si.EpisodePath(id, sNum, eNum)
	if err != nil {
		log.Errorf("Show not in the index, WEIRD")
		return err
	}

	// Delete the episode from the index
	delete(si.shows[id].Seasons[sNum].Episodes, eNum)

	return nil
}
