package polochon

import (
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
)

// NewMovieIndex returns a new movie index
func newFakeShowIndex() *ShowIndex {
	logger := logrus.NewEntry(logrus.New())
	return &ShowIndex{
		config: nil,
		log:    logger.WithField("function", "showIndexTest"),
		ids:    map[string]map[int]map[int]string{},
		slugs:  map[string]string{},
		epIDs:  map[string]string{},
	}
}

var showIdsIndex = map[string]map[int]map[int]string{
	"tt56789": map[int]map[int]string{
		1: map[int]string{
			1: "/home/test/show/season-1/show-s01e01.mp4",
			2: "/home/test/show/season-1/show-s01e02.mp4",
		},
		2: map[int]string{
			2: "/home/test/show/season-2/show-s02e02.mp4",
		},
	},
	"tt12345": map[int]map[int]string{
		2: map[int]string{
			1: "/home/test/showBis/season-2/show-s02e01.mp4",
		},
	},
	"tt123456": map[int]map[int]string{},
	"tt22222": map[int]map[int]string{
		2: map[int]string{},
	},
	"tt0397306": map[int]map[int]string{
		9: map[int]string{
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

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}
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

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}

	type res struct {
		path string
		err  error
	}

	for s, expected := range map[string]res{
		"show-s01e01": res{
			"/home/test/show/season-1/show-s01e01.mp4",
			nil,
		},
		"show-s02e02": res{
			"/home/test/show/season-2/show-s02e02.mp4",
			nil,
		},
		"showBis-s02e01": res{
			"/home/test/showBis/season-2/showBis-s01e01.mp4",
			nil,
		},
		"showBisBis-s02e01": res{
			"",
			ErrSlugNotFound,
		},
		"show-s01e03": res{
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

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}
	for s, expected := range map[string]bool{
		"tt456789": true,
		"tt123456": true,
		"tt56789":  false,
		"tt12345":  false,
	} {
		res, err := si.isShowEmpty(s)
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

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}
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
		res, err := si.isSeasonEmpty(s.imdbID, s.season)
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

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}
	res, err := si.isSeasonEmpty("tt0397306", 9)
	if err != nil {
		t.Fatal(err)
	}
	if res != false {
		t.Fatalf("TestRemoveSeason: expected %t, got %t", false, res)
	}

	// Create a fake show and a fake show episode
	e := fakeShowEpisode()
	s := newFakeShow()
	s.Episodes = append(s.Episodes, e)
	s.ImdbID = e.ShowImdbID
	e.Path = "/home/test/show/season-1/showTers-s09e18.mp4"

	si.RemoveSeasonFromIndex(s, 9)

	res, err = si.isSeasonEmpty("tt0397306", 9)
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

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}
	res, err := si.isShowEmpty("tt0397306")
	if err != nil {
		t.Fatal(err)
	}
	if res != false {
		t.Fatalf("TestRemoveShow: expected %t, got %t", false, res)
	}

	// Create a fake show and a fake show episode
	e := fakeShowEpisode()
	s := newFakeShow()
	s.ImdbID = e.ShowImdbID
	e.Show = s
	e.Path = "/home/test/show/season-1/showTers-s09e18.mp4"

	si.RemoveShowFromIndex(s)

	res, err = si.isShowEmpty("tt0397306")
	if err != nil {
		t.Fatal(err)
	}
	if res != true {
		t.Errorf("TestRemoveShow: expected %t, got %t", true, res)
	}
}

func TestAddAndDeleteEpisodeToIndex(t *testing.T) {
	si := newFakeShowIndex()

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}
	// The index is empty

	// Create a fake show and a fake show episode
	e := fakeShowEpisode()
	s := newFakeShow()
	s.ImdbID = e.ShowImdbID
	e.Show = s
	e.Path = "/home/test/show/season-1/showTers-s09e18.mp4"

	// Add it to the index
	err := si.AddToIndex(e)
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
	err = si.RemoveFromIndex(e)
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

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}

	expectedSlugs := []string{
		"show-s01e01",
		"show-s01e02",
		"show-s02e02",
		"showBis-s02e01",
	}

	slugs, err := si.ShowSlugs()
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

	buildShowIndex = func(si *ShowIndex) error {
		return nil
	}

	expectedIDs := showIdsIndex
	ids, err := si.ShowIds()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedIDs, ids) {
		t.Errorf("TestIDs: not the same ids")
	}
}
