package library

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Sirupsen/logrus"
	polochon "github.com/odwrtw/polochon/lib"
	index "github.com/odwrtw/polochon/lib/media_index"
)

// GetIndexedSeason returns a ShowSeason from its id
func (l *Library) GetIndexedSeason(id string, season int) (index.IndexedSeason, error) {
	s, err := l.showIndex.IndexedSeason(id, season)
	if err != nil {
		return index.IndexedSeason{}, err
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
	return l.showIndex.RemoveSeason(show, season, log)
}

func (l *Library) getSeasonDir(ep *polochon.ShowEpisode) string {
	return filepath.Join(l.ShowDir, ep.ShowTitle, fmt.Sprintf("Season %d", ep.Season))
}
