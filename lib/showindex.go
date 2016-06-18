package polochon

import (
	"sync"

	"github.com/Sirupsen/logrus"
)

// ShowIndex is an index for the shows
type ShowIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// ids keep the path of the show indexed by id, season and episode
	ids map[string]map[int]map[int]string
}

// NewShowIndex returns a new show index
func NewShowIndex() *ShowIndex {
	return &ShowIndex{
		ids: map[string]map[int]map[int]string{},
	}
}

// Clear clears the show index
func (si *ShowIndex) Clear() {
	si.Lock()
	defer si.Unlock()
	si.ids = map[string]map[int]map[int]string{}
}

// IDs returns the show ids
func (si *ShowIndex) IDs() (map[string]map[int]map[int]string, error) {
	si.RLock()
	defer si.RUnlock()
	return si.ids, nil
}

// Has searches for a show episode by id, season and episode and returns true
// if this episode is indexed
func (si *ShowIndex) Has(imdbID string, season, episode int) (bool, error) {
	si.RLock()
	defer si.RUnlock()

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

// SearchByImdbID returns a show from an id
func (si *ShowIndex) SearchByImdbID(imdbID string, sNum, eNum int) (string, error) {
	si.RLock()
	defer si.RUnlock()

	// Check if the id is in the index
	filePath, err := si.searchShowEpisodeByImdbID(imdbID, sNum, eNum)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

// Add adds a show episode to the index
func (si *ShowIndex) Add(episode *ShowEpisode) error {
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

	return nil
}

// IsShowEmpty returns true if the episode is the only episode in the
// whole show
func (si *ShowIndex) IsShowEmpty(imdbID string) (bool, error) {
	si.RLock()
	defer si.RUnlock()

	// Check if there is something in the show index
	if len(si.ids[imdbID]) != 0 {
		return false, nil
	}

	return true, nil
}

// IsSeasonEmpty returns true if the season index is empty
func (si *ShowIndex) IsSeasonEmpty(imdbID string, season int) (bool, error) {
	si.RLock()
	defer si.RUnlock()

	// More than one season
	if len(si.ids[imdbID][season]) != 0 {
		return false, nil
	}

	return true, nil
}

// RemoveSeason removes the season from the index
func (si *ShowIndex) RemoveSeason(show *Show, season int, log *logrus.Entry) error {
	log.Infof("Deleting whole season %d of %s from index", season, show.ImdbID)

	for _, ep := range show.Episodes {
		if ep.Season == season {
			si.Remove(ep, log)
		}
	}

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
	delete(si.ids, show.ImdbID)

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
	_, err := si.searchShowEpisodeByImdbID(id, sNum, eNum)
	if err != nil {
		log.Errorf("Show not in the index, WEIRD")
		return err
	}

	// Delete the episode from the index
	delete(si.ids[id][sNum], eNum)

	return nil
}

// searchShowEpisodeByImdbID searches for a show from its imdbId
func (si *ShowIndex) searchShowEpisodeByImdbID(imdbID string, sNum, eNum int) (string, error) {
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
