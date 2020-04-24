package eztv

import (
	"errors"
	"strings"

	"github.com/odwrtw/eztv"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a torrenter, an explorer and a searcher
var (
	_ polochon.Torrenter = (*Eztv)(nil)
)

// Register eztv as a Torrenter
func init() {
	polochon.RegisterModule(&Eztv{})
}

// Eztv errors
var (
	ErrInvalidArgument    = errors.New("eztv: invalid argument")
	ErrMissingShowImdbID  = errors.New("eztv: missing show imdb id")
	ErrInvalidShowEpisode = errors.New("eztv: missing show episode number")
)

// Module constants
const (
	moduleName = "eztv"
)

// Eztv is a source for show episode torrents
type Eztv struct{}

// Init implements the Module interface
func (e *Eztv) Init(p []byte) error {
	return nil
}

// Function to be overwritten during the tests
var eztvGetEpisode = func(imdbID string, season, episode int) ([]*eztv.EpisodeTorrent, error) {
	return eztv.GetEpisodeTorrents(imdbID, season, episode)
}

// Get the show infos from eztv
func (e *Eztv) getShowEpisodeDetails(s *polochon.ShowEpisode) error {
	if s.ShowImdbID == "" {
		return ErrMissingShowImdbID
	}

	if s.Episode == 0 {
		return ErrInvalidShowEpisode
	}

	episodeTorrents, err := eztvGetEpisode(s.ShowImdbID, s.Season, s.Episode)
	switch err {
	case nil:
		// continue
	case eztv.ErrEpisodeNotFound:
		return polochon.ErrShowEpisodeTorrentNotFound
	default:
		return err
	}

	if len(episodeTorrents) == 0 {
		return polochon.ErrTorrentNotFound
	}

	torrents := []*polochon.Torrent{}
	for _, t := range episodeTorrents {
		quality := getQuality(t.Filename)

		torrents = append(torrents, &polochon.Torrent{
			ImdbID:  t.ImdbID,
			Type:    polochon.TypeEpisode,
			Season:  t.Season,
			Episode: t.Episode,
			Quality: quality,
			Result: &polochon.TorrentResult{
				URL:    t.MagnetURL,
				Source: moduleName,
			},
		})
	}

	s.Torrents = torrents

	return nil
}

// Name implements the Module interface
func (e *Eztv) Name() string {
	return moduleName
}

// Status implements the Module interface
func (e *Eztv) Status() (polochon.ModuleStatus, error) {
	status := polochon.StatusOK

	// Get some torrents
	torrents, err := eztv.GetTorrents(10, 1)
	if err != nil {
		status = polochon.StatusFail
	}

	// Check if there is any results
	if len(torrents) == 0 {
		return polochon.StatusFail, polochon.ErrShowEpisodeTorrentNotFound
	}

	return status, err
}

// GetTorrents implements the Torrenter interface
func (e *Eztv) GetTorrents(i interface{}, log *logrus.Entry) error {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		return e.getShowEpisodeDetails(v)
	default:
		return ErrInvalidArgument
	}
}

// SearchTorrents implements the Torrenter interface
func (e *Eztv) SearchTorrents(s string) ([]*polochon.Torrent, error) {
	// Not yet implemented
	return nil, nil
}

// Helper to guess the quality of the torrent from its name
func getQuality(filename string) polochon.Quality {
	for _, quality := range []polochon.Quality{
		polochon.Quality480p,
		polochon.Quality720p,
		polochon.Quality1080p,
	} {
		if strings.Contains(filename, string(quality)) {
			return quality
		}
	}
	return polochon.Quality480p
}
