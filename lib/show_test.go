package polochon

import (
	"bytes"
	"io/ioutil"
	"os"
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
	s := NewShow()
	s.Title = "American Dad!"
	s.ShowTitle = "American Dad!"
	s.Rating = 8.5
	s.Plot = "Awesome plot"
	s.URL = "http://www.thetvdb.com/api/1D62F2F90030C444/series/73141/all/en.zip"
	s.TvdbID = 73141
	s.ImdbID = "tt0397306"
	s.Year = 2005

	s.config = &ShowConfig{Dir: "/shows"}
	s.log = logrus.NewEntry(logrus.New())

	return s
}

func TestShowStoreWriter(t *testing.T) {
	s := newFakeShow()
	s.log = nil
	s.config = nil

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
	expected.config = nil

	got, err := readShowSeasonNFO(bytes.NewBuffer(seasonNFOContent))
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize show season NFO.\nGot: %#v\nExpected: %#v", got, expected)
	}
}

func TestShowStorePath(t *testing.T) {
	s := newFakeShow()
	got := s.storePath()
	expected := "/shows/American Dad!"
	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}

func TestShowNfoPath(t *testing.T) {
	s := newFakeShow()
	got := s.nfoPath()
	expected := "/shows/American Dad!/tvshow.nfo"
	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}

func TestShowStore(t *testing.T) {
	show := newFakeShow()
	show.Banner = "fake"
	show.Poster = "fake"
	show.Fanart = "fake"

	// Create a tmp dir
	tmpDir, err := ioutil.TempDir(os.TempDir(), "polochon-show-store")
	if err != nil {
		t.Fatalf("failed to create temp dir for show store test")
	}
	defer os.RemoveAll(tmpDir)
	show.config.Dir = tmpDir

	downloadShowImage = func(URL, savePath string, log *logrus.Entry) error {
		return nil
	}

	if err := show.Store(); err != nil {
		t.Fatalf("failed to store the show: %q", err)
	}

	if f := NewFile(show.nfoPath()); !f.Exists() {
		t.Fatalf("the show nfo was not created")
	}
}
