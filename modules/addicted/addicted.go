package addicted

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/arbovm/levenshtein"
	"github.com/odwrtw/addicted"
	"github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

var langTranslate = map[polochon.Language]string{
	polochon.EN: "english",
	polochon.FR: "french",
}

// Module constants
const (
	moduleName = "addicted"
)

// Params represents the module params
type Params struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Register a new Subtitler
func init() {
	polochon.RegisterSubtitler(moduleName, NewFromRawYaml)
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Subtitler, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New module
func New(params *Params) (polochon.Subtitler, error) {
	// Handle auth if the user and password are provided
	var client *addicted.Client

	var err error
	if params.User != "" && params.Password != "" {
		client, err = addicted.NewWithAuth(params.User, params.Password)
	} else {
		client, err = addicted.New()
	}
	if err != nil {
		return nil, err
	}

	return &addictedProxy{client: *client}, nil
}

type addictedProxy struct {
	client addicted.Client
}

// Name implements the Module interface
func (a *addictedProxy) Name() string {
	return moduleName
}

// Status implements the Module interface
func (a *addictedProxy) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

func (a *addictedProxy) getShowSubtitle(reqEpisode *polochon.ShowEpisode, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	// TODO: add year
	// TODO: handle release

	// if language not available in addicted
	addictedLang, ok := langTranslate[lang]
	if !ok {
		return nil, fmt.Errorf("addicted: language %q no supported", lang)
	}

	shows, err := a.client.GetTvShows()
	if err != nil {
		return nil, err
	}
	var guessID string
	guessDist := 1000
	for showName, showID := range shows {
		dist := levenshtein.Distance(strings.ToLower(showName), strings.ToLower(reqEpisode.ShowTitle))
		if dist < guessDist {
			guessDist = dist
			guessID = showID
		}
	}

	subtitles, err := a.client.GetSubtitles(guessID, reqEpisode.Season, reqEpisode.Episode)
	if err != nil {
		return nil, err
	}

	filteredSubs := subtitles.FilterByLang(addictedLang)
	if len(filteredSubs) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}

	sort.Sort(addicted.ByDownloads(filteredSubs))

	if reqEpisode.ReleaseGroup == "" {
		log.Info("No release group specified get the most downloaded subtitle")
		return &filteredSubs[0], err
	}

	subDist := 1000
	var subtitle polochon.Subtitle
	var release string

	for _, sub := range filteredSubs {
		dist := levenshtein.Distance(strings.ToLower(reqEpisode.ReleaseGroup), strings.ToLower(sub.Release))
		if dist < subDist {
			subDist = dist
			subtitle = &sub
			release = sub.Release
		}
	}
	log.Info("Subtitle chosen ", release, " whit distance ", subDist)
	return subtitle, err
}

// GetSubtitle implements the Subtitler interface
func (a *addictedProxy) GetSubtitle(i interface{}, lang polochon.Language, log *logrus.Entry) (polochon.Subtitle, error) {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		return a.getShowSubtitle(v, lang, log)
	default:
		return nil, fmt.Errorf("addicted: invalid argument")
	}
}
