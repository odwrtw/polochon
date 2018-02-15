package yifysubtitles

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yifysubs"
	"github.com/sirupsen/logrus"
)

// YifySubs holds the YifySubs module
type YifySubs struct {
	Client Searcher
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

// Register a new Subtitler
func init() {
	polochon.RegisterSubtitler(moduleName, NewFromRawYaml)
}

// Params represents the module params
type Params struct{}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Subtitler, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New module
func New(params *Params) (polochon.Subtitler, error) {
	return &YifySubs{
		Client: yifysubs.New(endpoint),
	}, nil
}

// Name implements the Module interface
func (y *YifySubs) Name() string {
	return moduleName
}

// GetMovieSubtitle will get a movie subtitle
func (y *YifySubs) GetMovieSubtitle(m *polochon.Movie, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	if m.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	subLang, ok := langTranslate[lang]
	if !ok {
		fmt.Println(lang)
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
