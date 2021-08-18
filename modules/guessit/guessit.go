package guessit

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/odwrtw/guessit"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
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

// Params represents the module params
type Params struct {
	Endpoint string `yaml:"endpoint"`
}

// Guessit is a mix of opensubtitle and guessit
type Guessit struct {
	client     *guessit.Client
	configured bool
}

// Init implements the module interface
func (g *Guessit) Init(p []byte) error {
	if g.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return g.InitWithParams(params)
}

// InitWithParams configures the module
func (g *Guessit) InitWithParams(params *Params) error {
	g.client = guessit.New(params.Endpoint)
	g.configured = true
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
func (g *Guessit) GuessMetadata(file *polochon.File, log *logrus.Entry) (*polochon.VideoMetadata, error) {
	filePath := filepath.Base(file.Path)
	guess, err := g.client.Guess(filePath)
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
func (g *Guessit) Guess(file polochon.File, movieConf polochon.MovieConfig, showConf polochon.ShowConfig, log *logrus.Entry) (polochon.Video, error) {
	filename := filepath.Base(file.Path)
	guess, err := g.client.Guess(filename)
	if err != nil {
		return nil, err
	}

	// Format the title
	guess.Title = toUpperCaseFirst(guess.Title)

	var video polochon.Video

	switch guess.Type {
	case "movie":
		video = &polochon.Movie{
			MovieConfig: movieConf,
			Title:       guess.Title,
			Year:        guess.Year,
		}
	case "episode":
		show := polochon.NewShow(showConf)
		show.Year = guess.Year
		show.Title = guess.Title
		video = &polochon.ShowEpisode{
			ShowConfig: showConf,
			Show:       show,
			ShowTitle:  guess.Title,
			Episode:    guess.Episode,
			Season:     guess.Season,
		}
	default:
		return nil, fmt.Errorf("guessit: invalid guess type: %s", guess.Type)
	}

	video.SetFile(file)
	video.SetMetadata(&polochon.VideoMetadata{
		Quality:      polochon.Quality(guess.Quality),
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
	strs := strings.Split(s, " ")
	for _, str := range strs {
		if len(str) > 1 {
			str = strings.ToUpper(string(str[0])) + str[1:]
		}
		retStr = append(retStr, str)
	}

	return strings.Join(retStr, " ")
}
