package imdb

import (
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

var fakeLogEntry = logrus.NewEntry(logrus.New())

// Test data, fakes users wishlists
var testData = map[string]map[string][]string{
	"bob": {
		"movies": {"movie1", "movie2"},
		"shows":  {"show1", "show2"},
	},
	"joe": {
		"movies": {"movie1", "movie3"},
		"shows":  {"show1", "show3"},
	},
}

// Fake wishlist with defined users
var testWishlist = &Wishlist{
	UserIDs: []string{"bob", "joe", "robert"},
}

func TestMoviesWishlist(t *testing.T) {
	getMoviesFromImdb = func(userID string) (*[]string, error) {
		ids := testData[userID]["movies"]
		return &ids, nil
	}

	got, err := testWishlist.GetMovieWishlist(fakeLogEntry)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	expected := []*polochon.WishedMovie{
		{ImdbID: "movie1"},
		{ImdbID: "movie2"},
		{ImdbID: "movie3"},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %+v, got %+v", expected, got)
	}
}

func TestShowsWishlist(t *testing.T) {
	getShowsFromImdb = func(userID string) (*[]string, error) {
		ids := testData[userID]["shows"]
		return &ids, nil
	}

	got, err := testWishlist.GetShowWishlist(fakeLogEntry)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	expected := []*polochon.WishedShow{
		{ImdbID: "show1"},
		{ImdbID: "show2"},
		{ImdbID: "show3"},
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %+v, got %+v", expected, got)
	}
}

func TestEmptyWishlist(t *testing.T) {
	getMoviesFromImdb = func(userID string) (*[]string, error) {
		return nil, nil
	}

	got, err := testWishlist.GetMovieWishlist(fakeLogEntry)
	if err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	expected := []*polochon.WishedMovie{}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %+v, got %+v", expected, got)
	}
}
