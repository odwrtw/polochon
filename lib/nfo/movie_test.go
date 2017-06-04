package nfo

import (
	"bytes"
	"io/ioutil"
	"reflect"
	"testing"

	"github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

var mockLogEntry = logrus.NewEntry(&logrus.Logger{Out: ioutil.Discard})

func mockMovie() *polochon.Movie {
	m := polochon.NewMovie(polochon.MovieConfig{})
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
	m.Genres = []string{"horror", "action"}
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
  <genre>horror</genre>
  <genre>action</genre>
</movie>`)

func TestMovieWriteNFO(t *testing.T) {
	m := mockMovie()

	var b bytes.Buffer
	err := Write(&b, m)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(movieNFOContent, b.Bytes()) {
		t.Errorf("Failed to serialize movie NFO")
	}
}

func TestMovieReadNFO(t *testing.T) {
	expected := mockMovie()

	got := &polochon.Movie{}
	if err := Read(bytes.NewBuffer(movieNFOContent), got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize movie NFO")
	}
}
