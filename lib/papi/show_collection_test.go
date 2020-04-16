package papi

import (
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func TestShowCollectionAdd(t *testing.T) {
	for _, data := range []struct {
		show           *Show
		shouldHaveShow bool
		expectedError  error
	}{
		{
			show:           &Show{Show: &polochon.Show{ImdbID: "tt001"}},
			shouldHaveShow: true,
			expectedError:  nil,
		},
		{
			show:           &Show{Show: &polochon.Show{}},
			shouldHaveShow: false,
			expectedError:  ErrMissingShowID,
		},
		{
			show:           &Show{},
			shouldHaveShow: false,
			expectedError:  ErrMissingShow,
		},
		{
			show:           nil,
			shouldHaveShow: false,
			expectedError:  ErrMissingShow,
		},
	} {
		collection := NewShowCollection()

		err := collection.Add(data.show)

		// Check the error
		if err != data.expectedError {
			t.Fatalf("expected: %s, got %s", data.expectedError, err)
		}

		var imdbID string
		if data.show != nil && data.show.Show != nil && data.show.ImdbID != "" {
			imdbID = data.show.ImdbID
		}

		// Check if the show is in the collection
		gotShow, ok := collection.Has(imdbID)

		if ok != data.shouldHaveShow {
			t.Fatalf("expected: %t, got %t", data.shouldHaveShow, ok)
		}

		if !ok {
			continue
		}

		if gotShow != data.show {
			t.Fatalf("expected: %+v, got %+v", data.show, gotShow)
		}
	}
}

func TestShowCollectionList(t *testing.T) {
	m1 := &Show{Show: &polochon.Show{ImdbID: "tt001"}}
	m2 := &Show{Show: &polochon.Show{ImdbID: "tt002"}}
	mList := []*Show{m1, m2}

	collection := NewShowCollection()

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
		t.Fatalf("the show shouldn't be in the colleciton")
	}
}
