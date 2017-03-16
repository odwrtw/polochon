package openguessit

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/guessit"
	"github.com/odwrtw/polochon/lib"
)

// Video types
const (
	MovieType   = "movie"
	ShowType    = "episode"
	UnknownType = "unknown"
)

// Module constants
const (
	moduleName = "openguessit"
)

// Errors
var (
	ErrShowNameUnknown = errors.New("show title unknown")
)

// Register openguessit as a Guesser
func init() {
	polochon.RegisterGuesser(moduleName, NewFromRawYaml)
}

// OpenGuessit is a mix of opensubtitle and guessit
type OpenGuessit struct {
	GuessitClient *guessit.Client
}

// New returns an new openguessit
func New() (polochon.Guesser, error) {
	return &OpenGuessit{
		GuessitClient: guessit.New("http://guessit.quimbo.fr/guess/"),
	}, nil
}

// NewFromRawYaml returns an new openguessit
func NewFromRawYaml(p []byte) (polochon.Guesser, error) {
	return New()
}

// Name implements the Module interface
func (og *OpenGuessit) Name() string {
	return moduleName
}

// Guess implements the Guesser interface
func (og *OpenGuessit) Guess(file polochon.File, movieConf polochon.MovieConfig, showConf polochon.ShowConfig, log *logrus.Entry) (polochon.Video, error) {
	guess, err := og.GuessitClient.Guess(filepath.Base(file.Path))
	if err != nil {
		return nil, err
	}

	// Format the title
	guess.Title = toUpperCaseFirst(guess.Title)

	switch guess.Type {
	case "movie":
		return &polochon.Movie{
			MovieConfig: movieConf,
			File:        file,
			Title:       guess.Title,
			Year:        guess.Year,
		}, nil
	case "episode":
		return &polochon.ShowEpisode{
			ShowConfig: showConf,
			File:       file,
			Title:      guess.Title,
			ShowTitle:  guess.Title,
			Episode:    guess.Episode,
			Season:     guess.Season,
		}, nil
	default:
		return nil, fmt.Errorf("openguessit: invalid guess type: %s", guess.Type)
	}
}

// toUpperCaseFirst is an helper to get the uppercase first of a string
func toUpperCaseFirst(s string) string {
	retStr := []string{}
	strs := strings.Split(s, " ")
	for _, str := range strs {
		if len(str) > 1 {
			str = strings.ToUpper(string(str[0])) + str[1:]
		}
		retStr = append(retStr, str)
	}

	return strings.Join(retStr, " ")
}
