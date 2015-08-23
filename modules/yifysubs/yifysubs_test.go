package yifysubs

import (
	"io/ioutil"
	"log"
	"reflect"
	"testing"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yifysubs"
)

var fakeLogger = logrus.New()
var fakeLogEntry = logrus.NewEntry(fakeLogger)

var fakeSubs = map[string][]yifysubs.Subtitle{
	"french": {
		{
			ID:     58673,
			Rating: -1,
			URL:    "http://www.yifysubtitles.com/subtitle-api/aloha-yify-58673.zip",
		},
		{
			ID:     58809,
			Rating: 2,
			URL:    "http://www.yifysubtitles.com/subtitle-api/aloha-yify-58809.zip",
		},
	},
	"greek": {
		{
			ID:     59747,
			Rating: -3,
			URL:    "http://www.yifysubtitles.com/subtitle-api/aloha-yify-59747.zip",
		},
	},
}

func init() {
	fakeLogger.Out = ioutil.Discard
}

func TestNew(t *testing.T) {
	got, err := New(map[string]interface{}{"lang": "fr_FR"}, fakeLogEntry)
	if err != nil {
		log.Fatalf("Got error in New: %q", err)
	}

	expected := &YifySubs{
		lang: "french",
		log:  fakeLogEntry,
	}

	if !reflect.DeepEqual(got, expected) {
		t.Errorf("failed to create new YifySubs\nGot: %+v\nExpected: %+v", got, expected)
	}
}

func TestNewError(t *testing.T) {
	for expectedError, params := range map[error]map[string]interface{}{
		ErrMissingSubtitleLang: map[string]interface{}{},
		ErrInvalidArgument:     map[string]interface{}{"lang": 1},
		ErrInvalidSubtitleLang: map[string]interface{}{"lang": "yo"},
	} {
		_, err := New(params, fakeLogEntry)
		if err == nil {
			log.Fatal("expected an error, got none")
		}

		if err != expectedError {
			log.Fatalf("expected an %q, got %q", expectedError, err)
		}

	}
}

func TestName(t *testing.T) {
	y := &YifySubs{}
	if y.Name() != moduleName {
		log.Fatal("invalid name")
	}
}

func TestGetShowSubtitle(t *testing.T) {
	y := &YifySubs{}
	r, err := y.GetShowSubtitle(&polochon.ShowEpisode{})
	if r != nil {
		log.Fatalf("expected no result, got %+v", r)
	}

	if err != nil {
		log.Fatalf("expected no error, got %q", err)
	}
}

func TestGetMovieSubtitle(t *testing.T) {
	getSubtitles = func(imdbID string) (map[string][]yifysubs.Subtitle, error) {
		return fakeSubs, nil
	}
	m := &polochon.Movie{ImdbID: "tt9347238"}
	y := &YifySubs{lang: "french"}

	sub, err := y.GetMovieSubtitle(m)
	if err != nil {
		log.Fatalf("expected no error, got %q", err)
	}

	s, ok := sub.(*yifysubs.Subtitle)
	if !ok {
		log.Fatal("the sub is not a yifysubs.Subtitle")
	}

	expectedID := 58809
	if s.ID != expectedID {
		log.Fatalf("expected the sub with the ID %d, insted got %d", expectedID, s.ID)
	}
}

func TestGetMovieSubtitleNotFound(t *testing.T) {
	getSubtitles = func(imdbID string) (map[string][]yifysubs.Subtitle, error) {
		return fakeSubs, nil
	}
	m := &polochon.Movie{ImdbID: "tt9347238"}
	y := &YifySubs{lang: "test"}

	_, err := y.GetMovieSubtitle(m)
	if err == nil {
		log.Fatal("expected an error, got none")
	}

	if err != polochon.ErrNoSubtitleFound {
		log.Fatalf("expected %q, got %q", polochon.ErrNoSubtitleFound, err)
	}
}

func TestGetMovieSubtitleNoID(t *testing.T) {
	getSubtitles = func(imdbID string) (map[string][]yifysubs.Subtitle, error) {
		return fakeSubs, nil
	}
	m := &polochon.Movie{}
	y := &YifySubs{lang: "french"}

	_, err := y.GetMovieSubtitle(m)
	if err == nil {
		log.Fatal("expected an error, got none")
	}

	if err != ErrMissingImdbID {
		log.Fatalf("expected %q, got %q", ErrMissingImdbID, err)
	}
}
