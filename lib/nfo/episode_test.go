package nfo

import (
	"bytes"
	"reflect"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
)

func mockEpisode() *polochon.ShowEpisode {
	s := polochon.NewShowEpisode(polochon.ShowConfig{})
	s.VideoMetadata = polochon.VideoMetadata{
		DateAdded:    now(),
		Quality:      polochon.Quality720p,
		ReleaseGroup: "GoT[TGx]",
		AudioCodec:   "Dolby Digital Plus",
		VideoCodec:   "H.264",
		Container:    "mp4",
	}
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

	return s
}

var episodeNFOContent = []byte(`<episodedetails>
  <polochon>
    <date_added>2019-05-07T12:00:00Z</date_added>
    <quality>720p</quality>
    <release_group>GoT[TGx]</release_group>
    <audio_codec>Dolby Digital Plus</audio_codec>
    <video_codec>H.264</video_codec>
    <container>mp4</container>
  </polochon>
  <title>Lost in Space</title>
  <showtitle>American Dad!</showtitle>
  <season>9</season>
  <episode>18</episode>
  <uniqueid>4488786</uniqueid>
  <aired>2013-05-05</aired>
  <premiered>2013-05-05</premiered>
  <plot>Awesome plot</plot>
  <runtime>30</runtime>
  <thumb>http://thetvdb.com/banners/episodes/73141/4488786.jpg</thumb>
  <rating>7.6</rating>
  <showimdbid>tt0397306</showimdbid>
  <showtvdbid>73141</showtvdbid>
  <episodeimdbid></episodeimdbid>
</episodedetails>`)

func TestEpisodeWriteNFO(t *testing.T) {
	s := mockEpisode()

	var b bytes.Buffer
	err := Write(&b, s)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(episodeNFOContent, b.Bytes()) {
		t.Errorf("Failed to serialize show episode NFO")
	}
}

func TestEpisodeReadNFO(t *testing.T) {
	expected := mockEpisode()
	got := &polochon.ShowEpisode{}
	if err := Read(bytes.NewBuffer(episodeNFOContent), got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to deserialize show episode NFO.\nGot: %#v\nExpected: %#v", got, expected)
	}
}
