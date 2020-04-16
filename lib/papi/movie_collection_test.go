package papi

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestMovieCollectionAdd(t *testing.T) {
	for _, data := range []struct {
		movie           *Movie
		shouldHaveMovie bool
		expectedError   error
	}{
		{
			movie:           &Movie{Movie: &polochon.Movie{ImdbID: "tt001"}},
			shouldHaveMovie: true,
			expectedError:   nil,
		},
		{
			movie:           &Movie{Movie: &polochon.Movie{}},
			shouldHaveMovie: false,
			expectedError:   ErrMissingMovieID,
		},
		{
			movie:           nil,
			shouldHaveMovie: false,
			expectedError:   ErrMissingMovie,
		},
	} {
		collection := NewMovieCollection()

		err := collection.Add(data.movie)

		// Check the error
		if err != data.expectedError {
			t.Fatalf("expected: %s, got %s", data.expectedError, err)
		}

		var imdbID string
		if data.movie != nil && data.movie.ImdbID != "" {
			imdbID = data.movie.ImdbID
		}

		// Check if the movie is in the collection
		gotMovie, ok := collection.Has(imdbID)

		if ok != data.shouldHaveMovie {
			t.Fatalf("expected: %t, got %t", data.shouldHaveMovie, ok)
		}

		if !ok {
			continue
		}

		if gotMovie != data.movie {
			t.Fatalf("expected: %+v, got %+v", data.movie, gotMovie)
		}
	}
}

func TestMovieCollectionList(t *testing.T) {
	m1 := &Movie{Movie: &polochon.Movie{ImdbID: "tt001"}}
	m2 := &Movie{Movie: &polochon.Movie{ImdbID: "tt002"}}
	mList := []*Movie{m1, m2}

	collection := NewMovieCollection()

	for _, m := range mList {
		if err := collection.Add(m); err != nil {
			t.Fatalf("expected no error, got %s", err)
		}
	}

	cList := collection.List()
	if !reflect.DeepEqual(cList, mList) {
		t.Fatalf("expected: %+v, got %+v", mList, cList)
	}

	ret, ok := collection.Has("tt003")
	if ret != nil {
		t.Fatalf("expected nil, got %+v", ret)
	}

	if ok {
		t.Fatalf("the movie shouldn't be in the colleciton")
	}
}
