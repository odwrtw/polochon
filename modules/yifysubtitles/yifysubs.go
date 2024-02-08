package yifysubtitles

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/agnivade/levenshtein"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yifysubs"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a subtitler
var _ polochon.Subtitler = (*YifySubs)(nil)

func init() {
	polochon.RegisterModule(&YifySubs{})
}

// YifySubs holds the YifySubs module
type YifySubs struct {
	Client     Searcher
	configured bool
}

// Module constants
const (
	moduleName = "yifysubs"
)

var langTranslate = map[polochon.Language]string{
	polochon.EN: "English",
	polochon.FR: "French",
}

// Searcher is an interface to search subtitles
type Searcher interface {
	SearchByLang(imdbID, lang string) ([]*yifysubs.Subtitle, error)
}

// Errors
var (
	ErrInvalidSubtitleLang = errors.New("yifysub: invalid subtitle language")
	ErrMissingImdbID       = errors.New("yifysub: missing imdb id")
)

// Init implements the module interface
func (y *YifySubs) Init(p []byte) error {
	if y.configured {
		return nil
	}

	y.Client = yifysubs.NewDefault()
	y.configured = true
	return nil
}

// Name implements the Module interface
func (y *YifySubs) Name() string {
	return moduleName
}

// Status implements the Module interface
func (y *YifySubs) Status() (polochon.ModuleStatus, error) {
	_, err := y.Client.SearchByLang("tt0133093", "English")
	if err != nil {
		return polochon.StatusFail, err
	}

	return polochon.StatusOK, nil
}

func (y *YifySubs) getMovieSubtitle(m *polochon.Movie, lang polochon.Language, log *logrus.Entry) (*yifysubs.Subtitle, error) {
	if m.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	subLang, ok := langTranslate[lang]
	if !ok {
		return nil, ErrInvalidSubtitleLang
	}

	// Get the subs for this movie
	subs, err := y.Client.SearchByLang(m.ImdbID, subLang)
	if err != nil {
		if err == yifysubs.ErrNoSubtitleFound {
			return nil, polochon.ErrNoSubtitleFound
		}

		return nil, err
	}

	var selected *yifysubs.Subtitle
	minScore := 1000

	release := filepath.Base(m.PathWithoutExt())
	for _, sub := range subs {
		for _, subRelease := range sub.Releases {
			dist := levenshtein.ComputeDistance(release, subRelease)
			if dist < minScore {
				selected = sub
				minScore = dist
			}
		}
	}

	if selected == nil {
		return nil, polochon.ErrNoSubtitleFound
	}

	return selected, nil
}

// GetMovieSubtitle will get a movie subtitle
func (y *YifySubs) GetMovieSubtitle(m *polochon.Movie, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	s, err := y.getMovieSubtitle(m, lang, log)
	if err != nil {
		return nil, err
	}

	data := &bytes.Buffer{}
	_, err = data.ReadFrom(s)
	if err != nil {
		return nil, err
	}

	sub := polochon.NewSubtitleFromVideo(m, lang)
	sub.Data = data.Bytes()
	return sub, nil
}

// GetSubtitle implements the Subtitler interface
func (y *YifySubs) GetSubtitle(i interface{}, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	switch v := i.(type) {
	case *polochon.Movie:
		return y.GetMovieSubtitle(v, lang, log)
	default:
		return nil, fmt.Errorf("yifysubs: can only search for movie subtitles")
	}
}
