package library

import (
	"io/ioutil"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/configuration"
	"github.com/sirupsen/logrus"
)

var mockLogEntry = logrus.NewEntry(&logrus.Logger{Out: ioutil.Discard})

func TestStoreMovieNoPath(t *testing.T) {
	library := New(&configuration.Config{})
	movie := &polochon.Movie{}

	if err := library.Add(movie, mockLogEntry); err != ErrMissingMovieFilePath {
		t.Errorf("Expected %q, got %q", ErrMissingMovieFilePath, err)
	}
}

func TestGetMovieDir(t *testing.T) {
	l := &Library{
		LibraryConfig: configuration.LibraryConfig{
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
