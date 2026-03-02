package podnapisi

import (
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/agnivade/levenshtein"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

var _ polochon.Subtitler = (*Client)(nil)

func init() {
	polochon.RegisterModule(&Client{})
}

// Client represents the Podnapisi API client.
type Client struct{}

// Init implements the polochon.Module interface.
func (c *Client) Init(_ []byte) error {
	return nil
}

// Name implements the polochon.Module interface.
func (c *Client) Name() string {
	return moduleName
}

const moduleName = "podnapisi"

// Status implements the polochon.Module interface.
func (c *Client) Status() (polochon.ModuleStatus, error) {
	params := url.Values{}
	params.Set("keywords", "the matrix")
	params.Set("language", "en")
	params.Set("movie_type", "movie")
	params.Set("year", "1999")

	subs, err := podnapisiSearch(params)
	if err != nil || len(subs) == 0 {
		return polochon.StatusFail, err
	}

	return polochon.StatusOK, nil
}

// GetSubtitle implements the polochon.Subtitler interface.
func (c *Client) GetSubtitle(i any, lang polochon.Language, _ *logrus.Entry) (*polochon.Subtitle, error) {
	langCode, ok := langMap[lang]
	if !ok {
		return nil, polochon.ErrNoSubtitleFound
	}

	var params url.Values

	switch resource := i.(type) {
	case *polochon.Movie:
		params = movieParams(resource, langCode)
	case *polochon.ShowEpisode:
		params = showEpisodeParams(resource, langCode)
	default:
		return nil, ErrNotAVideo
	}

	subs, err := podnapisiSearch(params)
	if err != nil {
		return nil, err
	}

	// The API does not reliably filter by language; do it client-side.
	filtered := subs[:0]
	for _, s := range subs {
		if s.Language == langCode {
			filtered = append(filtered, s)
		}
	}
	subs = filtered

	if len(subs) == 0 {
		return nil, polochon.ErrNoSubtitleFound
	}

	video, ok := i.(polochon.Video)
	if !ok {
		return nil, ErrNotAVideo
	}

	release := filepath.Base(video.GetFile().PathWithoutExt())

	selected, err := bestSubtitle(subs, release)
	if err != nil {
		return nil, err
	}

	data, err := podnapisiDownload(selected.PublishID, release)
	if err != nil {
		return nil, err
	}

	s := polochon.NewSubtitleFromVideo(video, lang)
	s.Data = data

	return s, nil
}

// movieParams builds search parameters for a movie.
func movieParams(m *polochon.Movie, langCode string) url.Values {
	params := url.Values{}
	params.Set("keywords", m.Title)
	params.Set("language", langCode)
	params.Set("movie_type", "movie")
	if m.Year > 0 {
		params.Set("year", strconv.Itoa(m.Year))
	}
	return params
}

// showEpisodeParams builds search parameters for a show episode.
func showEpisodeParams(e *polochon.ShowEpisode, langCode string) url.Values {
	params := url.Values{}
	params.Set("keywords", e.ShowTitle)
	params.Set("language", langCode)
	params.Set("movie_type", "tv-series")
	params.Set("seasons", strconv.Itoa(e.Season))
	params.Set("episodes", strconv.Itoa(e.Episode))
	return params
}

// bestSubtitle picks the subtitle whose custom_releases entry is closest to
// the release name using Levenshtein distance.
func bestSubtitle(subs []*subtitle, release string) (*subtitle, error) {
	var selected *subtitle
	minScore := -1

	for _, sub := range subs {
		if sub.PublishID == "" {
			continue
		}
		for _, rel := range sub.CustomReleases {
			score := levenshtein.ComputeDistance(release, rel)
			if selected == nil || score < minScore {
				selected = sub
				minScore = score
			}
		}
		// If a subtitle has no custom_releases, still consider it with a
		// worst-case score so it can be a fallback.
		if len(sub.CustomReleases) == 0 {
			if selected == nil {
				selected = sub
			}
		}
	}

	if selected == nil {
		return nil, polochon.ErrNoSubtitleFound
	}

	return selected, nil
}
