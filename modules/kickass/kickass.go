package kickass

import (
	"errors"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/guessit"
	"github.com/odwrtw/kickass"
	"github.com/odwrtw/polochon/lib"
)

// Module constants
const (
	moduleName = "kickass"
)

// Custom errors
var (
	ErrInvalidType = errors.New("kickass: invalid type")
)

// Register yts as a Torrenter
func init() {
	polochon.RegisterTorrenter(moduleName, NewFromRawYaml)
}

// Searcher is an interface to search for torrents
type Searcher interface {
	validate() error
	searchStr() string
	category() string
	users() []string
	isValidGuess(*guessit.Response, *logrus.Entry) bool
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

	return New(params)
}

// New returns a new kickass
func New(params *Params) (polochon.Torrenter, error) {
	return &Kickass{
		client: kickass.New(),
		Params: params,
	}, nil
}

// Name implements the Module interface
func (k *Kickass) Name() string {
	return moduleName
}

// GetTorrents implements the torrenter interface
func (k *Kickass) GetTorrents(i interface{}, log *logrus.Entry) error {
	switch video := i.(type) {
	case *polochon.Movie:
		m := NewMovieSearcher(video, k.MoviesUsers)
		t, err := k.search(m, log)
		if err != nil {
			return err
		}
		m.Torrents = t
		return nil
	case *polochon.ShowEpisode:
		s := NewShowEpisodeSearcher(video, k.ShowsUsers)
		t, err := k.search(s, log)
		if err != nil {
			return err
		}
		s.Torrents = t
		return nil
	default:
		return ErrInvalidType
	}
}

func (k *Kickass) search(s Searcher, log *logrus.Entry) ([]polochon.Torrent, error) {
	users := s.users()
	result := []polochon.Torrent{}

	log = log.WithFields(logrus.Fields{
		"search_category": s.category(),
		"search_string":   s.searchStr(),
	})

	if err := s.validate(); err != nil {
		log.Error(err)
		return nil, err
	}

	for _, u := range users {
		torrents, err := k.searchUser(s, log, u)
		if err != nil {
			return nil, err
		}

		result = append(result, torrents...)
	}

	return polochon.FilterTorrents(result), nil
}

func (k *Kickass) searchUser(s Searcher, log *logrus.Entry, user string) ([]polochon.Torrent, error) {
	query := &kickass.Query{
		User:     user,
		OrderBy:  "seeders",
		Order:    "desc",
		Category: s.category(),
		Search:   s.searchStr(),
	}
	log = log.WithField("search_user", user)

	torrents, err := k.client.Search(query)
	if err != nil {
		return nil, err
	}

	result := []polochon.Torrent{}
	for _, t := range torrents {
		// Hack to make the torrent name look like a video name so that guessit
		// can guess the title, year and quality
		torrentStr := strings.Replace(t.Name, " ", ".", -1) + ".mp4"
		guess, err := guessit.Guess(torrentStr)
		if err != nil {
			continue
		}

		if !s.isValidGuess(guess, log) {
			continue
		}

		// Default quality
		if s.category() == "tv" && guess.Quality == "" {
			guess.Quality = string(polochon.Quality480p)
		}

		// Get the torrent quality
		torrentQuality := polochon.Quality(guess.Quality)
		if !torrentQuality.IsAllowed() {
			log.Infof("kickass: unhandled quality: %q", torrentQuality)
			continue
		}

		log.WithFields(logrus.Fields{
			"torrent_quality": guess.Quality,
			"torrent_name":    torrentStr,
		}).Debug("Adding torrent to the list")

		// Add the torrent
		result = append(result, polochon.Torrent{
			Quality: torrentQuality,
			URL:     t.MagnetURL,
		})
	}

	return result, nil
}
