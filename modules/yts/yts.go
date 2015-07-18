package yts

import (
	"errors"
	"log"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
)

// Yts errors
var (
	ErrInvalidArgument = errors.New("yts: invalid yts argument")
	ErrNoTorrentFound  = errors.New("yts: no torrent found on yts")
)

// Register yts as a Torrenter
func init() {
	polochon.RegisterTorrenter("yts", NewYts)
}

// Yts is a source for movie torrents
type Yts struct {
	log *logrus.Entry
}

// NewYts returns a new Yts
func NewYts(params map[string]interface{}, log *logrus.Entry) (polochon.Torrenter, error) {
	return &Yts{log: log}, nil
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
func (y *Yts) GetTorrents(i interface{}) error {
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
		return ErrNoTorrentFound
	}

	// since we searched by id, there should be only one movie in the list
	ytsMovie := matches[0]

	// Check the torrent
	if len(ytsMovie.Torrents) == 0 {
		return ErrNoTorrentFound
	}

	torrents := []polochon.Torrent{}
	for _, t := range ytsMovie.Torrents {
		q, err := polochon.GetQuality(t.Quality)
		if err != nil {
			log.Printf("Invalid quality %q", t.Quality)
			continue
		}

		torrents = append(torrents, polochon.Torrent{
			URL:     t.URL,
			Quality: q,
		})
	}

	m.Torrents = torrents

	return nil
}
