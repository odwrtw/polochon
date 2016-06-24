package nfo

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/odwrtw/polochon/lib"
)

// Content of a show nfo file
var showNFOContent = []byte(`<tvshow>
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
  <premiered>2015-09-24</premiered>
</tvshow>`)

func mockShow() *polochon.Show {
	premiered := time.Date(2015, time.September, 24, 0, 0, 0, 0, time.UTC)
	return &polochon.Show{
		Title:      "American Dad!",
		Rating:     8.5,
		Plot:       "Awesome plot",
		URL:        "http://www.thetvdb.com/api/1D62F2F90030C444/series/73141/all/en.zip",
		TvdbID:     73141,
		ImdbID:     "tt0397306",
		Year:       2005,
		FirstAired: &premiered,
	}
}

func TestShowWriteNFO(t *testing.T) {
	s := mockShow()

	var b bytes.Buffer
	if err := Write(&b, s); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(showNFOContent, b.Bytes()) {
		t.Fatalf("Failed to serialize show season NFO")
	}
}

func TestShowReadNFO(t *testing.T) {
	expected := mockShow()

	got := &polochon.Show{}
	if err := Read(bytes.NewBuffer(showNFOContent), got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize show season NFO.\nGot: %#v\nExpected: %#v", got, expected)
	}
}
