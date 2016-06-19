package kickass

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/odwrtw/kickass"
	"github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/polochon/lib/explorer"
)

// Module constants
const (
	moduleName = "kickass"
)

// Category represents the category of a movie or a tv show
type Category string

// Available categories
const (
	MoviesCategory Category = "movies"
	ShowsCategory  Category = "tv"
)

// Custom errors
var (
	ErrInvalidType = errors.New("kickass: invalid type")
)

// Register yts as a Torrenter
func init() {
	polochon.RegisterTorrenter(moduleName, NewFromRawYaml)
}

// Params represents the module params
type Params struct {
	ShowsUsers  []string `yaml:"shows_users"`
	MoviesUsers []string `yaml:"movies_users"`
}

// Kickass holds the kickass client
type Kickass struct {
	client *kickass.Client
	*Params
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Torrenter, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return NewTorrenter(params)
}

// NewTorrenter returns a new kickass torrenter
func NewTorrenter(params *Params) (polochon.Torrenter, error) {
	return &Kickass{
		client: kickass.New(),
		Params: params,
	}, nil
}

// NewExplorer returns a new explorer
func NewExplorer(params *Params) (explorer.Explorer, error) {
	return &Kickass{
		client: kickass.New(),
		Params: params,
	}, nil
}

// Name implements the Module interface
func (k *Kickass) Name() string {
	return moduleName
}

func torrentGuessitStr(t *kickass.Torrent) string {
	// Hack to make the torrent name look like a video name so that guessit
	// can guess the title, year and quality
	return strings.Replace(t.Name, " ", ".", -1) + ".mp4"
}
