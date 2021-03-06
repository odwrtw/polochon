package library

import (
	"fmt"
	"os"
	"path/filepath"

	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
	"github.com/sirupsen/logrus"
)

// GetIndexedSeason returns a ShowSeason from its id
func (l *Library) GetIndexedSeason(id string, season int) (*index.Season, error) {
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
func (l *Library) DeleteSeason(id string, season int, log *logrus.Entry) error {
	path, err := l.showIndex.SeasonPath(id, season)
	if err != nil {
		return err
	}

	// Remove whole season
	if err := os.RemoveAll(path); err != nil {
		return err
	}

	// Remove the season from the index
	show := &polochon.Show{ImdbID: id}
	if err := l.showIndex.RemoveSeason(show, season, log); err != nil {
		return err
	}

	// Check if the show is empty
	ok, err := l.showIndex.IsShowEmpty(id)
	if err != nil {
		return err
	}
	if ok {
		// Delete the whole Show
		if err := l.DeleteShow(id, log); err != nil {
			return err
		}
	}

	return nil
}

func (l *Library) getSeasonDir(ep *polochon.ShowEpisode) string {
	return filepath.Join(l.ShowDir, ep.ShowTitle, fmt.Sprintf("Season %d", ep.Season))
}
