package eztv

import (
	"errors"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/eztv"
	"github.com/odwrtw/polochon/lib"
)

// Eztv errors
var (
	ErrFailedToFindShowEpisode = errors.New("eztv: failed to find show episode")
	ErrInvalidArgument         = errors.New("eztv: invalid argument")
	ErrNoTorrentFound          = errors.New("eztv: failed to find torrent")
	ErrMissingShowImdbID       = errors.New("eztv: missing show imdb id")
	ErrInvalidShowEpisode      = errors.New("eztv: missing show episode or season")
)

// Module constants
const (
	moduleName = "eztv"
)

// Register yts as a Torrenter
func init() {
	polochon.RegisterTorrenter(moduleName, NewEztv)
}

// Eztv is a source for show episode torrents
type Eztv struct {
	log *logrus.Entry
}

// NewEztv returns a new Eztv
func NewEztv(params map[string]interface{}, log *logrus.Entry) (polochon.Torrenter, error) {
	return &Eztv{log: log}, nil
}

// Function to be overwritten during the tests
var eztvGetEpisode = func(imdbID string, season, episode int) (*eztv.ShowEpisode, error) {
	return eztv.GetEpisode(imdbID, season, episode)
}

// Get the show infos from eztv
func (e *Eztv) getShowEpisodeDetails(s *polochon.ShowEpisode) error {
	if s.ShowImdbID == "" {
		return ErrMissingShowImdbID
	}

	if s.Season == 0 || s.Episode == 0 {
		return ErrInvalidShowEpisode
	}

	episode, err := eztvGetEpisode(s.ShowImdbID, s.Season, s.Episode)
	switch err {
	case nil:
		// continue
	case eztv.ErrEpisodeNotFound:
		return ErrFailedToFindShowEpisode
	default:
		return err
	}

	if len(episode.Torrents) == 0 {
		return ErrNoTorrentFound
	}

	torrents := []polochon.Torrent{}
	for _, quality := range []polochon.Quality{
		polochon.Quality480p,
		polochon.Quality720p,
		polochon.Quality1080p,
	} {
		torrent, ok := episode.Torrents[string(quality)]
		if !ok {
			continue
		}

		torrents = append(torrents, polochon.Torrent{
			Quality: quality,
			URL:     torrent.URL,
		})
	}

	s.Torrents = torrents

	return nil
}

// Name implements the Module interface
func (e *Eztv) Name() string {
	return moduleName
}

// GetTorrents implements the Torrenter interface
func (e *Eztv) GetTorrents(i interface{}) error {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		return e.getShowEpisodeDetails(v)
	default:
		return ErrInvalidArgument
	}
}
