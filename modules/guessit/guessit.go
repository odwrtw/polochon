package guessit

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/odwrtw/whatsthis"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a guesser
var _ polochon.Guesser = (*Guessit)(nil)

// Register guessit as a Guesser
func init() {
	polochon.RegisterModule(&Guessit{})
}

// Module constants
const (
	moduleName = "guessit"
)

// Guessit implements the Guesser interface using go-guessit
type Guessit struct{}

// Init implements the module interface
func (g *Guessit) Init(_ []byte) error {
	return nil
}

// Name implements the Module interface
func (g *Guessit) Name() string {
	return moduleName
}

// Status implements the Module interface
func (g *Guessit) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// GuessMetadata guess the metadata of a file
func (g *Guessit) GuessMetadata(file *polochon.File, _ *logrus.Entry) (*polochon.VideoMetadata, error) {
	filePath := filepath.Base(file.Path)
	guess := whatsthis.Video(filePath)

	return &polochon.VideoMetadata{
		Quality:      polochon.Quality(guess.ScreenSize),
		ReleaseGroup: guess.ReleaseGroup,
		AudioCodec:   guess.AudioCodec,
		VideoCodec:   guess.VideoCodec,
		Container:    guess.Container,
	}, nil
}

// Guess implements the Guesser interface
func (g *Guessit) Guess(file polochon.File, movieConf polochon.MovieConfig, showConf polochon.ShowConfig, _ *logrus.Entry) (polochon.Video, error) {
	filename := filepath.Base(file.Path)
	guess := whatsthis.Video(filename)

	// Format the title
	title := toUpperCaseFirst(guess.Title)

	var video polochon.Video

	switch guess.Type {
	case whatsthis.Movie:
		video = &polochon.Movie{
			MovieConfig: movieConf,
			Title:       title,
			Year:        guess.Year,
		}
	case whatsthis.Episode:
		show := polochon.NewShow(showConf)
		show.Year = guess.Year
		show.Title = title
		video = &polochon.ShowEpisode{
			ShowConfig: showConf,
			Show:       show,
			ShowTitle:  title,
			Episode:    guess.Episode,
			Season:     guess.Season,
		}
	default:
		return nil, fmt.Errorf("guessit: invalid guess type: %s", guess.Type)
	}

	video.SetFile(file)
	video.SetMetadata(&polochon.VideoMetadata{
		Quality:      polochon.Quality(guess.ScreenSize),
		ReleaseGroup: guess.ReleaseGroup,
		AudioCodec:   guess.AudioCodec,
		VideoCodec:   guess.VideoCodec,
		Container:    guess.Container,
	})

	return video, nil
}

// toUpperCaseFirst is an helper to get the uppercase first of a string
func toUpperCaseFirst(s string) string {
	retStr := []string{}
	for str := range strings.SplitSeq(s, " ") {
		if len(str) > 1 {
			str = strings.ToUpper(string(str[0])) + str[1:]
		}
		retStr = append(retStr, str)
	}

	return strings.Join(retStr, " ")
}
