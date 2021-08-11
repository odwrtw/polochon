package yifysubtitles

import (
	"io/ioutil"
	"log"
	"testing"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yifysubs"
	"github.com/sirupsen/logrus"
)

var fakeLogger = logrus.New()
var fakeLogEntry = logrus.NewEntry(fakeLogger)

var fakeSubs = []*yifysubs.Subtitle{
	{
		Rating: -1,
		URL:    "http://www.yifysubtitles.com/subtitles/aloha-yify-58673",
		Lang:   "French",
	},
	{
		Rating: 2,
		URL:    "http://www.yifysubtitles.com/subtitles/aloha-yify-58809",
		Lang:   "French",
	},
	{
		Rating: -3,
		URL:    "http://www.yifysubtitles.com/subtitles/aloha-yify-59747",
		Lang:   "Greek",
	},
}

func init() {
	fakeLogger.Out = ioutil.Discard
}

func TestNew(t *testing.T) {
	y := &YifySubs{}
	err := y.Init([]byte("lang: fr_FR"))
	if err != nil {
		log.Fatalf("Got error in New: %q", err)
	}
}

func TestName(t *testing.T) {
	y := &YifySubs{}
	if y.Name() != moduleName {
		log.Fatal("invalid name")
	}
}

// fakeSearcher implements the Searcher interface
type fakeSearcher struct{}

// Search returns the fake subtitles
func (f fakeSearcher) SearchByLang(imdbID, lang string) ([]*yifysubs.Subtitle, error) {
	return yifysubs.FilterByLang(fakeSubs, lang), nil
}

var fakeClient fakeSearcher

func TestGetMovieSubtitle(t *testing.T) {
	m := &polochon.Movie{ImdbID: "tt9347238"}
	y := &YifySubs{
		Client: fakeClient,
	}

	s, err := y.getMovieSubtitle(m, polochon.FR, fakeLogEntry)
	if err != nil {
		log.Fatalf("expected no error, got %q", err)
	}

	expectedURL := "http://www.yifysubtitles.com/subtitles/aloha-yify-58809"
	if s.URL != expectedURL {
		log.Fatalf("expected the sub with the URL %s, insted got %s", expectedURL, s.URL)
	}
}

func TestGetMovieSubtitleNotFound(t *testing.T) {
	m := &polochon.Movie{ImdbID: "tt9347238"}
	y := &YifySubs{
		Client: fakeClient,
	}

	_, err := y.GetMovieSubtitle(m, polochon.EN, fakeLogEntry)
	if err == nil {
		log.Fatal("expected an error, got none")
	}

	if err != polochon.ErrNoSubtitleFound {
		log.Fatalf("expected %q, got %q", polochon.ErrNoSubtitleFound, err)
	}
}

func TestGetMovieSubtitleNoID(t *testing.T) {
	m := &polochon.Movie{}
	y := &YifySubs{}

	_, err := y.GetMovieSubtitle(m, polochon.FR, fakeLogEntry)
	if err == nil {
		log.Fatal("expected an error, got none")
	}

	if err != ErrMissingImdbID {
		log.Fatalf("expected %q, got %q", ErrMissingImdbID, err)
	}
}
