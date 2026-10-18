package guessit

import (
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/odwrtw/whatsthis"

	polochon "github.com/odwrtw/polochon/lib"
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

// Guessit implements the Guesser interface using whatsthis
type Guessit struct{}

// Init implements the module interface
func (g *Guessit) Init(_ []byte, _ *slog.Logger) error {
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
func (g *Guessit) GuessMetadata(file *polochon.File) (*polochon.VideoMetadata, error) {
	guess := whatsthis.File(filepath.Base(file.Path))
	return metadata(guess), nil
}

// Guess implements the Guesser interface
func (g *Guessit) Guess(file polochon.File, movieConf polochon.MovieConfig, showConf polochon.ShowConfig) (polochon.Video, error) {
	filename := filepath.Base(file.Path)
	guess := whatsthis.File(filename)
	if guess.Title == "" {
		return nil, polochon.ErrGuessingVideo
	}
	title := guess.Title

	var video polochon.Video

	switch guess.Type {
	case whatsthis.Movie:
		video = &polochon.Movie{
			MovieConfig: movieConf,
			Title:       title,
			Year:        guess.Year,
		}
	case whatsthis.Episode:
		if guess.Episode <= 0 {
			return nil, polochon.ErrGuessingVideo
		}
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
	video.SetMetadata(metadata(guess))

	return video, nil
}

func metadata(guess whatsthis.Info) *polochon.VideoMetadata {
	return &polochon.VideoMetadata{
		Quality:      polochon.Quality(guess.ScreenSize),
		ReleaseGroup: guess.ReleaseGroup,
		AudioCodec:   guess.AudioCodec,
		VideoCodec:   guess.VideoCodec,
		Container:    guess.Container,
	}
}
