package polochon

import (
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
)

// Fake movies
var fakeWishedShows = []*WishedShow{
	{ImdbID: "show1", Season: 5, Episode: 6},
	{ImdbID: "show2", Season: 1, Episode: 4},
	{ImdbID: "show2", Season: 1, Episode: 5},
	{ImdbID: "show2", Season: 1, Episode: 2},
	{ImdbID: "show2", Season: 3, Episode: 5},
	{ImdbID: "show3", Season: 4, Episode: 5},
	{ImdbID: "show1", Season: 1, Episode: 1},
	{ImdbID: "show1", Season: 0, Episode: 0},
}

var expectedWishedShows = []*WishedShow{
	{ImdbID: "show1", Season: 1, Episode: 1},
	{ImdbID: "show2", Season: 1, Episode: 2},
	{ImdbID: "show3", Season: 4, Episode: 5},
}

var expectedWishedShowsWithQualities = []*WishedShow{
	{ImdbID: "show1", Season: 1, Episode: 1, Qualities: []Quality{Quality480p, Quality1080p}},
	{ImdbID: "show2", Season: 1, Episode: 2, Qualities: []Quality{Quality480p, Quality1080p}},
	{ImdbID: "show3", Season: 4, Episode: 5, Qualities: []Quality{Quality480p, Quality1080p}},
}

// Fake shows
var fakeWishedMovies = []*WishedMovie{
	{ImdbID: "movie1"},
	{ImdbID: "movie2"},
	{ImdbID: "movie3"},
	{ImdbID: "movie1"},
}

var expectedWishedMovies = []*WishedMovie{
	{ImdbID: "movie1"},
	{ImdbID: "movie2"},
	{ImdbID: "movie3"},
}

var expectedWishedMoviesWithQualities = []*WishedMovie{
	{ImdbID: "movie1", Qualities: []Quality{Quality1080p, Quality720p}},
	{ImdbID: "movie2", Qualities: []Quality{Quality1080p, Quality720p}},
	{ImdbID: "movie3", Qualities: []Quality{Quality1080p, Quality720p}},
}

// Fake wishlister
type FakeWishlister struct{}

func (fw *FakeWishlister) Name() string {
	return "fake"
}

func (fw *FakeWishlister) GetMovieWishlist() ([]*WishedMovie, error) {
	return fakeWishedMovies, nil
}

func (fw *FakeWishlister) GetShowWishlist() ([]*WishedShow, error) {
	return fakeWishedShows, nil
}

func NewFakeWishlister(params map[string]interface{}, log *logrus.Entry) (Wishlister, error) {
	return &FakeWishlister{}, nil
}

func TestAddMoviesWishlist(t *testing.T) {
	wl := Wishlist{}

	for _, m := range fakeWishedMovies {
		if err := wl.AddMovie(m); err != nil {
			t.Fatalf("Expected no error, got %q", err)
		}
	}
	got := wl.Movies

	if !reflect.DeepEqual(got, expectedWishedMovies) {
		t.Errorf("Expected %#v, got %#v", expectedWishedMovies, got)
	}
}

func TestAddShowsWishlist(t *testing.T) {
	wl := Wishlist{}

	for _, s := range fakeWishedShows {
		if err := wl.AddShow(s); err != nil {
			t.Fatalf("Expected no error, got %q", err)
		}
	}
	got := wl.Shows

	if !reflect.DeepEqual(got, expectedWishedShows) {
		t.Errorf("Expected %#v, got %#v", expectedWishedShows, got)
	}
}

func TestFetchWishlist(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	params := map[string]interface{}{}
	wishlister, _ := NewFakeWishlister(params, log)

	conf := WishlistConfig{
		Wishlisters:           []Wishlister{wishlister},
		ShowDefaultQualities:  []Quality{Quality480p, Quality1080p},
		MovieDefaultQualities: []Quality{Quality1080p, Quality720p},
	}

	wl := NewWishlist(conf, logrus.New())

	if err := wl.Fetch(); err != nil {
		t.Fatalf("Expected no error, got %q", err)
	}

	// Test movies
	movies := wl.Movies
	if !reflect.DeepEqual(movies, expectedWishedMoviesWithQualities) {
		t.Errorf("Expected %#v, got %#v", expectedWishedMovies, movies)
	}

	shows := wl.Shows
	if !reflect.DeepEqual(shows, expectedWishedShowsWithQualities) {
		t.Errorf("Expected %#v, got %#v", expectedWishedShows, shows)
	}
}
