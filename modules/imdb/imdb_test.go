package imdb

import (
	"reflect"
	"testing"

	"github.com/odwrtw/polochon/lib"
)

// Test data, fakes users wishlists
var testData = map[string]map[string][]string{
	"bob": map[string][]string{
		"movies": []string{"movie1", "movie2"},
		"shows":  []string{"show1", "show2"},
	},
	"joe": map[string][]string{
		"movies": []string{"movie1", "movie3"},
		"shows":  []string{"show1", "show3"},
	},
}

// Fake wishlist with defined users
var testWishlist = &Wishlist{userIDs: []string{"bob", "joe"}}

func TestMoviesWishlist(t *testing.T) {
	getMoviesFromImdb = func(userID string) (*[]string, error) {
		ids := testData[userID]["movies"]
		return &ids, nil
	}

	got, err := testWishlist.GetMovieWishlist()
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

	got, err := testWishlist.GetShowWishlist()
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
