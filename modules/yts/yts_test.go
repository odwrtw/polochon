package yts

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yts"
	"github.com/sirupsen/logrus"
)

var fakeLogEntry = logrus.NewEntry(logrus.New())

func TestYtsBadInput(t *testing.T) {
	y := &Yts{}
	show := polochon.NewShowEpisode(polochon.ShowConfig{})

	err := y.GetTorrents(show, fakeLogEntry)
	if err != ErrInvalidArgument {
		t.Errorf("Got %q, expected %q", err, ErrInvalidArgument)
	}
}

func TestYtsNoResults(t *testing.T) {
	y := &Yts{}
	m := polochon.NewMovie(polochon.MovieConfig{})

	searchByImdbID = func(imdbID string) ([]yts.Movie, error) {
		return []yts.Movie{}, nil
	}

	err := y.GetTorrents(m, fakeLogEntry)
	if err != polochon.ErrTorrentNotFound {
		t.Errorf("Got %q, expected %q", err, polochon.ErrTorrentNotFound)
	}
}

func TestYtsNoTorrent(t *testing.T) {
	y := &Yts{}
	m := polochon.NewMovie(polochon.MovieConfig{})

	searchByImdbID = func(imdbID string) ([]yts.Movie, error) {
		return []yts.Movie{
			{},
		}, nil
	}

	err := y.GetTorrents(m, fakeLogEntry)
	if err != polochon.ErrTorrentNotFound {
		t.Errorf("Got %q, expected %q", err, polochon.ErrTorrentNotFound)
	}
}

func TestYtsTorrents(t *testing.T) {
	y := &Yts{}
	m := polochon.NewMovie(polochon.MovieConfig{})
	m.ImdbID = "tt0000"

	searchByImdbID = func(imdbID string) ([]yts.Movie, error) {
		return []yts.Movie{
			{
				ImdbID: imdbID,
				Torrents: []yts.Torrent{
					{Quality: "480p", URL: "http://test.480p.magnet"},
					{Quality: "720p", URL: "http://test.720p.magnet"},
					{Quality: "1080p", URL: "http://test.1080p.magnet"},
				},
			},
		}, nil
	}

	err := y.GetTorrents(m, fakeLogEntry)
	if err != nil {
		t.Fatal(err)
	}

	expected := polochon.NewMovie(polochon.MovieConfig{})
	expected.ImdbID = "tt0000"
	expected.Torrents = []*polochon.Torrent{
		{
			ImdbID:  "tt0000",
			Type:    polochon.TypeMovie,
			Quality: polochon.Quality480p,
			Result: &polochon.TorrentResult{
				Source: "yts",
				URL:    "http://test.480p.magnet",
			},
		},
		{
			ImdbID:  "tt0000",
			Type:    polochon.TypeMovie,
			Quality: polochon.Quality720p,
			Result: &polochon.TorrentResult{
				Source: "yts",
				URL:    "http://test.720p.magnet",
			},
		},
		{
			ImdbID:  "tt0000",
			Type:    polochon.TypeMovie,
			Quality: polochon.Quality1080p,
			Result: &polochon.TorrentResult{
				Source: "yts",
				URL:    "http://test.1080p.magnet",
			},
		},
	}

	if !reflect.DeepEqual(m, expected) {
		t.Errorf("Torrents not well parsed from yts")
	}
}
