package library

import (
	"io/ioutil"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

var mockLogEntry = logrus.NewEntry(&logrus.Logger{Out: ioutil.Discard})

type mockInvalidType string

func (m *mockInvalidType) GetDetails(log *logrus.Entry) error  { return nil }
func (m *mockInvalidType) GetTorrents(log *logrus.Entry) error { return nil }
func (m *mockInvalidType) SetFile(f *polochon.File)            {}
func (m *mockInvalidType) GetFile() *polochon.File             { return &polochon.File{} }
func (m *mockInvalidType) GetSubtitlers() []polochon.Subtitler { return nil }
func (m *mockInvalidType) SubtitlePath(lang polochon.Language) string {
	return "path_" + string(lang)
}

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

func TestAddVideo(t *testing.T) {
	var invalidType *mockInvalidType

	lib := &Library{}
	expected := ErrInvalidIndexVideoType
	got := lib.Add(invalidType, mockLogEntry)
	if got != expected {
		t.Errorf("expected error %q, got %q", expected, got)
	}
}

func TestHasVideo(t *testing.T) {
	var invalidType *mockInvalidType

	lib := &Library{}
	expected := ErrInvalidIndexVideoType
	_, got := lib.HasVideo(invalidType)
	if got != expected {
		t.Errorf("expected error %q, got %q", expected, got)
	}
}
