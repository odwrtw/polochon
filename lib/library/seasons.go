package library

import (
	"fmt"
	"os"
	"path/filepath"

	polochon "github.com/odwrtw/polochon/lib"
)

// GetIndexedSeason returns an indexed ShowSeason
func (l *Library) GetIndexedSeason(id string, season int) (*polochon.ShowSeason, error) {
	s, err := l.showIndex.IndexedSeason(id, season)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// GetSeason returns a ShowSeason from its id
func (l *Library) GetSeason(id string, season int) (*polochon.ShowSeason, error) {
	_, err := l.showIndex.SeasonPath(id, season)
	if err != nil {
		return nil, err
	}

	s := polochon.NewShowSeason(l.showConfig)
	s.Season = season
	s.ShowImdbID = id

	return s, nil
}

// DeleteSeason deletes a season
func (l *Library) DeleteSeason(id string, season int) error {
	log := l.log.With("type", "show_season", "imdb_id", id, "season", season)

	// Capture show path before modifying the index (while episodes still exist).
	// Falls back to sidecar file paths if all episodes were already removed.
	showPath, _ := l.showIndex.ShowPath(id)

	path, err := l.showIndex.SeasonPath(id, season)
	if err != nil {
		// If the season has no episodes (already removed), derive from show path.
		if showPath == "" {
			return err
		}
		path = filepath.Join(showPath, fmt.Sprintf("Season %d", season))
	}

	// Remove whole season
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	log.Info("removing season from index")
	// Remove the season from the index
	show := &polochon.Show{ImdbID: id}
	if err := l.showIndex.RemoveSeason(show, season); err != nil {
		return err
	}

	// Check if the show is empty
	ok, err := l.showIndex.IsShowEmpty(id)
	if err != nil {
		return err
	}
	if ok && showPath != "" {
		// Delete the whole show directory and remove from index
		log.Info("removing show")
		if err := os.RemoveAll(showPath); err != nil {
			return err
		}
		return l.showIndex.RemoveShow(show)
	}

	return nil
}

func (l *Library) getSeasonDir(ep *polochon.ShowEpisode) string {
	return filepath.Join(l.ShowDir, ep.ShowTitle, fmt.Sprintf("Season %d", ep.Season))
}
