package index

import (
	"path/filepath"
	"sync"

	polochon "github.com/odwrtw/polochon/lib"
)

// ShowIndex is an index for the shows
type ShowIndex struct {
	// Mutex to protect reads / writes made concurrently by the http server
	sync.RWMutex
	// shows represents the index of the show
	shows map[string]*polochon.Show
}

// NewShowIndex returns a new show index
func NewShowIndex() *ShowIndex {
	return &ShowIndex{
		shows: map[string]*polochon.Show{},
	}
}

// Clear clears the show index
func (si *ShowIndex) Clear() {
	si.Lock()
	defer si.Unlock()
	si.shows = map[string]*polochon.Show{}
}

// Index returns the show index
func (si *ShowIndex) Index() map[string]*polochon.Show {
	si.RLock()
	defer si.RUnlock()
	return si.shows
}

// HasShow returns true if the show is already in the index
func (si *ShowIndex) HasShow(imdbID string) (bool, error) {
	si.RLock()
	defer si.RUnlock()
	_, ok := si.shows[imdbID]
	return ok, nil
}

// HasSeason returns true if the season is already in the index
func (si *ShowIndex) HasSeason(imdbID string, season int) (bool, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return false, nil
	}
	_, ok = show.Seasons[season]
	return ok, nil
}

// HasEpisode searches for a show episode and returns true if indexed
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

// HasEpisodeSubtitle returns true if the episode has a subtitle in the given language
func (si *ShowIndex) HasEpisodeSubtitle(imdbID string, season, episode int, sub *polochon.Subtitle) (bool, error) {
	e, err := si.Episode(imdbID, season, episode)
	if err != nil {
		return false, err
	}
	for _, s := range e.Subtitles {
		if s.Lang == sub.Lang {
			return true, nil
		}
	}
	return false, nil
}

// Episode returns the episode from the index
func (si *ShowIndex) Episode(imdbID string, sNum, eNum int) (*polochon.ShowEpisode, error) {
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
func (si *ShowIndex) IndexedSeason(imdbID string, sNum int) (*polochon.ShowSeason, error) {
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

// SeasonPath returns the season path derived from its episodes
func (si *ShowIndex) SeasonPath(imdbID string, sNum int) (string, error) {
	season, err := si.IndexedSeason(imdbID, sNum)
	if err != nil {
		return "", err
	}

	// Derive path from the first episode in the season
	for _, ep := range season.Episodes {
		return filepath.Dir(ep.Path), nil
	}

	return "", ErrNotFound
}

// IndexedShow returns the indexed show from the index
func (si *ShowIndex) IndexedShow(imdbID string) (*polochon.Show, error) {
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

	// Derive the show path from the first episode
	for _, season := range show.Seasons {
		for _, ep := range season.Episodes {
			return filepath.Dir(filepath.Dir(ep.Path)), nil
		}
	}

	// Fall back to sidecar file paths when all episodes have been removed
	for _, f := range []*polochon.File{show.FanartFile, show.BannerFile, show.PosterFile, show.NFOFile} {
		if f != nil && f.Path != "" {
			return filepath.Dir(f.Path), nil
		}
	}

	return "", ErrNotFound
}

// Add adds a show episode to the index
func (si *ShowIndex) Add(episode *polochon.ShowEpisode) error {
	if episode.Name == "" {
		episode.Name = episode.Filename()
	}

	// Get the parent paths
	seasonPath := filepath.Dir(episode.Path)
	showPath := filepath.Dir(seasonPath)

	// Check if the show is in the index
	hasShow, err := si.HasShow(episode.ShowImdbID)
	if err != nil {
		return err
	}
	if !hasShow {
		show := &polochon.Show{
			ImdbID:  episode.ShowImdbID,
			Title:   episode.ShowTitle,
			Seasons: map[int]*polochon.ShowSeason{},
		}
		// Populate show sidecar files from disk
		show.FanartFile = polochon.NewSidecarFile(filepath.Join(showPath, "fanart.jpg"))
		show.BannerFile = polochon.NewSidecarFile(filepath.Join(showPath, "banner.jpg"))
		show.PosterFile = polochon.NewSidecarFile(filepath.Join(showPath, "poster.jpg"))
		show.NFOFile = polochon.NewSidecarFile(filepath.Join(showPath, "tvshow.nfo"))

		si.Lock()
		si.shows[episode.ShowImdbID] = show
		si.Unlock()
	}

	// Check if the season is in the index
	hasSeason, err := si.HasSeason(episode.ShowImdbID, episode.Season)
	if err != nil {
		return err
	}
	if !hasSeason {
		si.Lock()
		si.shows[episode.ShowImdbID].Seasons[episode.Season] = &polochon.ShowSeason{
			Season:     episode.Season,
			ShowImdbID: episode.ShowImdbID,
			Episodes:   map[int]*polochon.ShowEpisode{},
		}
		si.Unlock()
	}

	// Populate sidecar file references on the episode
	episode.NFOFile = polochon.NewSidecarFile(episode.NfoPath())

	si.Lock()
	si.shows[episode.ShowImdbID].Seasons[episode.Season].Episodes[episode.Episode] = episode
	si.Unlock()

	return nil
}

// UpsertSubtitle updates or inserts a subtitle
func (si *ShowIndex) UpsertSubtitle(e *polochon.ShowEpisode, s *polochon.Subtitle) error {
	episode, err := si.Episode(e.ShowImdbID, e.Season, e.Episode)
	if err != nil {
		return err
	}

	si.Lock()
	episode.Subtitles = upsertSubtitle(episode.Subtitles, s)
	si.Unlock()

	return nil
}

// IsShowEmpty returns true if the show has no indexed episodes
func (si *ShowIndex) IsShowEmpty(imdbID string) (bool, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return true, nil
	}

	return len(show.Seasons) == 0, nil
}

// IsSeasonEmpty returns true if the season has no indexed episodes
func (si *ShowIndex) IsSeasonEmpty(imdbID string, season int) (bool, error) {
	si.RLock()
	defer si.RUnlock()

	show, ok := si.shows[imdbID]
	if !ok {
		return true, nil
	}

	s, ok := show.Seasons[season]
	if !ok {
		return true, nil
	}

	return len(s.Episodes) == 0, nil
}

// RemoveSeason removes the season from the index
func (si *ShowIndex) RemoveSeason(show *polochon.Show, season int) error {
	si.Lock()
	defer si.Unlock()
	delete(si.shows[show.ImdbID].Seasons, season)
	return nil
}

// RemoveShow removes the show from the index
func (si *ShowIndex) RemoveShow(show *polochon.Show) error {
	si.Lock()
	defer si.Unlock()
	delete(si.shows, show.ImdbID)
	return nil
}

// RemoveEpisode removes the show episode from the index
func (si *ShowIndex) RemoveEpisode(episode *polochon.ShowEpisode) error {
	id := episode.ShowImdbID
	sNum := episode.Season
	eNum := episode.Episode

	if _, err := si.Episode(id, sNum, eNum); err != nil {
		return err
	}

	si.Lock()
	defer si.Unlock()
	delete(si.shows[id].Seasons[sNum].Episodes, eNum)

	return nil
}
