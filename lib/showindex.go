package polochon

import (
	"sync"

	"github.com/Sirupsen/logrus"
)

// ShowIndex is an index for the shows
type ShowIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// Logger
	log *logrus.Entry
	// ids keep the path of the show indexed by id, season and episode
	ids map[string]map[int]map[int]string
	// slugs keep the episode index by slug
	slugs map[string]string
}

// NewShowIndex returns a new show index
func NewShowIndex(log *logrus.Entry) *ShowIndex {
	return &ShowIndex{
		log:   log.WithField("function", "showIndex"),
		ids:   map[string]map[int]map[int]string{},
		slugs: map[string]string{},
	}
}

// Clear clears the show index
func (si *ShowIndex) Clear() {
	si.Lock()
	defer si.Unlock()
	si.ids = map[string]map[int]map[int]string{}
	si.slugs = map[string]string{}
}

// IDs returns the show ids
func (si *ShowIndex) IDs() (map[string]map[int]map[int]string, error) {
	si.RLock()
	defer si.RUnlock()
	return si.ids, nil
}

// Slugs returns the show slugs
func (si *ShowIndex) Slugs() ([]string, error) {
	si.RLock()
	defer si.RUnlock()
	return extractMapKeys(si.slugs)
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

// SearchBySlug returns a show from a slug
func (si *ShowIndex) SearchBySlug(slug string) (string, error) {
	si.RLock()
	defer si.RUnlock()

	filePath, err := si.searchShowEpisodeBySlug(slug)
	if err != nil {
		return "", err
	}
	return filePath, nil
}

// SearchByImdbID returns a show from a slug
func (si *ShowIndex) SearchByImdbID(imdbID string, sNum, eNum int) (string, error) {
	si.RLock()
	defer si.RUnlock()

	// Check if the slug is in the index
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
	// then by slug
	si.slugs[episode.Slug()] = episode.Path

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
func (si *ShowIndex) RemoveSeason(show *Show, season int) error {
	si.log.Infof("Deleting whole season %d of %s from index", season, show.ImdbID)

	for _, ep := range show.Episodes {
		if ep.Season == season {
			si.Remove(ep)
		}
	}

	return nil
}

// RemoveShow removes the show from the index
func (si *ShowIndex) RemoveShow(show *Show) error {
	si.log.Infof("Deleting whole show %s from index", show.ImdbID)

	for _, ep := range show.Episodes {
		si.Remove(ep)
	}
	si.Lock()
	defer si.Unlock()
	delete(si.ids, show.ImdbID)

	return nil
}

// Remove removes the show episode from the index
func (si *ShowIndex) Remove(episode *ShowEpisode) error {
	si.Lock()
	defer si.Unlock()

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

	// Delete the episode from the index
	delete(si.slugs, slug)
	delete(si.ids[imdbID][season], episode.Episode)

	return nil
}

// searchShowEpisodeBySlug returns a show from a slug
func (si *ShowIndex) searchShowEpisodeBySlug(slug string) (string, error) {
	filePath, ok := si.slugs[slug]
	if !ok {
		return "", ErrSlugNotFound
	}

	return filePath, nil
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
