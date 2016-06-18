package polochon

import (
	"reflect"
	"testing"
)

// NewMovieIndex returns a new movie index
func newFakeShowIndex() *ShowIndex {
	return &ShowIndex{
		ids: map[string]map[int]map[int]string{},
	}
}

var showIdsIndex = map[string]map[int]map[int]string{
	"tt56789": {
		1: {
			1: "/home/test/show/season-1/show-s01e01.mp4",
			2: "/home/test/show/season-1/show-s01e02.mp4",
		},
		2: {
			2: "/home/test/show/season-2/show-s02e02.mp4",
		},
	},
	"tt12345": {
		2: {
			1: "/home/test/showBis/season-2/show-s02e01.mp4",
		},
	},
	"tt123456": {},
	"tt22222": {
		2: {},
	},
	"tt0397306": {
		9: {
			18: "/home/test/show/season-1/showTers-s09e18.mp4",
		},
	},
}

func TestHasShowEpisode(t *testing.T) {
	s := newFakeShowIndex()

	s.ids = showIdsIndex

	type has struct {
		imdbID  string
		season  int
		episode int
	}
	for h, expected := range map[has]bool{
		has{"tt56789", 1, 1}: true,
		has{"tt56789", 1, 2}: true,
		has{"tt56789", 1, 3}: false,
		has{"tt12345", 1, 3}: false,
		has{"tt12345", 2, 1}: true,
		has{"tt11111", 2, 1}: false,
	} {
		res, err := s.Has(h.imdbID, h.season, h.episode)
		if err != nil {
			t.Fatal(err)
		}
		if expected != res {
			t.Errorf("TestHasMovie: expected %t, got %t for %s s%d e%d", expected, res, h.imdbID, h.season, h.episode)
		}
	}
}

func TestIsShowEmpty(t *testing.T) {
	si := newFakeShowIndex()

	si.ids = showIdsIndex
	for s, expected := range map[string]bool{
		"tt456789": true,
		"tt123456": true,
		"tt56789":  false,
		"tt12345":  false,
	} {
		res, err := si.IsShowEmpty(s)
		if err != nil {
			t.Fatal(err)
		}
		if expected != res {
			t.Errorf("TestIsShowEmpty: expected %t, got %t for %s", expected, res, s)
		}
	}
}

func TestIsSeasonEmpty(t *testing.T) {
	si := newFakeShowIndex()

	si.ids = showIdsIndex

	type test struct {
		imdbID string
		season int
	}
	for s, expected := range map[test]bool{
		test{"tt12345", 2}: false,
		test{"tt12345", 3}: true,
		test{"tt22222", 2}: true,
		test{"tt22222", 1}: true,
		test{"tt56789", 1}: false,
		test{"tt56789", 2}: false,
	} {
		res, err := si.IsSeasonEmpty(s.imdbID, s.season)
		if err != nil {
			t.Fatal(err)
		}
		if expected != res {
			t.Errorf("TestIsShowEmpty: expected %t, got %t for %s and %d", expected, res, s.imdbID, s.season)
		}
	}
}

func TestRemoveSeason(t *testing.T) {
	si := newFakeShowIndex()

	si.ids = showIdsIndex

	res, err := si.IsSeasonEmpty("tt0397306", 9)
	if err != nil {
		t.Fatal(err)
	}
	if res != false {
		t.Fatalf("TestRemoveSeason: expected %t, got %t", false, res)
	}

	// Create a fake show and a fake show episode
	e := mockShowEpisode()
	s := mockShow()
	s.Episodes = append(s.Episodes, e)
	s.ImdbID = e.ShowImdbID
	e.Path = "/home/test/show/season-1/showTers-s09e18.mp4"

	si.RemoveSeason(s, 9, mockLogEntry)

	res, err = si.IsSeasonEmpty("tt0397306", 9)
	if err != nil {
		t.Fatal(err)
	}
	if res != true {
		t.Errorf("TestRemoveSeason: expected %t, got %t", true, res)
	}
}

func TestRemoveShow(t *testing.T) {
	si := newFakeShowIndex()

	si.ids = showIdsIndex

	res, err := si.IsShowEmpty("tt0397306")
	if err != nil {
		t.Fatal(err)
	}
	if res != false {
		t.Fatalf("TestRemoveShow: expected %t, got %t", false, res)
	}

	// Create a fake show and a fake show episode
	e := mockShowEpisode()
	s := mockShow()
	s.ImdbID = e.ShowImdbID
	e.Show = s
	e.Path = "/home/test/show/season-1/showTers-s09e18.mp4"

	si.RemoveShow(s, mockLogEntry)

	res, err = si.IsShowEmpty("tt0397306")
	if err != nil {
		t.Fatal(err)
	}
	if res != true {
		t.Errorf("TestRemoveShow: expected %t, got %t", true, res)
	}
}

func TestAddAndDeleteEpisodeToIndex(t *testing.T) {
	si := newFakeShowIndex()

	// Create a fake show and a fake show episode
	e := mockShowEpisode()
	s := mockShow()
	s.ImdbID = e.ShowImdbID
	e.Show = s
	e.Path = "/home/test/show/season-1/showTers-s09e18.mp4"

	// Add it to the index
	err := si.Add(e)
	if err != nil {
		t.Fatal(err)
	}

	// Check
	res, err := si.Has(e.ShowImdbID, e.Season, e.Episode)
	if err != nil {
		t.Fatal(err)
	}
	if res != true {
		t.Errorf("Should have the episode %+v in index", e)
	}

	// Remove it
	err = si.Remove(e, mockLogEntry)
	if err != nil {
		t.Fatal(err)
	}

	// Check
	res, err = si.Has(e.ShowImdbID, e.Season, e.Episode)
	if err != nil {
		t.Fatal(err)
	}
	if res != false {
		t.Errorf("Should not have the episode %+v in index", e)
	}
}

func TestShowIDs(t *testing.T) {
	si := newFakeShowIndex()

	si.ids = showIdsIndex

	expectedIDs := showIdsIndex
	ids, err := si.IDs()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedIDs, ids) {
		t.Errorf("TestIDs: not the same ids")
	}
}
