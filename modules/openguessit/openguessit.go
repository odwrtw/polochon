package openguessit

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/odwrtw/guessit"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a guesser
var _ polochon.Guesser = (*OpenGuessit)(nil)

// Register openguessit as a Guesser
func init() {
	polochon.RegisterModule(&OpenGuessit{
		GuessitClient: guessit.New("http://guessit.quimbo.fr/guess"),
	})
}

// Module constants
const (
	moduleName = "openguessit"
)

// OpenGuessit is a mix of opensubtitle and guessit
type OpenGuessit struct {
	GuessitClient *guessit.Client
}

// Init implements the module interface
func (og *OpenGuessit) Init(p []byte) error {
	return nil
}

// Name implements the Module interface
func (og *OpenGuessit) Name() string {
	return moduleName
}

// Status implements the Module interface
func (og *OpenGuessit) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// GuessMetadata guess the metadata of a file
func (og *OpenGuessit) GuessMetadata(file *polochon.File) (*polochon.VideoMetadata, error) {
	filePath := filepath.Base(file.Path)
	guess, err := og.GuessitClient.Guess(filePath)
	if err != nil {
		return nil, err
	}

	return &polochon.VideoMetadata{
		Quality:      polochon.Quality(guess.Quality),
		ReleaseGroup: guess.ReleaseGroup,
		AudioCodec:   guess.AudioCodec,
		VideoCodec:   guess.VideoCodec,
		Container:    guess.Container,
	}, nil
}

// Guess implements the Guesser interface
func (og *OpenGuessit) Guess(file polochon.File, movieConf polochon.MovieConfig, showConf polochon.ShowConfig, log *logrus.Entry) (polochon.Video, error) {
	filename := filepath.Base(file.Path)
	guess, err := og.GuessitClient.Guess(filename)
	if err != nil {
		return nil, err
	}

	// Format the title
	guess.Title = toUpperCaseFirst(guess.Title)

	metadata := polochon.VideoMetadata{
		Quality:      polochon.Quality(guess.Quality),
		ReleaseGroup: guess.ReleaseGroup,
		AudioCodec:   guess.AudioCodec,
		VideoCodec:   guess.VideoCodec,
		Container:    guess.Container,
	}

	switch guess.Type {
	case "movie":
		return &polochon.Movie{
			VideoMetadata: metadata,
			MovieConfig:   movieConf,
			File:          file,
			Title:         guess.Title,
			Year:          guess.Year,
		}, nil
	case "episode":
		show := polochon.NewShow(showConf)
		show.Year = guess.Year
		show.Title = guess.Title
		return &polochon.ShowEpisode{
			VideoMetadata: metadata,
			ShowConfig:    showConf,
			Show:          show,
			File:          file,
			ShowTitle:     guess.Title,
			Episode:       guess.Episode,
			Season:        guess.Season,
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
