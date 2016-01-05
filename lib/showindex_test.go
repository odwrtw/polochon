package polochon

import (
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
)

// NewMovieIndex returns a new movie index
func newFakeShowIndex() *ShowIndex {
	log := logrus.NewEntry(&logrus.Logger{Out: ioutil.Discard})
	return &ShowIndex{
		log:   log.WithField("function", "showIndexTest"),
		ids:   map[string]map[int]map[int]string{},
		slugs: map[string]string{},
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

var showSlugsIndex = map[string]string{
	"show-s01e01":         "/home/test/show/season-1/show-s01e01.mp4",
	"show-s01e02":         "/home/test/show/season-1/show-s01e02.mp4",
	"show-s02e02":         "/home/test/show/season-2/show-s02e02.mp4",
	"showBis-s02e01":      "/home/test/showBis/season-2/showBis-s01e01.mp4",
	"american-dad-s09e18": "/home/test/show/season-1/showTers-s09e18.mp4",
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

func TestSearchShowBySlug(t *testing.T) {
	si := newFakeShowIndex()

	si.slugs = showSlugsIndex

	type res struct {
		path string
		err  error
	}

	for s, expected := range map[string]res{
		"show-s01e01": {
			"/home/test/show/season-1/show-s01e01.mp4",
			nil,
		},
		"show-s02e02": {
			"/home/test/show/season-2/show-s02e02.mp4",
			nil,
		},
		"showBis-s02e01": {
			"/home/test/showBis/season-2/showBis-s01e01.mp4",
			nil,
		},
		"showBisBis-s02e01": {
			"",
			ErrSlugNotFound,
		},
		"show-s01e03": {
			"",
			ErrSlugNotFound,
		},
	} {
		res, err := si.searchShowEpisodeBySlug(s)
		if expected.path != res {
			t.Errorf("TestSearchBySlug: expected %s, got %s for %s", expected.path, res, s)
		}
		if expected.err != err {
			t.Errorf("TestSearchBySlug: expected error %s, got %s for %s", expected.err, err, s)
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
	si.slugs = showSlugsIndex

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

	si.RemoveSeason(s, 9)

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

	si.RemoveShow(s)

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
	err = si.Remove(e)
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

func TestShowSlugs(t *testing.T) {
	si := newFakeShowIndex()

	si.slugs = showSlugsIndex

	expectedSlugs := []string{
		"show-s01e01",
		"show-s01e02",
		"show-s02e02",
		"showBis-s02e01",
	}

	slugs, err := si.Slugs()
	if err != nil {
		t.Fatal(err)
	}
LOOP:
	for _, exp := range expectedSlugs {
		for _, s := range slugs {
			// if we found the element, go to the next one
			if exp == s {
				continue LOOP
			}
		}
		t.Errorf("TestIDs: %s is not in the result", exp)
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
