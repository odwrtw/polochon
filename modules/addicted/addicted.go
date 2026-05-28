package addicted

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/agnivade/levenshtein"
	"github.com/goccy/go-yaml"
	"github.com/odwrtw/addicted"

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
)

// Params represents the module params
type Params struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type addictedProxy struct {
	log        *slog.Logger
	client     *addicted.Client
	configured bool
}

// Init implements the module interface
func (a *addictedProxy) Init(p []byte, log *slog.Logger) error {
	if log == nil {
		log = slog.Default()
	}
	a.log = log.With("module", moduleName)

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
	if a.configured {
		return nil
	}

	if a.log == nil {
		a.log = slog.Default().With("module", moduleName)
	}

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

// Name implements the Module interface
func (a *addictedProxy) Name() string {
	return moduleName
}

// Status implements the Module interface
func (a *addictedProxy) Status() (polochon.ModuleStatus, error) {
	_, err := a.getShowSubtitle(context.Background(), &polochon.ShowEpisode{
		ShowTitle: "Black Mirror",
		Season:    1,
		Episode:   1,
	}, polochon.EN)
	if err != nil {
		return polochon.StatusFail, err
	}
	return polochon.StatusOK, nil
}

// getFilteredSubtitles fetches and filters subtitles by language for a show episode.
func (a *addictedProxy) getFilteredSubtitles(ctx context.Context, showTitle string, season, episode int, lang polochon.Language) (addicted.Subtitles, error) {
	langName, err := lang.Name()
	if err != nil {
		return nil, fmt.Errorf("addicted: language %q not supported", lang)
	}

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

func (a *addictedProxy) getShowSubtitle(ctx context.Context, reqEpisode *polochon.ShowEpisode, lang polochon.Language) (*polochon.Subtitle, error) {
	subCtx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	filteredSubs, err := a.getFilteredSubtitles(subCtx, reqEpisode.ShowTitle, reqEpisode.Season, reqEpisode.Episode, lang)
	if err != nil {
		return nil, err
	}

	sort.Sort(addicted.ByDownloads(filteredSubs))

	subtitle := polochon.NewSubtitleFromVideo(reqEpisode, lang)
	data := &bytes.Buffer{}

	dlCtx, dlCancel := context.WithTimeout(ctx, httpTimeout)
	defer dlCancel()

	if reqEpisode.ReleaseGroup == "" {
		// No release group specified: get the most downloaded subtitle
		r, err := a.client.Download(dlCtx, filteredSubs[0])
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

	a.log.Info("subtitle chosen", "release", chosen.Release, "distance", subDist)

	r, err := a.client.Download(dlCtx, *chosen)
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
func (a *addictedProxy) ListSubtitles(ctx context.Context, i any, lang polochon.Language) ([]*polochon.SubtitleEntry, error) {
	reqEpisode, ok := i.(*polochon.ShowEpisode)
	if !ok {
		return nil, polochon.ErrNotAvailable
	}

	subCtx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()

	filteredSubs, err := a.getFilteredSubtitles(subCtx, reqEpisode.ShowTitle, reqEpisode.Season, reqEpisode.Episode, lang)
	if err != nil {
		return nil, err
	}

	entries := make([]*polochon.SubtitleEntry, 0, len(filteredSubs))
	for _, s := range filteredSubs {
		entries = append(entries, &polochon.SubtitleEntry{
			Language:    lang,
			ID:          s.Link,
			Description: fmt.Sprintf("%s - %s (HI:%t, Downloads:%d)", s.Title, s.Release, s.HearingImpaired, s.Download),
		})
	}
	return entries, nil
}

// DownloadSubtitle implements the Subtitler interface.
func (a *addictedProxy) DownloadSubtitle(ctx context.Context, i any, entry *polochon.SubtitleEntry) (*polochon.Subtitle, error) {
	video, ok := i.(polochon.Video)
	if !ok {
		return nil, fmt.Errorf("addicted: invalid argument")
	}

	dlCtx, cancel := context.WithTimeout(ctx, httpTimeout)
	defer cancel()
	r, err := a.client.Download(dlCtx, addicted.Subtitle{Link: entry.ID})
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
func (a *addictedProxy) GetSubtitle(ctx context.Context, i any, lang polochon.Language) (*polochon.Subtitle, error) {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		return a.getShowSubtitle(ctx, v, lang)
	default:
		return nil, fmt.Errorf("addicted: invalid argument")
	}
}
