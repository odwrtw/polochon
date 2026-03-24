package addicted

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/agnivade/levenshtein"
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
	var client *addicted.Client

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

func (a *addictedProxy) getShowSubtitle(reqEpisode *polochon.ShowEpisode, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	// TODO: add year
	// TODO: handle release

	langName, err := lang.Name()
	if err != nil {
		return nil, fmt.Errorf("addicted: language %q no supported", lang)
	}
	addictedLang := strings.ToLower(langName)

	shows, err := a.client.GetTvShows()
	if err != nil {
		return nil, err
	}
	var guessID string
	guessDist := 1000
	for showName, showID := range shows {
		dist := levenshtein.ComputeDistance(strings.ToLower(showName), strings.ToLower(reqEpisode.ShowTitle))
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

	subtitle := polochon.NewSubtitleFromVideo(reqEpisode, lang)

	data := &bytes.Buffer{}

	if reqEpisode.ReleaseGroup == "" {
		// No release group specified get the most downloaded subtitle
		_, err = data.ReadFrom(&filteredSubs[0])
		if err != nil {
			return nil, err
		}

		subtitle.Data = data.Bytes()
		return subtitle, nil
	}

	subDist := 1000
	var release string
	var chosen *addicted.Subtitle

	for _, sub := range filteredSubs {
		dist := levenshtein.ComputeDistance(strings.ToLower(reqEpisode.ReleaseGroup), strings.ToLower(sub.Release))
		if dist < subDist {
			subDist = dist
			chosen = &sub
			release = sub.Release
		}
	}

	if chosen == nil {
		return nil, nil
	}

	log.WithFields(logrus.Fields{
		"release":  release,
		"distance": subDist,
	}).Info("subtitle chosen")

	_, err = data.ReadFrom(&filteredSubs[0])
	if err != nil {
		return nil, err
	}

	subtitle.Data = data.Bytes()
	return subtitle, nil
}

// ListSubtitles implements the Subtitler interface.
func (a *addictedProxy) ListSubtitles(i any, lang polochon.Language, log *logrus.Entry) ([]*polochon.SubtitleEntry, error) {
	reqEpisode, ok := i.(*polochon.ShowEpisode)
	if !ok {
		return nil, polochon.ErrNotAvailable
	}

	langName, err := lang.Name()
	if err != nil {
		return nil, fmt.Errorf("addicted: language %q not supported", lang)
	}
	addictedLang := strings.ToLower(langName)

	shows, err := a.client.GetTvShows()
	if err != nil {
		return nil, err
	}
	var guessID string
	guessDist := 1000
	for showName, showID := range shows {
		dist := levenshtein.ComputeDistance(strings.ToLower(showName), strings.ToLower(reqEpisode.ShowTitle))
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

	entries := make([]*polochon.SubtitleEntry, 0, len(filteredSubs))
	for _, s := range filteredSubs {
		entries = append(entries, &polochon.SubtitleEntry{
			Language: lang,
			Release:  s.Release,
			Token:    s.Link,
		})
	}
	return entries, nil
}

// addictedBaseURL mirrors the base URL used by the addicted library.
const addictedBaseURL = "https://www.addic7ed.com/"

// DownloadSubtitle implements the Subtitler interface.
func (a *addictedProxy) DownloadSubtitle(i any, entry *polochon.SubtitleEntry, _ *logrus.Entry) (*polochon.Subtitle, error) {
	video, ok := i.(polochon.Video)
	if !ok {
		return nil, fmt.Errorf("addicted: invalid argument")
	}

	// entry.Token is the Link field (e.g. "/updated/5/1234/5") — same URL the
	// addicted library uses in Subtitle.Read() to fetch the file.
	resp, err := a.client.Get(addictedBaseURL+entry.Token[1:], true)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	data := &bytes.Buffer{}
	if _, err := data.ReadFrom(resp.Body); err != nil {
		return nil, err
	}

	sub := polochon.NewSubtitleFromVideo(video, entry.Language)
	sub.Data = data.Bytes()
	return sub, nil
}

// GetSubtitle implements the Subtitler interface
func (a *addictedProxy) GetSubtitle(i any, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		return a.getShowSubtitle(v, lang, log)
	default:
		return nil, fmt.Errorf("addicted: invalid argument")
	}
}
