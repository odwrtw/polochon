package polochon

import (
	"bytes"
	"reflect"
	"testing"
	"time"
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
  <premiered>2015-09-24</premiered>
</tvshow>`)

func mockShow() *Show {
	premiered := time.Date(2015, time.September, 24, 0, 0, 0, 0, time.UTC)
	return &Show{
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

func TestShowStoreWriter(t *testing.T) {
	s := mockShow()

	var b bytes.Buffer
	if err := WriteNFO(&b, s); err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(seasonNFOContent, b.Bytes()) {
		t.Fatalf("Failed to serialize show season NFO")
	}
}

func TestShowReader(t *testing.T) {
	expected := mockShow()

	got := &Show{}
	if err := ReadNFO(bytes.NewBuffer(seasonNFOContent), got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize show season NFO.\nGot: %#v\nExpected: %#v", got, expected)
	}
}
