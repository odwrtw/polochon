package yts

import (
	"errors"

	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
	"github.com/sirupsen/logrus"
)

// Yts errors
var (
	ErrInvalidArgument = errors.New("yts: invalid yts argument")
)

// Module constants
const (
	moduleName = "yts"
)

func init() {
	// Register yts as a Torrenter
	polochon.RegisterTorrenter(moduleName, NewFromRawYaml)
	// Register yts as an Explorer
	polochon.RegisterExplorer(moduleName, NewExplorer)
	// Register yts as a Searcher
	polochon.RegisterSearcher(moduleName, NewSearcher)
}

// Yts is a source for movie torrents
type Yts struct{}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Torrenter, error) {
	return New()
}

// New returns a new Yts
// func New() (polochon.Torrenter, error) {
func New() (*Yts, error) {
	return &Yts{}, nil
}

// Name implements the Module interface
func (y *Yts) Name() string {
	return moduleName
}

// Ensure that the given interface is an Movie
func (y *Yts) getMovieArgument(i interface{}) (*polochon.Movie, error) {
	if m, ok := i.(*polochon.Movie); ok {
		return m, nil
	}

	return nil, ErrInvalidArgument
}

// searchByImdbID is a function made to be overwitten during the tests
var searchByImdbID = func(imdbID string) ([]yts.Movie, error) {
	return yts.Search(imdbID)
}

// GetTorrents implements the Torrenter interface
func (y *Yts) GetTorrents(i interface{}, log *logrus.Entry) error {
	m, err := y.getMovieArgument(i)
	if err != nil {
		return err
	}

	// matches returns a list of matching movie
	matches, err := searchByImdbID(m.ImdbID)
	if err != nil {
		return err
	}

	if len(matches) == 0 {
		return polochon.ErrMovieTorrentNotFound
	}

	// since we searched by id, there should be only one movie in the list
	ytsMovie := matches[0]

	// Check the torrent
	if len(ytsMovie.Torrents) == 0 {
		return polochon.ErrMovieTorrentNotFound
	}

	torrents := []polochon.Torrent{}
	for _, t := range ytsMovie.Torrents {
		q := polochon.Quality(t.Quality)
		if !q.IsAllowed() {
			continue
		}

		torrents = append(torrents, polochon.Torrent{
			URL:      t.URL,
			Quality:  q,
			Source:   moduleName,
			Seeders:  t.Seeds,
			Leechers: t.Peers,
		})
	}

	m.Torrents = torrents

	return nil
}

// SearchTorrents implements the Torrenter interface
func (y *Yts) SearchTorrents(s string) ([]*polochon.Torrent, error) {
	torrents := []*polochon.Torrent{}

	movies, err := yts.Search(s)
	if err != nil {
		return nil, err
	}

	for _, m := range movies {
		if m.Torrents == nil {
			continue
		}

		for _, t := range m.Torrents {
			q := polochon.Quality(t.Quality)
			if !q.IsAllowed() {
				continue
			}

			torrents = append(torrents, &polochon.Torrent{
				Name:     m.Title,
				Quality:  q,
				URL:      t.URL,
				Seeders:  t.Seeds,
				Leechers: t.Peers,
				Source:   moduleName,
			})
		}
	}

	return torrents, nil
}
