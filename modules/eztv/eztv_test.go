package eztv

import (
	"reflect"
	"strings"
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
	if err != polochon.ErrTorrentNotFound {
		t.Fatalf("expected %q got %q", polochon.ErrTorrentNotFound, err)
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

func TestEztvInit(t *testing.T) {
	for _, test := range []struct {
		name     string
		params   []byte
		expected []string
	}{
		{
			name: "single item",
			params: []byte(strings.Join([]string{
				"exclude_torrents_containing:",
				"- x265",
			}, "\n")),
			expected: []string{"x265"},
		},
		{
			name:     "no data",
			params:   []byte{},
			expected: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := &Eztv{}
			err := e.Init(test.params)
			if err != nil {
				t.Fatalf("unexpected error on init %+v", err)
			}
			if !reflect.DeepEqual(test.expected, e.ExcludeTorrentsContaining) {
				t.Fatalf("expected %+v, got %+v", test.expected, e.ExcludeTorrentsContaining)
			}
		})
	}
}

func TestEztvExcludedTorrents(t *testing.T) {
	eztvNOx265 := &Eztv{
		ExcludeTorrentsContaining: []string{"x265"},
	}
	eztvNOsubbed := &Eztv{
		ExcludeTorrentsContaining: []string{"subbed"},
	}
	eztvNOcam := &Eztv{
		ExcludeTorrentsContaining: []string{"cam"},
	}
	eztvNOfrench := &Eztv{
		ExcludeTorrentsContaining: []string{
			"[fr]", "french", "canada", "canadian",
		},
	}

	for _, test := range []struct {
		filename string
		x265     bool
		subbed   bool
		french   bool
		cam      bool
	}{
		{"pwet-[CAM].avi", false, false, false, true},
		{"pwet-x265.avi", true, false, false, false},
		{"pwet-x265-CANADIAN-CAMHD.avi", true, false, true, true},
		{"pwet-SUBBED-[FR].avi", false, true, true, false},
	} {
		t.Run(test.filename, func(t *testing.T) {
			got := eztvNOx265.isTorrentExcluded(test.filename)
			if test.x265 != got {
				t.Fatalf("x265? expected %+v, got %+v", test.x265, got)
			}
			got = eztvNOsubbed.isTorrentExcluded(test.filename)
			if test.subbed != got {
				t.Fatalf("subbed? expected %+v, got %+v", test.subbed, got)
			}
			got = eztvNOfrench.isTorrentExcluded(test.filename)
			if test.french != got {
				t.Fatalf("french? expected %+v, got %+v", test.french, got)
			}
			got = eztvNOcam.isTorrentExcluded(test.filename)
			if test.cam != got {
				t.Fatalf("cam? expected %+v, got %+v", test.cam, got)
			}
		})
	}
}
