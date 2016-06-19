package library

import (
	"fmt"
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

func TestStoreMovie(t *testing.T) {
	exists = func(path string) bool {
		return false
	}

	mkdir = func(path string) error {
		return nil
	}

	move = func(from string, to string) error {
		return nil
	}

	remove = func(path string) error {
		return nil
	}
	download = func(URL, savePath string) error {
		return nil
	}

	library := New(polochon.FileConfig{}, polochon.MovieConfig{}, polochon.ShowConfig{}, Config{
		MovieDir: "/movie",
		ShowDir:  "/show",
	})

	writeNFOFile = func(filePath string, i interface{}, library *Library) error {
		return nil
	}

	movie := &polochon.Movie{
		Title: "Test Movie",
		Year:  1,
		File: polochon.File{
			Path: "/testmovie.avi",
		},
		Fanart: "/",
		Thumb:  "/",
	}

	expectedNewPath := "/movie/Test Movie (1)/testmovie.avi"

	if err := library.Add(movie, mockLogEntry); err != nil {
		t.Errorf("Expected nil, got %q", err)
	}

	if movie.Path != expectedNewPath {
		t.Errorf("Expected %q, got %q", expectedNewPath, movie.Path)
	}
}

type FakeShowDetailer struct {
	show polochon.Show
}

func (d *FakeShowDetailer) Name() string {
	return "FakeShowDetailer"
}

func (d *FakeShowDetailer) GetDetails(i interface{}, log *logrus.Entry) error {
	switch v := i.(type) {
	case *polochon.Show:
		return d.showDetails(v)
	default:
		return fmt.Errorf("Error invalid type")
	}
}

func (d *FakeShowDetailer) showDetails(s *polochon.Show) error {
	s.Title = d.show.Title
	s.Plot = d.show.Plot
	s.TvdbID = d.show.TvdbID
	s.URL = d.show.URL
	s.ImdbID = d.show.ImdbID
	s.Banner = d.show.Banner
	s.Fanart = d.show.Fanart
	s.Poster = d.show.Poster
	return nil
}

func TestStoreShow(t *testing.T) {
	exists = func(path string) bool {
		return false
	}

	mkdir = func(path string) error {
		return nil
	}

	move = func(from string, to string) error {
		return nil
	}

	remove = func(path string) error {
		return nil
	}
	download = func(URL, savePath string) error {
		return nil
	}

	showDetailer := &FakeShowDetailer{
		show: polochon.Show{
			Title:  "Test show",
			Plot:   "Test show plot",
			TvdbID: 0,
			URL:    "http://fakeurlibrary.test",
			ImdbID: "ttFakeShow",
			Banner: "/",
			Fanart: "/",
			Poster: "/",
		},
	}

	library := New(polochon.FileConfig{}, polochon.MovieConfig{}, polochon.ShowConfig{}, Config{
		MovieDir: "/movie",
		ShowDir:  "/show",
	})

	writeNFOFile = func(filePath string, i interface{}, library *Library) error {
		return nil
	}

	episode := &polochon.ShowEpisode{
		Title:     "Test Episode",
		ShowTitle: "Test show",
		Season:    1,
		File: polochon.File{
			Path: "/episode.avi",
		},
		ShowConfig: polochon.ShowConfig{
			Detailers: []polochon.Detailer{showDetailer},
		},
	}

	expectedNewPath := "/show/Test show/Season 1/episode.avi"

	if err := library.Add(episode, mockLogEntry); err != nil {
		t.Errorf("Expected nil, got %q", err)
	}

	if episode.Path != expectedNewPath {
		t.Errorf("Expected %q, got %q", expectedNewPath, episode.Path)
	}
}
