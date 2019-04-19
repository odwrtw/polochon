package addicted

import (
	"fmt"
	"sort"
	"strings"

	"github.com/arbovm/levenshtein"
	"github.com/odwrtw/addicted"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Make sure that the module is a subtitler
var _ polochon.Subtitler = (*addictedProxy)(nil)

// Register a new Subtitler
func init() {
	polochon.RegisterModule(&addictedProxy{})
}

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

type addictedProxy struct {
	client     *addicted.Client
	configured bool
}

// Init implements the module interface
func (a *addictedProxy) Init(p []byte) error {
	if a.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return a.InitWithParams(params)
}

// InitWithParams configures the module
func (a *addictedProxy) InitWithParams(params *Params) error {
	// Handle auth if the user and password are provided
	client := &addicted.Client{}

	var err error
	if params.User != "" && params.Password != "" {
		client, err = addicted.NewWithAuth(params.User, params.Password)
	} else {
		client, err = addicted.New()
	}
	if err != nil {
		return err
	}

	a.client = client
	a.configured = true

	return nil
}

// Name implements the Module interface
func (a *addictedProxy) Name() string {
	return moduleName
}

// Status implements the Module interface
func (a *addictedProxy) Status() (polochon.ModuleStatus, error) {
	_, err := a.getShowSubtitle(&polochon.ShowEpisode{
		ShowTitle: "Black Mirror",
		Season:    1,
		Episode:   1,
	}, polochon.EN, logrus.NewEntry(logrus.New()))
	if err != nil {
		return polochon.StatusFail, err
	}
	return polochon.StatusOK, nil
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
		// No release group specified get the most downloaded subtitle
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
