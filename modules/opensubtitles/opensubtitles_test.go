package opensubtitles

import (
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/odwrtw/polochon/lib"
	"github.com/oz/osdb"
	"github.com/sirupsen/logrus"
)

var fakeLogger = &logrus.Logger{Out: ioutil.Discard}
var fakeLoggerEntry = logrus.NewEntry(fakeLogger)
var errFake = fmt.Errorf("fake error")
var fakeClient = &osdb.Client{Token: "pwet"}

// fake functions for tests
var fakeNewOsdbClient = func() (*osdb.Client, error) {
	return fakeClient, nil
}
var fakeNewOsdbClientError = func() (*osdb.Client, error) {
	return nil, errFake
}
var fakeCheckOsdbClient = func(c *osdb.Client) error {
	return nil
}
var fakeCheckOsdbClientError = func(c *osdb.Client) error {
	return errFake
}
var fakeLogInOsdbClient = func(c *osdb.Client, user, password, language string) error {
	return nil
}
var fakeLogInOsdbClientError = func(c *osdb.Client, user, password, language string) error {
	return errFake
}
var fakeSearchOsdbSubtitles = func(c *osdb.Client, params []interface{}) (osdb.Subtitles, error) {
	return fakeSubtitles, nil
}
var fakeSearchOsdbSubtitlesError = func(c *osdb.Client, params []interface{}) (osdb.Subtitles, error) {
	return nil, errFake
}
var fakeSearchOsdbSubtitlesPartialError = func(c *osdb.Client, params []interface{}) (osdb.Subtitles, error) {
	// Need to check within the params to see if a imdbid or season is given,
	// if yes, return results, else return an error
	for _, p := range params {
		switch reflect.TypeOf(p).Kind() {
		case reflect.Slice:
			s := reflect.ValueOf(p)
			for i := 0; i < s.Len(); i++ {
				t := s.Index(i).Interface()
				v, ok := t.(map[string]string)
				if !ok {
					return nil, ErrInvalidArgument
				}
				for key := range v {
					if key == "imdbid" || key == "season" {
						return fakeSubtitles, nil
					}
				}
			}
		}
	}
	return nil, ErrInvalidArgument
}
var fakeFileSearchSubtitles = func(c *osdb.Client, filePath string, languages []string) (osdb.Subtitles, error) {
	return fakeSubtitles, nil
}
var fakeFileSearchSubtitlesError = func(c *osdb.Client, filePath string, languages []string) (osdb.Subtitles, error) {
	return nil, errFake
}
var fakeProxy = osProxy{
	client:   nil,
	language: "fre",
	user:     "fakeUser",
	password: "fakePass",
}
var fakeMovieSub = osdb.Subtitle{
	IDMovieImdb:      "12345",
	SeriesIMDBParent: "1234",
	SeriesEpisode:    "1",
	SeriesSeason:     "1",
}
var fakeSubtitles = osdb.Subtitles{
	fakeMovieSub,
	{
		IDMovieImdb:      "54321",
		SeriesIMDBParent: "4321",
		SeriesEpisode:    "2",
		SeriesSeason:     "2",
	},
}
var fakeShowEpisode = polochon.ShowEpisode{
	ShowImdbID: "tt0001234",
	Season:     1,
	Episode:    1,
}
var fakeMovie = polochon.Movie{
	ImdbID: "tt0012345",
}

func TestInvalidNew(t *testing.T) {
	for expected, paramsStr := range map[error][]string{
		// Not a string
		ErrInvalidArgument: {
			"lang: fr_FR",
			"user: 6",
			"password: passTest",
		},
		// Missing password
		ErrMissingArgument: {
			"lang: fr_FR",
			"user: userTest",
			"password: ",
		},
		// Bad language
		ErrInvalidArgument: {
			"lang: bad_language",
			"user: userTest",
			"password: passTest",
		},
	} {
		params := []byte(strings.Join(paramsStr, "\n"))
		_, err := NewFromRawYaml(params)
		if err != expected {
			log.Fatalf("Got %q, expected %q with params %s", err, expected, params)
		}
	}
}

func TestSuccessfulNew(t *testing.T) {
	params := []byte(strings.Join([]string{
		"lang: fr_FR",
		"user: userTest",
		"password: passTest",
	}, "\n"))
	got, err := NewFromRawYaml(params)
	if err != nil {
		log.Fatalf("Got error in New: %q", err)
	}
	expected := &osProxy{
		language: "fre",
		user:     "userTest",
		password: "passTest",
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to get movie details\nGot: %+v\nExpected: %+v", got, expected)
	}
}

