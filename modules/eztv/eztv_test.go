package eztv

import (
	"reflect"
	"testing"

	"github.com/odwrtw/eztv"
	"gitlab.quimbo.fr/odwrtw/polochon/lib"
)

func TestEztvGetTorrentsInvalidArgumens(t *testing.T) {
	eztv := &Eztv{}
	m := "invalid type"

	err := eztv.GetTorrents(m)
	if err != ErrInvalidArgument {
		t.Fatalf("Expected %q got %q", ErrInvalidArgument, err)
	}
}

func TestEztvInvalidArguments(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})

	err := e.GetTorrents(s)
	if err != ErrMissingShowImdbID {
		t.Fatalf("Expected %q got %q", ErrMissingShowImdbID, err)
	}

	s.ShowImdbID = "tt2562232"
	err = e.GetTorrents(s)
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

	err := e.GetTorrents(s)
	if err != ErrFailedToFindShowEpisode {
		t.Fatalf("Expected %q got %q", ErrFailedToFindShowEpisode, err)
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

	err := e.GetTorrents(s)
	if err != ErrNoTorrentFound {
		t.Fatalf("Expected %q got %q", ErrNoTorrentFound, err)
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

	err := e.GetTorrents(s)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	expected := []polochon.Torrent{
		{Quality: polochon.Quality480p, URL: "http://480p.torrent"},
		{Quality: polochon.Quality720p, URL: "http://720p.torrent"},
		{Quality: polochon.Quality1080p, URL: "http://1080p.torrent"},
	}

	if !reflect.DeepEqual(expected, s.Torrents) {
		t.Errorf("Failed to get torrents from eztv\nExpected %+v\nGot %+v", expected, s.Torrents)
	}
}
