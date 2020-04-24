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
		t.Fatalf("expected %q got %q", ErrInvalidArgument, err)
	}
}

func TestEztvInvalidArguments(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})

	err := e.GetTorrents(s, fakeLogEntry)
	if err != ErrMissingShowImdbID {
		t.Fatalf("expected %q got %q", ErrMissingShowImdbID, err)
	}

	s.ShowImdbID = "tt2562232"
	err = e.GetTorrents(s, fakeLogEntry)
	if err != ErrInvalidShowEpisode {
		t.Fatalf("expected %q got %q", ErrInvalidShowEpisode, err)
	}
}

func TestEztvNoShowEpisodeFound(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.ShowImdbID = "tt2562232"
	s.Season = 1
	s.Episode = 1

	eztvGetEpisode = func(imdbID string, season, episode int) ([]*eztv.EpisodeTorrent, error) {
		return nil, eztv.ErrEpisodeNotFound
	}

	err := e.GetTorrents(s, fakeLogEntry)
	if err != polochon.ErrShowEpisodeTorrentNotFound {
		t.Fatalf("expected %q got %q", polochon.ErrShowEpisodeTorrentNotFound, err)
	}
}

func TestEztvNoTorrentFound(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.ShowImdbID = "tt2562232"
	s.Season = 1
	s.Episode = 1

	eztvGetEpisode = func(imdbID string, season, episode int) ([]*eztv.EpisodeTorrent, error) {
		return []*eztv.EpisodeTorrent{}, nil
	}

	err := e.GetTorrents(s, fakeLogEntry)
	if err != polochon.ErrTorrentNotFound {
		t.Fatalf("expected %q got %q", polochon.ErrTorrentNotFound, err)
	}
}

func TestEztvGetTorrents(t *testing.T) {
	e := &Eztv{}
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.ShowImdbID = "tt2562232"
	s.Season = 1
	s.Episode = 1

	eztvGetEpisode = func(imdbID string, season, episode int) ([]*eztv.EpisodeTorrent, error) {
		return []*eztv.EpisodeTorrent{
			{
				ID:        2,
				ImdbID:    s.ShowImdbID,
				Season:    s.Season,
				Episode:   s.Episode,
				Hash:      "yoshi",
				Filename:  "pwet.mkv",
				MagnetURL: "magnet:?xt=urn:btih:yoshi2",
			},
			{
				ID:        1,
				ImdbID:    s.ShowImdbID,
				Season:    s.Season,
				Episode:   s.Episode,
				Hash:      "yoshi",
				Filename:  "pwet.720p.mkv",
				MagnetURL: "magnet:?xt=urn:btih:yoshi1",
			},
			{
				ID:        2,
				ImdbID:    s.ShowImdbID,
				Season:    s.Season,
				Episode:   s.Episode,
				Hash:      "yoshi",
				Filename:  "pwet.1080p.mkv",
				MagnetURL: "magnet:?xt=urn:btih:yoshi3",
			},
		}, nil
	}

	err := e.GetTorrents(s, fakeLogEntry)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	expected := []*polochon.Torrent{
		{
			ImdbID:  s.ShowImdbID,
			Type:    polochon.TypeEpisode,
			Season:  s.Season,
			Episode: s.Episode,
			Quality: polochon.Quality480p,
			Result: &polochon.TorrentResult{
				Source: "eztv",
				URL:    "magnet:?xt=urn:btih:yoshi2",
			},
		},
		{
			ImdbID:  s.ShowImdbID,
			Type:    polochon.TypeEpisode,
			Season:  s.Season,
			Episode: s.Episode,
			Quality: polochon.Quality720p,
			Result: &polochon.TorrentResult{
				Source: "eztv",
				URL:    "magnet:?xt=urn:btih:yoshi1",
			},
		},
		{
			ImdbID:  s.ShowImdbID,
			Type:    polochon.TypeEpisode,
			Season:  s.Season,
			Episode: s.Episode,
			Quality: polochon.Quality1080p,
			Result: &polochon.TorrentResult{
				Source: "eztv",
				URL:    "magnet:?xt=urn:btih:yoshi3",
			},
		},
	}

	if !reflect.DeepEqual(expected, s.Torrents) {
		t.Errorf("failed to get torrents from eztv\nexpected %+v\ngot %+v", expected, s.Torrents)
	}
}
