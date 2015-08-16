package polochon

import (
	"log"
	"testing"

	"github.com/Sirupsen/logrus"
)

// NewMovieIndex returns a new movie index
func newFakeMovieIndex() *MovieIndex {
	logger := logrus.NewEntry(logrus.New())
	return &MovieIndex{
		config: nil,
		log:    logger.WithField("function", "movieIndexTest"),
		ids:    map[string]string{},
		slugs:  map[string]string{},
	}
}

var idsIndex = map[string]string{
	"tt56789": "/home/test/movie/movie.mp4",
	"tt12345": "/home/test/movieBis/movieBis.mp4",
}

var slugsIndex = map[string]string{
	"movie":    "/home/test/movie/movie.mp4",
	"movieBis": "/home/test/movieBis/movieBis.mp4",
}

func TestHasMovie(t *testing.T) {
	m := newFakeMovieIndex()

	m.ids = idsIndex

	buildMovieIndex = func(mo *MovieIndex) error {
		return nil
	}

	for i, expected := range map[string]bool{
		"tt56789": true,
		"tt12345": true,
		"tt1234":  false,
	} {
		res, err := m.Has(i)
		if err != nil {
			t.Fatal(err)
		}
		if expected != res {
			t.Errorf("TestHasMovie: expected %t, got %t for %s", expected, res, i)
		}
	}
}

func TestSearchMovieBySlug(t *testing.T) {
	m := newFakeMovieIndex()

	m.slugs = slugsIndex

	buildMovieIndex = func(mo *MovieIndex) error {
		return nil
	}

	type res struct {
		path string
		err  error
	}

	for s, expected := range map[string]res{
		"movie": res{
			"/home/test/movie/movie.mp4",
			nil,
		},
		"movieBis": res{
			"/home/test/movieBis/movieBis.mp4",
			nil,
		},
		"movieDoubleBis": res{
			"",
			ErrSlugNotFound,
		},
	} {
		res, err := m.searchMovieBySlug(s)
		if expected.path != res {
			t.Errorf("TestSearchBySlug: expected %s, got %s for %s", expected.path, res, s)
		}
		if expected.err != err {
			t.Errorf("TestSearchBySlug: expected error %s, got %s for %s", expected.err, err, s)
		}
	}
}

func TestSearchMovieByImdbID(t *testing.T) {
	m := newFakeMovieIndex()

	m.ids = idsIndex

	buildMovieIndex = func(mo *MovieIndex) error {
		return nil
	}

	type res struct {
		path string
		err  error
	}

	for i, expected := range map[string]res{
		"tt12345": res{
			"/home/test/movieBis/movieBis.mp4",
			nil,
		},
		"tt56789": res{
			"/home/test/movie/movie.mp4",
			nil,
		},
		"tt1234": res{
			"",
			ErrImdbIDNotFound,
		},
	} {
		res, err := m.searchMovieByImdbID(i)
		if expected.path != res {
			t.Errorf("TestSearchByImdbID: expected %s, got %s for %s", expected.path, res, i)
		}
		if expected.err != err {
			t.Errorf("TestSearchByImdbID: expected error %s, got %s for %s", expected.err, err, i)
		}
	}
}

func TestAddAndRemoveMovieToIndex(t *testing.T) {
	mi := newFakeMovieIndex()

	buildMovieIndex = func(mo *MovieIndex) error {
		return nil
	}
	// The index is empty

	m := newFakeMovie()
	m.Path = "/home/test/movie/movie.mp4"
	err := mi.AddToIndex(m)
	if err != nil {
		t.Fatal(err)
	}

	res, err := mi.Has(m.ImdbID)
	if err != nil {
		t.Fatal(err)
	}
	if res != true {
		log.Println("index : ", mi.ids)
		t.Errorf("Should have the movie %s in index", m.ImdbID)
	}

	err = mi.RemoveFromIndex(m)
	if err != nil {
		t.Fatal(err)
	}

	res, err = mi.Has(m.ImdbID)
	if err != nil {
		t.Fatal(err)
	}
	if res != false {
		t.Errorf("Should not have the movie %s in index", m.ImdbID)
	}
}

func TestMovieSlugs(t *testing.T) {
	m := newFakeMovieIndex()

	m.slugs = slugsIndex

	buildMovieIndex = func(mo *MovieIndex) error {
		return nil
	}

	expectedSlugs := []string{"movie", "movieBis"}
	slugs, err := m.MovieSlugs()
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

func TestMovieIDs(t *testing.T) {
	m := newFakeMovieIndex()

	m.ids = idsIndex

	buildMovieIndex = func(mo *MovieIndex) error {
		return nil
	}

	expectedIDs := []string{"tt56789", "tt12345"}
	ids, err := m.MovieIds()
	if err != nil {
		t.Fatal(err)
	}
	if len(expectedIDs) != len(ids) {
		t.Errorf("TestIDs: not the same number of elements in the result")
	}
LOOP:
	for _, exp := range expectedIDs {
		for _, i := range ids {
			// if we found the element, go to the next one
			if exp == i {
				continue LOOP
			}
		}
		t.Errorf("TestIDs: %s is not in the result", exp)
	}
}
