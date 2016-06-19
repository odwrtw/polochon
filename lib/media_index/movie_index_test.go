package index

import (
	"reflect"
	"testing"

	"github.com/odwrtw/polochon/lib"
)

// mockMovieIndex returns a mock movie index
func mockMovieIndex() *MovieIndex {
	return &MovieIndex{
		ids: map[string]string{
			"tt56789": "/home/test/movie/movie.mp4",
			"tt12345": "/home/test/movieBis/movieBis.mp4",
		},
	}
}

func TestMovieIndexHas(t *testing.T) {
	idx := mockMovieIndex()
	for id, expected := range map[string]bool{
		"tt56789": true,
		"tt12345": true,
		"tt1234":  false,
	} {
		got, err := idx.Has(id)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}
		if expected != got {
			t.Errorf("expected %t, got %t for %s", expected, got, id)
		}
	}
}

func TestMovieIndexMoviePath(t *testing.T) {
	idx := mockMovieIndex()
	for _, mock := range []struct {
		id            string
		expectedPath  string
		expectedError error
	}{
		{
			id:            "tt12345",
			expectedPath:  "/home/test/movieBis/movieBis.mp4",
			expectedError: nil,
		},
		{
			id:            "tt56789",
			expectedPath:  "/home/test/movie/movie.mp4",
			expectedError: nil,
		},
		{
			id:            "tt1234",
			expectedPath:  "",
			expectedError: ErrNotFound,
		},
	} {
		path, err := idx.MoviePath(mock.id)
		if path != mock.expectedPath {
			t.Errorf("expected %s, got %s for %s", mock.expectedPath, path, mock.id)
		}

		if err != mock.expectedError {
			t.Errorf("expected error %s, got %s for %s", mock.expectedError, err, mock.id)
		}
	}
}

func TestMovieIndexAddAndRemove(t *testing.T) {
	idx := NewMovieIndex()
	m := &polochon.Movie{ImdbID: "tt2562232"}
	m.Path = "/home/test/movie/movie.mp4"

	if err := idx.Add(m); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	inIndex, err := idx.Has(m.ImdbID)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !inIndex {
		t.Fatalf("the movie %q should be in the index", m.ImdbID)
	}

	if err = idx.Remove(m, mockLogEntry); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	inIndex, err = idx.Has(m.ImdbID)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if inIndex {
		t.Fatalf("the movie %q should not be in the index", m.ImdbID)
	}
}

func TestMovieIndexIDs(t *testing.T) {
	idx := mockMovieIndex()
	expected := []string{"tt12345", "tt56789"}

	got, err := idx.IDs()
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %+v , got %+v", expected, got)
	}
}

func TestMovieIndexClear(t *testing.T) {
	idx := mockMovieIndex()
	expected := map[string]string{}

	idx.Clear()

	if !reflect.DeepEqual(idx.ids, expected) {
		t.Errorf("expected %+v , got %+v", expected, idx)
	}
}
