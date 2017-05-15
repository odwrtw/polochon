package index

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/odwrtw/polochon/lib"
)

// mockMovieIndex returns a mock movie index
func mockMovieIndex() *MovieIndex {
	return &MovieIndex{
		ids: map[string]*Movie{
			"tt56789": &Movie{
				Path: "/home/test/movie/movie.mp4",
				Subtitles: []polochon.Language{
					polochon.FR,
					polochon.EN,
				},
			},
			"tt12345": &Movie{
				Path: "/home/test/movieBis/movieBis.mp4",
			},
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
		movie, err := idx.Movie(mock.id)
		if err != mock.expectedError {
			t.Errorf("expected error %s, got %s for %s", mock.expectedError, err, mock.id)
		}

		if movie != nil && movie.Path != mock.expectedPath {
			t.Errorf("expected %s, got %s for %s", mock.expectedPath, movie.Path, mock.id)
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

	got := idx.IDs()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %+v , got %+v", expected, got)
	}
}

func TestMovieIndex(t *testing.T) {
	idx := mockMovieIndex()
	expected := map[string]*Movie{
		"tt56789": &Movie{
			Path: "/home/test/movie/movie.mp4",
			Subtitles: []polochon.Language{
				polochon.FR,
				polochon.EN,
			},
		},
		"tt12345": &Movie{
			Path: "/home/test/movieBis/movieBis.mp4",
		},
	}

	got := idx.Index()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("expected %+v , got %+v", expected, got)
	}
}

func TestMovieIndexClear(t *testing.T) {
	idx := mockMovieIndex()
	expected := map[string]*Movie{}

	idx.Clear()

	if !reflect.DeepEqual(idx.ids, expected) {
		t.Errorf("expected %+v , got %+v", expected, idx)
	}
}

func TestMovieIndexHasSubtitles(t *testing.T) {
	idx := mockMovieIndex()
	for _, test := range []struct {
		imdbID      string
		lang        polochon.Language
		expected    bool
		expectedErr error
	}{
		{
			imdbID:      "tt56789",
			lang:        polochon.EN,
			expected:    true,
			expectedErr: nil,
		},
		{
			imdbID:      "tt12345",
			lang:        polochon.EN,
			expected:    false,
			expectedErr: nil,
		},
		{
			imdbID:      "tt123456",
			lang:        polochon.EN,
			expected:    false,
			expectedErr: ErrNotFound,
		},
	} {
		got, err := idx.HasSubtitle(test.imdbID, test.lang)
		if err != nil {
			t.Fatalf("expected no error, got %q", err)
		}
		if test.expected != got {
			t.Errorf("expected %t, got %t for %s", test.expected, got, test.imdbID)
		}
	}
}

func TestMovieIndexAddSubtitles(t *testing.T) {
	idx := mockMovieIndex()

	m := &polochon.Movie{ImdbID: "tt2562232"}
	m.Path = "/home/test/movie/movie.mp4"

	// Check to add subtitle if movie not yet added
	err := idx.AddSubtitle(m, polochon.FR)
	if err == nil {
		t.Fatal("expected error")
	}

	expectedErrString := fmt.Sprintf("failed to add subtitle : movie %s not indexed", m.ImdbID)
	if err.Error() != expectedErrString {
		t.Fatalf("expected error %q - got %q", expectedErrString, err)
	}

	if err := idx.Add(m); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	subInIndex, err := idx.HasSubtitle(m.ImdbID, polochon.FR)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if subInIndex {
		t.Fatalf("the movie subtitle %q should not be in the index", m.ImdbID)
	}

	// Add the subtitle
	err = idx.AddSubtitle(m, polochon.FR)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}

	subInIndex, err = idx.HasSubtitle(m.ImdbID, polochon.FR)
	if err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if !subInIndex {
		t.Fatalf("the movie subtitle %q should be in the index", m.ImdbID)
	}
}
