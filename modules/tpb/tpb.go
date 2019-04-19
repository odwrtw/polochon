package tpb

import (
	"errors"
	"fmt"
	"strings"

	yaml "gopkg.in/yaml.v2"

	"github.com/odwrtw/guessit"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/tpb"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a torrenter
var _ polochon.Torrenter = (*TPB)(nil)

func init() {
	polochon.RegisterModule(&TPB{})
}

// Module constants
const (
	moduleName = "thepiratebay"
)

// TPB errors
var (
	ErrInvalidArgument = errors.New("tpb: invalid argument")
)

// Params represents the module params
type Params struct {
	ShowUsers  []string `yaml:"show_users"`
	MovieUsers []string `yaml:"movie_users"`
}

// TPB is a source for torrents
type TPB struct {
	Client     *tpb.Client
	MovieUsers []string
	ShowUsers  []string
	configured bool
}

const endpoint = "https://thepiratebay.org"

// Init implements the module interface
func (t *TPB) Init(p []byte) error {
	if t.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return t.InitWithParams(params)
}

// InitWithParams configures the module
func (t *TPB) InitWithParams(params *Params) error {
	t.Client = tpb.New(endpoint)
	t.MovieUsers = params.MovieUsers
	t.ShowUsers = params.ShowUsers
	t.configured = true
	return nil
}

// Name implements the Module interface
func (t *TPB) Name() string {
	return moduleName
}

// Status implements the Module interface
func (t *TPB) Status() (polochon.ModuleStatus, error) {
	torrents, err := t.SearchTorrents("black-mirror")
	if err != nil {
		return polochon.StatusFail, err
	}

	if len(torrents) == 0 {
		return polochon.StatusFail, nil
	}

	return polochon.StatusOK, nil
}

// Searcher represents an interface to search torrent
type Searcher interface {
	key() string
	users() []string
	videoType() string
	defaultQuality() string
	category() tpb.TorrentCategory
	setTorrents([]polochon.Torrent, *logrus.Entry)
	isValidGuess(guess *guessit.Response, log *logrus.Entry) bool
}

// NewSearcher will return a new Searcher
func (t *TPB) NewSearcher(i interface{}) (Searcher, error) {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		return &showSearcher{
			Episode: v,
			Users:   t.ShowUsers,
		}, nil
	case *polochon.Movie:
		return &movieSearcher{
			Movie: v,
			Users: t.MovieUsers,
		}, nil
	default:
		return nil, fmt.Errorf("unknown type")
	}
}

// GetTorrents implements the Torrenter interface
func (t *TPB) GetTorrents(i interface{}, log *logrus.Entry) error {
	// Create a new Searcher
	searcher, err := t.NewSearcher(i)
	if err != nil {
		return err
	}

	// Search for torrents
	torrents, err := t.Client.Search(tpb.SearchOptions{
		Key:      searcher.key(),
		OrderBy:  tpb.OrderBySeeds,
		Sort:     tpb.Desc,
		Category: searcher.category(),
	})
	if err != nil {
		return err
	}

	// Transform and filter the torrents we found
	pTorrents := t.transformTorrents(searcher, torrents, log)

	// Set the torrents into the video object
	searcher.setTorrents(pTorrents, log)
	return nil
}

// SearchTorrents implements the Torrenter interface
func (t *TPB) SearchTorrents(s string) ([]*polochon.Torrent, error) {
	// Search for torrents
	torrents, err := t.Client.Search(tpb.SearchOptions{
		Key:      s,
		OrderBy:  tpb.OrderBySeeds,
		Sort:     tpb.Desc,
		Category: tpb.Video,
	})
	if err != nil {
		return nil, err
	}

	pTorrents := []*polochon.Torrent{}
	for _, t := range torrents {
		pTorrents = append(pTorrents, &polochon.Torrent{
			Name:       t.Name,
			URL:        t.Magnet,
			Seeders:    t.Seeds,
			Leechers:   t.Peers,
			Source:     moduleName,
			UploadUser: t.User,
			Quality:    getQuality(t.Name),
		})
	}
	return pTorrents, nil
}

// transmforTorrents will filter and transform tpb.Torrent in polochon.Torrent
func (t *TPB) transformTorrents(s Searcher, list []tpb.Torrent, log *logrus.Entry) []polochon.Torrent {

	// Filter the torrents by user
	users := s.users()
	if len(users) > 0 {
		list = tpb.FilterByUsers(list, s.users())
	}

	// Use guessit to check the torrents with its infos
	guessClient := guessit.New("http://guessit.quimbo.fr/guess/")

	torrents := []polochon.Torrent{}
	for _, t := range list {
		torrentStr := torrentGuessitStr(&t)
		// Make a guess
		guess, err := guessClient.Guess(torrentStr)
		if err != nil {
			continue
		}

		// Check the guess validity
		if !s.isValidGuess(guess, log) {
			continue
		}

		// Set the default quality if none is defined
		if guess.Quality == "" {
			log.Debugf("tpb: default quality for %s", t.Name)
			guess.Quality = s.defaultQuality()
		}

		// Check that the Quality is valid
		torrentQuality := polochon.Quality(guess.Quality)
		if !torrentQuality.IsAllowed() {
			log.Debugf("tpb: unhandled quality: %q", torrentQuality)
			continue
		}

		log.WithFields(logrus.Fields{
			"torrent_quality": guess.Quality,
			"torrent_name":    torrentStr,
		}).Debug("Adding torrent to the list")

		torrents = append(torrents, polochon.Torrent{
			Name:       t.Name,
			URL:        t.Magnet,
			Seeders:    t.Seeds,
			Leechers:   t.Peers,
			Source:     moduleName,
			UploadUser: t.User,
			Quality:    torrentQuality,
		})
	}
	// Filter the torrents to keep only the best ones
	return polochon.FilterTorrents(torrents)
}

func torrentGuessitStr(t *tpb.Torrent) string {
	// Hack to make the torrent name look like a video name so that guessit
	// can guess the title, year and quality
	return strings.Replace(t.Name, " ", ".", -1) + ".mp4"
}

func getQuality(s string) polochon.Quality {
	for _, q := range []polochon.Quality{
		polochon.Quality480p,
		polochon.Quality720p,
		polochon.Quality1080p,
		polochon.Quality3D,
	} {
		if strings.Contains(s, string(q)) {
			return q
		}
	}
	return polochon.Quality("unknown")
}
