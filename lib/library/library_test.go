package library

import (
	"io/ioutil"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

var mockLogEntry = logrus.NewEntry(&logrus.Logger{Out: ioutil.Discard})

func TestStoreMovieNoPath(t *testing.T) {
	library := New(polochon.FileConfig{}, polochon.MovieConfig{}, polochon.ShowConfig{}, Config{})
	movie := &polochon.Movie{}

	if err := library.Add(movie, mockLogEntry); err != ErrMissingMovieFilePath {
		t.Errorf("Expected %q, got %q", ErrMissingMovieFilePath, err)
	}
}

func TestGetMovieDir(t *testing.T) {
	l := &Library{
		Config: Config{
			MovieDir: "/",
		},
	}
	m := &polochon.Movie{Title: "Test"}

	// Without year
	expected := "/Test"
	got := l.getMovieDir(m)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}

	// With year
	m.Year = 2000
	expected = "/Test (2000)"
	got = l.getMovieDir(m)
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
