package polochon

import (
	"bytes"
	"encoding/xml"
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
)

func fakeShowEpisode() *ShowEpisode {
	s := NewShowEpisode(ShowConfig{Dir: "/shows"})
	s.XMLName = xml.Name{Space: "", Local: "episodedetails"}
	s.Title = "Lost in Space"
	s.ShowTitle = "American Dad!"
	s.Season = 9
	s.Episode = 18
	s.TvdbID = 4488786
	s.Aired = "2013-05-05"
	s.Plot = "Awesome plot"
	s.Runtime = 30
	s.Thumb = "http://thetvdb.com/banners/episodes/73141/4488786.jpg"
	s.Rating = 7.6
	s.ShowImdbID = "tt0397306"
	s.ShowTvdbID = 73141

	s.log = logrus.NewEntry(logrus.New())

	return s
}

var episodeNFOContent = []byte(`<episodedetails>
  <title>Lost in Space</title>
  <showtitle>American Dad!</showtitle>
  <season>9</season>
  <episode>18</episode>
  <uniqueid>4488786</uniqueid>
  <aired>2013-05-05</aired>
  <plot>Awesome plot</plot>
  <runtime>30</runtime>
  <thumb>http://thetvdb.com/banners/episodes/73141/4488786.jpg</thumb>
  <rating>7.6</rating>
  <showimdbid>tt0397306</showimdbid>
  <showtvdbid>73141</showtvdbid>
  <episodeimdbid></episodeimdbid>
</episodedetails>`)

func TestShowEpisodeStoreWriter(t *testing.T) {
	s := fakeShowEpisode()
	s.log = nil

	var b bytes.Buffer
	err := writeNFO(&b, s)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(episodeNFOContent, b.Bytes()) {
		t.Errorf("Failed to serialize show episode NFO")
	}
}

func TestShowEpisodeReader(t *testing.T) {
	expected := fakeShowEpisode()
	expected.log = nil
	got, err := readShowEpisodeNFO(bytes.NewBuffer(episodeNFOContent), ShowConfig{Dir: "/shows"})
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize show episode NFO.\nGot: %#v\nExpected: %#v", got, expected)
	}
}

func TestShowEpisodeStorePath(t *testing.T) {
	s := fakeShowEpisode()
	got := s.storePath()
	expected := "/shows/American Dad!/Season 9"

	if got != expected {
		t.Errorf("got %q, expected %q", got, expected)
	}
}
