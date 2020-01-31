package tpb

import (
	"context"
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/odwrtw/guessit"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/tpb"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a torrenter
var _ polochon.Torrenter = (*TPB)(nil)

var (
	defaultTimeout         = 30 * time.Second
	defaultGuessitEndpoint = "http://guessit.quimbo.fr/guess/"
)

func init() {
	polochon.RegisterModule(&TPB{})
}

// Module constants
const (
	moduleName = "thepiratebay"
)

// Params represents the module params
type Params struct {
	URLs            []string `yaml:"urls"`
	Timeout         string   `yaml:"timeout"`
	GuessitEndpoint string   `yaml:"guessit_endpoint"`
	ShowUsers       []string `yaml:"show_users"`
	MovieUsers      []string `yaml:"movie_users"`
}

// TPB is a source for torrents
type TPB struct {
	Client          *tpb.Client
	Timeout         time.Duration
	GuessitEndpoint string
	MovieUsers      []string
	ShowUsers       []string
	configured      bool
}

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
	t.Client = tpb.New(params.URLs...)
	t.MovieUsers = params.MovieUsers
	t.ShowUsers = params.ShowUsers
	t.configured = true
	t.GuessitEndpoint = params.GuessitEndpoint
	if t.GuessitEndpoint == "" {
		t.GuessitEndpoint = defaultGuessitEndpoint
	}

	var err error
	if params.Timeout == "" {
		t.Timeout = defaultTimeout
	} else {
		t.Timeout, err = time.ParseDuration(params.Timeout)
	}

	return err
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

// searcher represents an interface to search torrent
type searcher interface {
	key() string
	users() []string
	defaultQuality() string
	setTorrents([]polochon.Torrent)
	isValidGuess(guess *guessit.Response, log *logrus.Entry) bool
}

// newSearcher will return a new Searcher
func (t *TPB) newSearcher(i interface{}) (searcher, error) {
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
func (t *TPB) search(s string) ([]*tpb.Torrent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), t.Timeout)
	defer cancel()

	return t.Client.Search(ctx, s, &tpb.Options{
		OrderBy:  tpb.OrderBySeeds,
		Sort:     tpb.Desc,
		Category: tpb.Video,
	})
}

// GetTorrents implements the Torrenter interface
func (t *TPB) GetTorrents(i interface{}, log *logrus.Entry) error {
	// Create a new Searcher
	searcher, err := t.newSearcher(i)
	if err != nil {
		return err
	}

	torrents, err := t.search(searcher.key())
	if err != nil {
		return err
	}

	// Transform and filter the torrents we found
	pTorrents := t.transformTorrents(searcher, torrents, log)

	// Set the torrents into the video object
	searcher.setTorrents(pTorrents)
	return nil
}

// SearchTorrents implements the Torrenter interface
func (t *TPB) SearchTorrents(s string) ([]*polochon.Torrent, error) {
	results, err := t.search(s)
	if err != nil {
		return nil, err
	}

	torrents := make([]*polochon.Torrent, len(results))
	for i := 0; i < len(results); i++ {
		t := results[i]

		torrents[i] = &polochon.Torrent{
			Name:       t.Name,
			URL:        t.Magnet,
			Seeders:    t.Seeders,
			Leechers:   t.Leechers,
			Source:     moduleName,
			UploadUser: t.User,
			Quality:    getQuality(t.Name),
			Size:       int(t.Size),
		}
	}

	return torrents, nil
}

func filterTorrents(torrents []*tpb.Torrent, allowedUsers []string) []*tpb.Torrent {
	filteredList := []*tpb.Torrent{}
	if len(torrents) == 0 || len(allowedUsers) == 0 {
		return filteredList
	}

	// Create a set of users
	userMap := map[string]struct{}{}
	for _, u := range allowedUsers {
		userMap[u] = struct{}{}
	}

	for _, t := range torrents {
		if _, ok := userMap[t.User]; ok {
			filteredList = append(filteredList, t)
		}
	}

	return filteredList
}

// transformTorrents will filter and transform tpb.Torrent in polochon.Torrent
func (t *TPB) transformTorrents(s searcher, list []*tpb.Torrent, log *logrus.Entry) []polochon.Torrent {
	// Use guessit to check the torrents with its infos
	guessClient := guessit.New(t.GuessitEndpoint)

	torrents := []polochon.Torrent{}
	for _, t := range filterTorrents(list, s.users()) {
		torrentStr := torrentGuessitStr(t)
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
			Seeders:    t.Seeders,
			Leechers:   t.Leechers,
			Source:     moduleName,
			UploadUser: t.User,
			Quality:    torrentQuality,
			Size:       int(t.Size),
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
