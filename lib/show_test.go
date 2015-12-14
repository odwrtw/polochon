package polochon

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
)

// Content of a season nfo file
var seasonNFOContent = []byte(`<tvshow>
  <title>American Dad!</title>
  <showtitle>American Dad!</showtitle>
  <rating>8.5</rating>
  <plot>Awesome plot</plot>
  <episodeguide>
    <url>http://www.thetvdb.com/api/1D62F2F90030C444/series/73141/all/en.zip</url>
  </episodeguide>
  <tvdbid>73141</tvdbid>
  <imdbid>tt0397306</imdbid>
  <year>2005</year>
</tvshow>`)

func newFakeShow() *Show {
	s := NewShow(ShowConfig{})
	s.Title = "American Dad!"
	s.ShowTitle = "American Dad!"
	s.Rating = 8.5
	s.Plot = "Awesome plot"
	s.URL = "http://www.thetvdb.com/api/1D62F2F90030C444/series/73141/all/en.zip"
	s.TvdbID = 73141
	s.ImdbID = "tt0397306"
	s.Year = 2005

	s.log = logrus.NewEntry(logrus.New())

	return s
}

func TestShowStoreWriter(t *testing.T) {
	s := newFakeShow()
	s.log = nil

	var b bytes.Buffer
	err := writeNFO(&b, s)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(seasonNFOContent, b.Bytes()) {
		t.Errorf("Failed to serialize show season NFO")
	}
}

func TestShowReader(t *testing.T) {
	expected := newFakeShow()
	expected.log = nil

	got, err := readShowNFO(bytes.NewBuffer(seasonNFOContent), ShowConfig{})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize show season NFO.\nGot: %#v\nExpected: %#v", got, expected)
	}
}
