package yifysubs

import (
	"errors"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/yifysubs"
)

// YifySubs holds the YifySubs module
type YifySubs struct {
	lang string
}

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
	ErrMissingSubtitleLang = errors.New("yifysub: missing subtitle language")
	ErrMissingImdbID       = errors.New("yifysub: missing imdb id")
)

// Register a new Subtitler
func init() {
	polochon.RegisterSubtitler(moduleName, New)
}

// Params represents the module params
type Params struct {
	Lang string `yaml:"lang"`
}

// New module
func New(p []byte) (polochon.Subtitler, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	if params.Lang == "" {
		return nil, ErrMissingSubtitleLang
	}

	language := polochon.Language(params.Lang)
	subLang, ok := langTranslate[language]
	if !ok {
		return nil, ErrInvalidSubtitleLang
	}

	return &YifySubs{
		lang: subLang,
	}, nil
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
func (y *YifySubs) GetMovieSubtitle(m *polochon.Movie, log *logrus.Entry) (polochon.Subtitle, error) {
	if m.ImdbID == "" {
		return nil, ErrMissingImdbID
	}

	// Get the subs for this movie
	subs, err := getSubtitles(m.ImdbID)
	if err != nil {
		return nil, err
	}

	// Only keep the configured lang
	subsByLang, ok := subs[y.lang]
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
func (y *YifySubs) GetShowSubtitle(s *polochon.ShowEpisode, log *logrus.Entry) (polochon.Subtitle, error) {
	// Return nil values
	return nil, nil
}
