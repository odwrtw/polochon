package polochon

import (
	"fmt"
	"testing"

	"github.com/Sirupsen/logrus"
)

func TestStoreMovieNoPath(t *testing.T) {
	vs := NewVideoStore(FileConfig{}, MovieConfig{}, ShowConfig{}, VideoStoreConfig{})
	movie := mockMovie(MovieConfig{})

	if err := vs.Add(movie, mockLogEntry); err != ErrMissingMovieFilePath {
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

	vs := NewVideoStore(FileConfig{}, MovieConfig{}, ShowConfig{}, VideoStoreConfig{
		MovieDir: "/movie",
		ShowDir:  "/show",
	})

	writeNFOFile = func(filePath string, i interface{}, vs *VideoStore) error {
		return nil
	}

	movie := &Movie{
		Title: "Test Movie",
		Year:  1,
		File: File{
			Path: "/testmovie.avi",
		},
		Fanart: "/",
		Thumb:  "/",
	}

	expectedNewPath := "/movie/Test Movie (1)/testmovie.avi"

	if err := vs.Add(movie, mockLogEntry); err != nil {
		t.Errorf("Expected nil, got %q", err)
	}

	if movie.Path != expectedNewPath {
		t.Errorf("Expected %q, got %q", expectedNewPath, movie.Path)
	}
}

type FakeShowDetailer struct {
	show Show
}

func (d *FakeShowDetailer) Name() string {
	return "FakeShowDetailer"
}

func (d *FakeShowDetailer) GetDetails(i interface{}, log *logrus.Entry) error {
	switch v := i.(type) {
	case *Show:
		return d.showDetails(v)
	default:
		return fmt.Errorf("Error invalid type")
	}
}

func (d *FakeShowDetailer) showDetails(s *Show) error {
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
		show: Show{
			Title:  "Test show",
			Plot:   "Test show plot",
			TvdbID: 0,
			URL:    "http://fakeurl.test",
			ImdbID: "ttFakeShow",
			Banner: "/",
			Fanart: "/",
			Poster: "/",
		},
	}

	vs := NewVideoStore(FileConfig{}, MovieConfig{}, ShowConfig{}, VideoStoreConfig{
		MovieDir: "/movie",
		ShowDir:  "/show",
	})

	writeNFOFile = func(filePath string, i interface{}, vs *VideoStore) error {
		return nil
	}

	episode := &ShowEpisode{
		Title:     "Test Episode",
		ShowTitle: "Test show",
		Season:    1,
		File: File{
			Path: "/episode.avi",
		},
		ShowConfig: ShowConfig{
			Detailers: []Detailer{showDetailer},
		},
	}

	expectedNewPath := "/show/Test show/Season 1/episode.avi"

	if err := vs.Add(episode, mockLogEntry); err != nil {
		t.Errorf("Expected nil, got %q", err)
	}

	if episode.Path != expectedNewPath {
		t.Errorf("Expected %q, got %q", expectedNewPath, episode.Path)
	}
}
