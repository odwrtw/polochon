package polochon

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
)

var fakeLogger = logrus.NewEntry(logrus.New())

func newFakeMovie(conf MovieConfig) *Movie {
	m := NewMovie(conf)
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
	m := newFakeMovie(MovieConfig{})

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
	expected := newFakeMovie(MovieConfig{})

	got, err := readMovieNFO(bytes.NewBuffer(movieNFOContent), MovieConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize movie NFO")
	}
}

func TestMovieSlug(t *testing.T) {
	s := newFakeMovie(MovieConfig{})
	got := s.Slug()
	expected := "birdman"

	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}