func TestSuccessfulDefaultLang(t *testing.T) {
	params := []byte(strings.Join([]string{
		"user: userTest",
		"password: passTest",
	}, "\n"))
	got, err := NewFromRawYaml(params)
	if err != nil {
		log.Fatalf("Got error in New: %q", err)
	}
	expected := &osProxy{
		language: "eng",
		user:     "userTest",
		password: "passTest",
	}
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Failed to get movie details\nGot: %+v\nExpected: %+v", got, expected)
	}
}

func TestGetClientFailed(t *testing.T) {
	situations := []struct {
		newOsdbClient   func() (*osdb.Client, error)
		checkOsdbClient func(c *osdb.Client) error
		logInOsdbClient func(c *osdb.Client, user, password, language string) error
		expectedErr     error
	}{
		{
			fakeNewOsdbClient,
			fakeCheckOsdbClient,
			fakeLogInOsdbClient,
			nil,
		},
		{
			fakeNewOsdbClient,
			fakeCheckOsdbClientError,
			fakeLogInOsdbClient,
			nil,
		},
		{
			fakeNewOsdbClient,
			fakeCheckOsdbClientError,
			fakeLogInOsdbClientError,
			errFake,
		},
	}
	for _, situation := range situations {
		newOsdbClient = situation.newOsdbClient
		checkOsdbClient = situation.checkOsdbClient
		logInOsdbClient = situation.logInOsdbClient

		err := fakeProxy.getOpenSubtitleClient()
		if err != situation.expectedErr {
			log.Fatalf("Got bad error in getOpenSubtitleClient: %q", err)
		}
	}
}

func TestSearchSubtitles(t *testing.T) {
	situations := []struct {
		newOsdbClient       func() (*osdb.Client, error)
		checkOsdbClient     func(c *osdb.Client) error
		logInOsdbClient     func(c *osdb.Client, user, password, language string) error
		searchOsdbSubtitles func(c *osdb.Client, params []interface{}) (osdb.Subtitles, error)
		fileSearchSubtitles func(c *osdb.Client, filePath string, languages []string) (osdb.Subtitles, error)
		expectedErr         error
		expectedSubtitle    *osdb.Subtitle
	}{
		{
			fakeNewOsdbClient,
			fakeCheckOsdbClient,
			fakeLogInOsdbClient,
			fakeSearchOsdbSubtitles,
			fakeFileSearchSubtitles,
			nil,
			&fakeMovieSub,
		},
		{
			fakeNewOsdbClient,
			fakeCheckOsdbClient,
			fakeLogInOsdbClient,
			fakeSearchOsdbSubtitles,
			fakeFileSearchSubtitlesError,
			nil,
			&fakeMovieSub,
		},
		{
			fakeNewOsdbClient,
			fakeCheckOsdbClient,
			fakeLogInOsdbClient,
			fakeSearchOsdbSubtitlesPartialError,
			fakeFileSearchSubtitlesError,
			nil,
			&fakeMovieSub,
		},
		{
			fakeNewOsdbClient,
			fakeCheckOsdbClient,
			fakeLogInOsdbClient,
			fakeSearchOsdbSubtitlesError,
			fakeFileSearchSubtitlesError,
			polochon.ErrNoSubtitleFound,
			nil,
		},
	}
	for _, situation := range situations {
		newOsdbClient = situation.newOsdbClient
		checkOsdbClient = situation.checkOsdbClient
		logInOsdbClient = situation.logInOsdbClient
		searchOsdbSubtitles = situation.searchOsdbSubtitles
		fileSearchSubtitles = situation.fileSearchSubtitles

		for _, video := range []interface{}{
			fakeMovie,
			fakeShowEpisode,
		} {
			sub, err := fakeProxy.searchSubtitles(video, "fakePath", polochon.FR, fakeLoggerEntry)
			if err != situation.expectedErr {
				log.Fatalf("Got error in searchMovieSubtitles: %q, wanted : %q", err, situation.expectedErr)
			}
			if err == nil {
				expected := situation.expectedSubtitle
				if !reflect.DeepEqual(sub.os, expected) {
					t.Errorf("Failed to get the good subtitle\nGot: %+v\nExpected: %+v", sub, expected)
				}
			}
		}
	}
}
