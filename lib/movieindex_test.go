package polochon

import (
	"log"
	"testing"
)

// NewMovieIndex returns a new movie index
func mockMovieIndex() *MovieIndex {
	return &MovieIndex{
		ids:   map[string]string{},
		slugs: map[string]string{},
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
	m := mockMovieIndex()

	m.ids = idsIndex

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
	m := mockMovieIndex()

	m.slugs = slugsIndex

	type res struct {
		path string
		err  error
	}

	for s, expected := range map[string]res{
		"movie": {
			"/home/test/movie/movie.mp4",
			nil,
		},
		"movieBis": {
			"/home/test/movieBis/movieBis.mp4",
			nil,
		},
		"movieDoubleBis": {
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
	m := mockMovieIndex()

	m.ids = idsIndex

	type res struct {
		path string
		err  error
	}

	for i, expected := range map[string]res{
		"tt12345": {
			"/home/test/movieBis/movieBis.mp4",
			nil,
		},
		"tt56789": {
			"/home/test/movie/movie.mp4",
			nil,
		},
		"tt1234": {
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
	mi := mockMovieIndex()

	m := mockMovie(MovieConfig{})
	m.Path = "/home/test/movie/movie.mp4"
	err := mi.Add(m)
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

	err = mi.Remove(m, mockLogEntry)
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
	m := mockMovieIndex()

	m.slugs = slugsIndex

	expectedSlugs := []string{"movie", "movieBis"}
	slugs, err := m.Slugs()
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
	m := mockMovieIndex()

	m.ids = idsIndex

	expectedIDs := []string{"tt56789", "tt12345"}
	ids, err := m.IDs()
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
