package eztv

import (
	"errors"

	"github.com/odwrtw/eztv"
	"github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

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

// Register eztv as a Torrenter
func init() {
	polochon.RegisterTorrenter(moduleName, NewFromRawYaml)
}

// Eztv is a source for show episode torrents
type Eztv struct{}

// New is an helper to avoid passing bytes
func New() (*Eztv, error) {
	return &Eztv{}, nil
}

// NewFromRawYaml returns a new Eztv
func NewFromRawYaml(p []byte) (polochon.Torrenter, error) {
	return New()
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

	if s.Episode == 0 {
		return ErrInvalidShowEpisode
	}

	episode, err := eztvGetEpisode(s.ShowImdbID, s.Season, s.Episode)
	switch err {
	case nil:
		// continue
	case eztv.ErrEpisodeNotFound:
		return polochon.ErrShowEpisodeTorrentNotFound
	default:
		return err
	}

	if len(episode.Torrents) == 0 {
		return polochon.ErrTorrentNotFound
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
			Source:  moduleName,
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
	return polochon.StatusNotImplemented, nil
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
