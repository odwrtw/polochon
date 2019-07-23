package yts

import (
	"errors"

	"github.com/dustin/go-humanize"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
)

// Make sure that the module is a torrenter, an explorer and a searcher
var (
	_ polochon.Torrenter = (*Yts)(nil)
	_ polochon.Explorer  = (*Yts)(nil)
	_ polochon.Searcher  = (*Yts)(nil)
)

func init() {
	polochon.RegisterModule(&Yts{})
}

// Yts errors
var (
	ErrInvalidArgument = errors.New("yts: invalid yts argument")
)

// Module constants
const (
	moduleName = "yts"
)

// Yts is a source for movie torrents
type Yts struct{}

// Init implements the module interface
func (y *Yts) Init(p []byte) error {
	return nil
}

// Name implements the Module interface
func (y *Yts) Name() string {
	return moduleName
}

// Status implements the Module interface
func (y *Yts) Status() (polochon.ModuleStatus, error) {
	status := polochon.StatusOK
	err := yts.Status()
	if err != nil {
		status = polochon.StatusFail
	}

	return status, err
}

// Ensure that the given interface is an Movie
func getMovieArgument(i interface{}) (*polochon.Movie, error) {
	if m, ok := i.(*polochon.Movie); ok {
		return m, nil
	}

	return nil, ErrInvalidArgument
}

// searchByImdbID is a function made to be overwitten during the tests
var searchByImdbID = func(imdbID string) ([]yts.Movie, error) {
	return yts.Search(imdbID)
}

func polochonTorrents(m *yts.Movie) []polochon.Torrent {
	if m.Torrents == nil || len(m.Torrents) == 0 {
		return nil
	}

	torrents := []polochon.Torrent{}
	for _, t := range m.Torrents {
		q := polochon.Quality(t.Quality)
		if !q.IsAllowed() {
			continue
		}

		size, _ := humanize.ParseBytes(t.Size)

		torrents = append(torrents, polochon.Torrent{
			Name:     m.Title,
			URL:      t.URL,
			Quality:  q,
			Source:   moduleName,
			Seeders:  t.Seeds,
			Leechers: t.Peers,
			Size:     int(size),
		})
	}

	return torrents
}
