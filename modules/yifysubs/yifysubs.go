package yifysubs

import (
	"errors"
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yifysubs"
	"github.com/sirupsen/logrus"
)

// YifySubs holds the YifySubs module
type YifySubs struct{}

// Module constants
const (
	moduleName = "yifysubs"
)

var langTranslate = map[polochon.Language]string{
	polochon.EN: "english",
	polochon.FR: "french",
}

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

	return &YifySubs{}, nil
}

// Name implements the Module interface
func (y *YifySubs) Name() string {
	return moduleName
}

// Function to be overwritten during the tests
var getSubtitles = func(imdbID string) (map[string][]yifysubs.Subtitle, error) {
	return yifysubs.GetSubtitles(imdbID)
}

// GetMovieSubtitle will get a movie subtitle
func (y *YifySubs) GetMovieSubtitle(m *polochon.Movie, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	if m.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	// Get the subs for this movie
	subs, err := getSubtitles(m.ImdbID)
	if err != nil {
		return nil, err
	}

	subLang, ok := langTranslate[lang]
	if !ok {
		fmt.Println(lang)
		return nil, ErrInvalidSubtitleLang
	}

	// Only keep the configured lang
	subsByLang, ok := subs[subLang]
	if !ok {
		return nil, polochon.ErrNoSubtitleFound
	}

	// Search for the best rated sub
	var bestRating int
	var bestSub *yifysubs.Subtitle
	for _, s := range subsByLang {
		if s.Rating < bestRating {
			continue
		}

		bestRating = s.Rating
		bestSub = &s
	}

	// No sub found ?
	if bestSub == nil {
		return nil, polochon.ErrNoSubtitleFound
	}

	return bestSub, nil
}

// GetShowSubtitle implements the Subtitler interface but will not be used here
func (y *YifySubs) GetShowSubtitle(s *polochon.ShowEpisode, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	// Return nil values
	return nil, polochon.ErrNoSubtitleFound
}

func (y *YifySubs) GetSubtitle(i interface{}, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	switch v := i.(type) {
	case *polochon.Movie:
		return y.GetMovieSubtitle(v, lang, log)
	default:
		return nil, fmt.Errorf("opensub: invalid argument")
	}
}
