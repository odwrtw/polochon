package yifysubtitles

import (
	"errors"
	"fmt"

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

const endpoint = "https://yifysubtitles.com"

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

	y.Client = yifysubs.New(endpoint)
	y.configured = true
	return nil
}

// Name implements the Module interface
func (y *YifySubs) Name() string {
	return moduleName
}

// Status implements the Module interface
func (y *YifySubs) Status() (polochon.ModuleStatus, error) {
	results, err := y.Client.SearchByLang("tt0133093", "English")
	if err != nil {
		return polochon.StatusFail, err
	}
	if len(results) == 0 {
		return polochon.StatusFail, fmt.Errorf("failed to find subtitles")
	}
	return polochon.StatusOK, nil
}

// GetMovieSubtitle will get a movie subtitle
func (y *YifySubs) GetMovieSubtitle(m *polochon.Movie, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	if m.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	subLang, ok := langTranslate[lang]
	if !ok {
		return nil, ErrInvalidSubtitleLang
	}

	// Get the subs for this movie
	subs, err := y.Client.SearchByLang(m.ImdbID, subLang)
	switch err {
	case nil:
		//continue
	case yifysubs.ErrNoSubtitleFound:
		return nil, polochon.ErrNoSubtitleFound
	default:
		return nil, err
	}

	// No sub found ?
	if len(subs) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}

	return subs[0], nil
}

// GetShowSubtitle implements the Subtitler interface but will not be used here
func (y *YifySubs) GetShowSubtitle(s *polochon.ShowEpisode, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	// Return nil values
	return nil, polochon.ErrNoSubtitleFound
}

// GetSubtitle implements the Subtitler interface
func (y *YifySubs) GetSubtitle(i interface{}, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	switch v := i.(type) {
	case *polochon.Movie:
		return y.GetMovieSubtitle(v, lang, log)
	default:
		return nil, fmt.Errorf("opensub: invalid argument")
	}
}
