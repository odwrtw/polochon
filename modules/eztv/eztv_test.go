package eztv

import (
	"reflect"
	"testing"

	"github.com/odwrtw/eztv"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

var fakeLogEntry = logrus.NewEntry(logrus.New())

func TestEztvGetTorrentsInvalidArgumens(t *testing.T) {
	eztv := &Eztv{}
	m := "invalid type"

	err := eztv.GetTorrents(m, fakeLogEntry)
	if err != ErrInvalidArgument {
		t.Fatalf("Expected %q got %q", ErrInvalidArgument, err)
	}
}

func TestEztvInvalidArguments(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})

	err := e.GetTorrents(s, fakeLogEntry)
	if err != ErrMissingShowImdbID {
		t.Fatalf("Expected %q got %q", ErrMissingShowImdbID, err)
	}

	s.ShowImdbID = "tt2562232"
	err = e.GetTorrents(s, fakeLogEntry)
	if err != ErrInvalidShowEpisode {
		t.Fatalf("Expected %q got %q", ErrInvalidShowEpisode, err)
	}
}

func TestEztvNoShowEpisodeFound(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.ShowImdbID = "tt2562232"
	s.Season = 1
	s.Episode = 1

	eztvGetEpisode = func(imdbID string, season, episode int) (*eztv.ShowEpisode, error) {
		return nil, eztv.ErrEpisodeNotFound
	}

	err := e.GetTorrents(s, fakeLogEntry)
	if err != polochon.ErrShowEpisodeTorrentNotFound {
		t.Fatalf("Expected %q got %q", polochon.ErrShowEpisodeTorrentNotFound, err)
	}
}

func TestEztvNoTorrentFound(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.ShowImdbID = "tt2562232"
	s.Season = 1
	s.Episode = 1

	eztvGetEpisode = func(imdbID string, season, episode int) (*eztv.ShowEpisode, error) {
		return &eztv.ShowEpisode{}, nil
	}

	err := e.GetTorrents(s, fakeLogEntry)
	if err != polochon.ErrTorrentNotFound {
		t.Fatalf("Expected %q got %q", polochon.ErrTorrentNotFound, err)
	}
}

func TestEztvGetTorrents(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.ShowImdbID = "tt2562232"
	s.Season = 1
	s.Episode = 1

	eztvGetEpisode = func(imdbID string, season, episode int) (*eztv.ShowEpisode, error) {
		return &eztv.ShowEpisode{
			Torrents: map[string]*eztv.ShowTorrent{
				"0":     {URL: "http://0.torrent"},
				"480p":  {URL: "http://480p.torrent"},
				"720p":  {URL: "http://720p.torrent"},
				"1080p": {URL: "http://1080p.torrent"},
			},
		}, nil
	}

	err := e.GetTorrents(s, fakeLogEntry)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	expected := []*polochon.Torrent{
		{Source: "eztv", Quality: polochon.Quality480p, URL: "http://480p.torrent"},
		{Source: "eztv", Quality: polochon.Quality720p, URL: "http://720p.torrent"},
		{Source: "eztv", Quality: polochon.Quality1080p, URL: "http://1080p.torrent"},
	}

	if !reflect.DeepEqual(expected, s.Torrents) {
		t.Errorf("Failed to get torrents from eztv\nExpected %+v\nGot %+v", expected, s.Torrents)
	}
}
