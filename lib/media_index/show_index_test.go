package index

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

var mockLogEntry = logrus.NewEntry(&logrus.Logger{Out: ioutil.Discard})

func mockShowIndex() ShowIndex {
	return ShowIndex{
		shows: map[string]IndexedShow{
			"tt56789": {
				Path: "/home/test/tt56789",
				Seasons: map[int]IndexedSeason{
					1: {
						Path: "/home/test/tt56789/1",
						Episodes: map[int]string{
							1: "/home/test/tt56789/1/s01e01.mp4",
							2: "/home/test/tt56789/1/s01e02.mp4",
						},
					},
					2: {
						Path: "/home/test/tt56789/2",
						Episodes: map[int]string{
							2: "/home/test/tt56789/2/s02e02.mp4",
						},
					},
				},
			},

			"tt12345": {
				Path: "/home/test/tt12345",
				Seasons: map[int]IndexedSeason{
					2: {
						Path: "/home/test/tt12345/2",
						Episodes: map[int]string{
							1: "/home/test/tt12345/2/s02e01.mp4",
						},
					},
				},
			},

			"tt0397306": {
				Path: "/home/test/tt0397306",
				Seasons: map[int]IndexedSeason{
					9: {
						Path: "/home/test/tt0397306/9",
						Episodes: map[int]string{
							18: "/home/test/tt0397306/18/s09e18.mp4",
						},
					},
				},
			},

			"tt123456": {},

			"tt22222": {
				Path: "/home/test/tt0397306",
				Seasons: map[int]IndexedSeason{
					2: {},
				},
			},
		},
	}
}

func TestHasShowEpisode(t *testing.T) {
	idx := mockShowIndex()

	for _, mock := range []struct {
		imdbID   string
		season   int
		episode  int
		expected bool
	}{
		{"tt56789", 1, 1, true},
		{"tt56789", 1, 2, true},
		{"tt56789", 1, 3, false},
		{"tt12345", 1, 3, false},
		{"tt12345", 2, 1, true},
		{"tt11111", 2, 1, false},
	} {
		got, err := idx.Has(mock.imdbID, mock.season, mock.episode)
		if err != nil {
			t.Fatalf("failed to get episode from index: %q", err)
		}

		if mock.expected != got {
			t.Errorf("expected %t, got %t for %s s%d e%d", mock.expected, got, mock.imdbID, mock.season, mock.episode)
		}
	}
}

func TestIsShowEmpty(t *testing.T) {
	idx := mockShowIndex()
	for id, expected := range map[string]bool{
		"tt456789": true,
		"tt123456": true,
		"tt56789":  false,
		"tt12345":  false,
	} {
		got, err := idx.IsShowEmpty(id)
		if err != nil {
			t.Fatalf("failed to check if the show is empty: %q", err)
		}
		if expected != got {
			t.Errorf("expected %t, got %t for %s", expected, got, id)
		}
	}
}

func TestIsSeasonEmpty(t *testing.T) {
	idx := mockShowIndex()
	for _, mock := range []struct {
		imdbID   string
		season   int
		expected bool
	}{
		{"tt12345", 2, false},
		{"tt12345", 3, true},
		{"tt22222", 2, true},
		{"tt22222", 1, true},
		{"tt56789", 1, false},
		{"tt56789", 2, false},
	} {
		got, err := idx.IsSeasonEmpty(mock.imdbID, mock.season)
		if err != nil {
			t.Fatalf("failed to check if the show season is empty: %q", err)
		}
		if mock.expected != got {
			t.Errorf("expected %t, got %t for %s and %d", mock.expected, got, mock.imdbID, mock.season)
		}
	}
}

func TestRemoveSeason(t *testing.T) {
	idx := mockShowIndex()

	id := "tt0397306"
	season := 9

	empty, err := idx.IsSeasonEmpty(id, season)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if empty {
		t.Fatal("season should not be empty")
	}

	s := &polochon.Show{ImdbID: id}
	if err := idx.RemoveSeason(s, season, mockLogEntry); err != nil {
		t.Fatalf("error while removing season from the index: %q", err)
	}

	empty, err = idx.IsSeasonEmpty(id, season)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !empty {
		t.Fatal("season should be empty")
	}
}

func TestRemoveShow(t *testing.T) {
	idx := mockShowIndex()
	id := "tt0397306"

	empty, err := idx.IsShowEmpty(id)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if empty {
		t.Fatal("show should not be empty")
	}

	s := &polochon.Show{ImdbID: id}
	if err := idx.RemoveShow(s, mockLogEntry); err != nil {
		t.Fatalf("error while removing show from the index: %q", err)
	}

	empty, err = idx.IsShowEmpty(id)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !empty {
		t.Fatal("show should be empty")
	}
}

func TestAddEpisodeToIndex(t *testing.T) {
	idx := NewShowIndex()

	expectedShowPath := "/home/shows/My show 1"
	expectedSeasonPath := filepath.Join(expectedShowPath, "Season 1")

	// Create a fake show and a fake show episode
	e := &polochon.ShowEpisode{
		ShowImdbID: "tt0397306",
		Season:     1,
		Episode:    1,
	}
	e.Path = filepath.Join(expectedSeasonPath, "My show 1 - s01e01.mp4")

	// Add it to the index
	if err := idx.Add(e); err != nil {
		t.Fatalf("error while adding show in the index: %q", err)
	}

	// Check
	hasEpisode, err := idx.Has(e.ShowImdbID, e.Season, e.Episode)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !hasEpisode {
		t.Fatal("the index should have the episode")
	}

	// Ensures the paths are correct
	showPath, err := idx.ShowPath(e.ShowImdbID)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if showPath != expectedShowPath {
		t.Errorf("expected show path to be %q, got %q", expectedShowPath, showPath)
	}

	seasonPath, err := idx.SeasonPath(e.ShowImdbID, e.Season)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if seasonPath != expectedSeasonPath {
		t.Errorf("expected season path to be %q, got %q", expectedSeasonPath, seasonPath)
	}
}
