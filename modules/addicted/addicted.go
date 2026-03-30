package addicted

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/agnivade/levenshtein"
	"github.com/odwrtw/addicted"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	polochon "github.com/odwrtw/polochon/lib"
)

const httpTimeout = 30 * time.Second

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

// Custom errors
var (
	ErrMissingCredentials = errors.New("addicted: user and password are required")
	ErrInvalidToken       = errors.New("addicted: invalid token")
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
	if params.User == "" || params.Password == "" {
		return ErrMissingCredentials
	}

	client, err := addicted.NewWithAuth(params.User, params.Password)
	if err != nil {
		return err
	}

	a.client = client
	a.configured = true

	return nil
}

// buildToken encodes subtitle metadata into a human-readable token string.
func buildToken(s addicted.Subtitle) string {
	return fmt.Sprintf("%s - HearingImpaired:%t - Downloads:%d - %s",
		s.Title, s.HearingImpaired, s.Download, s.Link)
}

// parseTokenLink extracts the download link from a token built by buildToken.
func parseTokenLink(token string) (string, error) {
	idx := strings.LastIndex(token, " - ")
	if idx < 0 {
		return "", ErrInvalidToken
	}
	return token[idx+3:], nil
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

// getFilteredSubtitles fetches and filters subtitles by language for a show episode.
func (a *addictedProxy) getFilteredSubtitles(showTitle string, season, episode int, lang polochon.Language) (addicted.Subtitles, error) {
	langName, err := lang.Name()
	if err != nil {
		return nil, fmt.Errorf("addicted: language %q not supported", lang)
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	subtitles, err := a.client.GetSubtitles(ctx, showTitle, season, episode)
	if err != nil {
		return nil, err
	}

	filtered := subtitles.FilterByLang(strings.ToLower(langName))
	if len(filtered) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}
	return filtered, nil
}

func (a *addictedProxy) getShowSubtitle(reqEpisode *polochon.ShowEpisode, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	filteredSubs, err := a.getFilteredSubtitles(reqEpisode.ShowTitle, reqEpisode.Season, reqEpisode.Episode, lang)
	if err != nil {
		return nil, err
	}

	sort.Sort(addicted.ByDownloads(filteredSubs))

	subtitle := polochon.NewSubtitleFromVideo(reqEpisode, lang)
	data := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()

	if reqEpisode.ReleaseGroup == "" {
		// No release group specified: get the most downloaded subtitle
		r, err := a.client.Download(ctx, filteredSubs[0])
		if err != nil {
			return nil, err
		}
		defer r.Close()
		if _, err = data.ReadFrom(r); err != nil {
			return nil, err
		}
		subtitle.Data = data.Bytes()
		return subtitle, nil
	}

	subDist := 1000
	releaseGroup := strings.ToLower(reqEpisode.ReleaseGroup)
	var chosen *addicted.Subtitle

	for i := range filteredSubs {
		dist := levenshtein.ComputeDistance(releaseGroup, strings.ToLower(filteredSubs[i].Release))
		if dist < subDist {
			subDist = dist
			chosen = &filteredSubs[i]
		}
	}

	if chosen == nil {
		return nil, nil
	}

	log.WithFields(logrus.Fields{
		"release":  chosen.Release,
		"distance": subDist,
	}).Info("subtitle chosen")

	r, err := a.client.Download(ctx, *chosen)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	if _, err = data.ReadFrom(r); err != nil {
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

	filteredSubs, err := a.getFilteredSubtitles(reqEpisode.ShowTitle, reqEpisode.Season, reqEpisode.Episode, lang)
	if err != nil {
		return nil, err
	}

	entries := make([]*polochon.SubtitleEntry, 0, len(filteredSubs))
	for _, s := range filteredSubs {
		entries = append(entries, &polochon.SubtitleEntry{
			Language: lang,
			Release:  s.Release,
			Token:    buildToken(s),
		})
	}
	return entries, nil
}

// DownloadSubtitle implements the Subtitler interface.
func (a *addictedProxy) DownloadSubtitle(i any, entry *polochon.SubtitleEntry, _ *logrus.Entry) (*polochon.Subtitle, error) {
	video, ok := i.(polochon.Video)
	if !ok {
		return nil, fmt.Errorf("addicted: invalid argument")
	}

	link, err := parseTokenLink(entry.Token)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	r, err := a.client.Download(ctx, addicted.Subtitle{Link: link})
	if err != nil {
		return nil, err
	}
	defer r.Close()

	data := &bytes.Buffer{}
	if _, err := data.ReadFrom(r); err != nil {
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
