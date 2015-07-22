package polochon

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/Sirupsen/logrus"
)

func newFakeMovie() *Movie {
	m := NewMovie(MovieConfig{})
	m.ImdbID = "tt2562232"
	m.OriginalTitle = "Birdman"
	m.Plot = "Awesome plot"
	m.Rating = 7.7
	m.Runtime = 119
	m.SortTitle = "Birdman"
	m.Tagline = "or (The Unexpected Virtue of Ignorance)"
	m.Thumb = "https://image.tmdb.org/t/p/original/rSZs93P0LLxqlVEbI001UKoeCQC.jpg"
	m.Fanart = "https://image.tmdb.org/t/p/original/AsJVim0Hk3KbQPbfjyijfjqmaoZ.jpg"
	m.Title = "Birdman"
	m.TmdbID = 194662
	m.Votes = 747
	m.Year = 2014
	return m
}

var movieNFOContent = []byte(`<movie>
  <id>tt2562232</id>
  <originaltitle>Birdman</originaltitle>
  <plot>Awesome plot</plot>
  <rating>7.7</rating>
  <runtime>119</runtime>
  <sorttitle>Birdman</sorttitle>
  <tagline>or (The Unexpected Virtue of Ignorance)</tagline>
  <thumb>https://image.tmdb.org/t/p/original/rSZs93P0LLxqlVEbI001UKoeCQC.jpg</thumb>
  <customfanart>https://image.tmdb.org/t/p/original/AsJVim0Hk3KbQPbfjyijfjqmaoZ.jpg</customfanart>
  <title>Birdman</title>
  <tmdbid>194662</tmdbid>
  <votes>747</votes>
  <year>2014</year>
</movie>`)

func TestMovieNFOWriter(t *testing.T) {
	m := newFakeMovie()

	var b bytes.Buffer
	err := writeNFO(&b, m)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(movieNFOContent, b.Bytes()) {
		t.Errorf("Failed to serialize movie NFO")
	}
}

func TestMovieNFOReader(t *testing.T) {
	expected := newFakeMovie()

	got, err := readMovieNFO(bytes.NewBuffer(movieNFOContent), MovieConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize movie NFO")
	}
}

func TestMovieStoreMissingArguments(t *testing.T) {
	m := NewMovie(MovieConfig{})

	if err := m.Store(); err != ErrMissingMovieFilePath {
		t.Errorf("Expected %q, got %q", ErrMissingMovieFilePath, err)
	}

	m.File = *NewFile("fakepath")
	if err := m.Store(); err != ErrMissingMovieDir {
		t.Errorf("Expected %q, got %q", ErrMissingMovieDir, err)
	}
}

func TestMovieStorePath(t *testing.T) {
	m := newFakeMovie()
	m.MovieConfig = MovieConfig{Dir: "/movies"}

	// New movie has a year in its data, we expect to find it in the store path
	got := m.storePath()
	if !strings.Contains(got, strconv.Itoa(m.Year)) {
		t.Errorf("if the movie has a year, it sould be in the path, got %q", got)
	}

	// Unset the year
	m.Year = 0
	got = m.storePath()
	if strings.Contains(got, strconv.Itoa(m.Year)) {
		t.Errorf("if the movie has a no year, it sould not be in the path, got %q", got)
	}
}

func TestMovieStore(t *testing.T) {
	downloadMovieImage = func(URL, savePath string, log *logrus.Entry) error {
		return nil
	}

	// Create a tmp dir
	tmpDir, err := ioutil.TempDir(os.TempDir(), "polochon-movie-store")
	if err != nil {
		t.Fatalf("failed to create temp dir for movie store test")
	}
	defer os.RemoveAll(tmpDir)

	fakeLogger := logrus.New()
	c := VideoConfig{Movie: MovieConfig{Dir: tmpDir}}
	m := newFakeMovie()
	m.SetConfig(&c, fakeLogger)

	// Create a fake movie file
	file, err := ioutil.TempFile(os.TempDir(), "polochon-fake-movie")
	if err != nil {
		t.Fatalf("failed to create fake movie file in movie store test")
	}
	m.File = *NewFile(file.Name())

	// Create a tmp folder in where the movie should be stored.
	// It should be removed when storing the movie
	if err := os.Mkdir(m.storePath(), os.ModePerm); err != nil {
		t.Fatalf("failed to create a fake movie folder")
	}

	// Create a fake movie file in the destination folder, it should be removed
	// when the new one is stored
	fileToBeDeleted, err := ioutil.TempFile(m.storePath(), "file-to-be-removed")
	if err != nil {
		t.Fatalf("failed to create fake file to be removed: %q", err)
	}

	if err := m.Store(); err != nil {
		t.Errorf("Expected no error got %q", err)
	}

	// Check new movie file path
	relativePath, err := filepath.Rel(tmpDir, m.File.Path)
	if err != nil {
		t.Fatal(err)
	}

	if len(strings.Split(relativePath, "/")) != 2 {
		t.Error("relative path should contain MovieFolder/MovieFile")
	}

	// Ensure the nfo file is written
	if _, err := os.Stat(m.File.NfoPath()); err != nil {
		t.Errorf("nfo file was not created: %q", err)
	}

	// Ensure file in the destination folder has been removed
	if _, err := os.Stat(fileToBeDeleted.Name()); err == nil {
		t.Errorf("destination folder was not removed before storing movie: %q", err)
	}
}

func TestDownloadImagesInvalidArguments(t *testing.T) {
	m := newFakeMovie()
	m.log = logrus.NewEntry(logrus.New())

	m.Fanart = ""
	if err := m.downloadImages(); err != ErrMissingMovieImageURL {
		t.Errorf("expected error %q, got %q", ErrMissingMovieImageURL, err)
	}

	m.Fanart = "url"
	m.Thumb = ""
	if err := m.downloadImages(); err != ErrMissingMovieImageURL {
		t.Errorf("expected error %q, got %q", ErrMissingMovieImageURL, err)
	}
}

func TestMovieSlug(t *testing.T) {
	s := newFakeMovie()
	got := s.Slug()
	expected := "birdman"

	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}
