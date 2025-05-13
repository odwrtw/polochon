package subtitlor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/agnivade/levenshtein"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Make sure that the module is a subtitler
var _ polochon.Subtitler = (*Subtitlor)(nil)

func init() {
	polochon.RegisterModule(&Subtitlor{})
}

// Subtitlor holds the Subtitlor module
type Subtitlor struct {
	configured bool
	endpoint   string
	token      string
}

// Module constants
const (
	moduleName = "subtitlor"
)

// Errors
var (
	ErrMissingImdbID = errors.New("subtitlor: missing imdb id")
)

// Init implements the module interface
func (s *Subtitlor) Init(p []byte) error {
	if s.configured {
		return nil
	}

	// Params are the params for webhooks
	type Params struct {
		Endpoint string `yaml:"endpoint"`
		Token    string `yaml:"token"`
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, &params); err != nil {
		return err
	}

	if params.Endpoint == "" || params.Token == "" {
		return fmt.Errorf("subtitlor: endpoint and token are required")
	}

	s.configured = true
	s.endpoint = params.Endpoint
	s.token = params.Token
	return nil
}

// Name implements the Module interface
func (s *Subtitlor) Name() string {
	return moduleName
}

// Status implements the Module interface
func (s *Subtitlor) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusOK, nil
}

// getSubtitle will search and downalod a subtitle
func (s *Subtitlor) getSubtitle(m polochon.Video, url string, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	url, err := s.searchSubtitle(m, url, lang, log)
	if err != nil {
		return nil, err
	}
	// Add a context with a timeout to the request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", s.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subtitlor: HTTP status %d when downloading", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	sub := polochon.NewSubtitleFromVideo(m, lang)
	sub.Data = data
	return sub, nil
}

func (s *Subtitlor) searchSubtitle(m polochon.Video, url string, lang polochon.Language, _ *logrus.Entry) (string, error) {
	// Add a context with a timeout to the request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", s.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", polochon.ErrNoSubtitleFound
	} else if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("subtitlor: HTTP status %d when searching", resp.StatusCode)
	}

	type subtitleResult struct {
		Lang     string `json:"language"`
		Filename string `json:"filename"`
	}

	var minScore = 1000
	var selected *subtitleResult
	var result map[string][]subtitleResult

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("subtitlor: failed to decode response: %w", err)
	}

	videoName := filepath.Base(m.GetFile().PathWithoutExt())
	for l, subs := range result {
		if lang.ShortForm() != l {
			continue
		}

		for _, sub := range subs {
			dist := levenshtein.ComputeDistance(videoName, sub.Filename)
			if dist < minScore {
				selected = &sub
				minScore = dist
			}
		}
	}
	if selected == nil {
		return "", polochon.ErrNoSubtitleFound
	}

	return fmt.Sprintf("%s/%s/%s", url, selected.Lang, selected.Filename), nil
}

// GetSubtitle implements the Subtitler interface
func (s *Subtitlor) GetSubtitle(i any, lang polochon.Language, log *logrus.Entry) (*polochon.Subtitle, error) {
	switch v := i.(type) {
	case *polochon.Movie:
		if v.ImdbID == "" {
			return nil, ErrMissingImdbID
		}
		path := fmt.Sprintf("%s/movies/%s", s.endpoint, v.ImdbID)
		return s.getSubtitle(v, path, lang, log)
	case *polochon.ShowEpisode:
		if v.ShowImdbID == "" {
			return nil, ErrMissingImdbID
		}
		path := fmt.Sprintf("%s/shows/%s/%d/%d", s.endpoint, v.ShowImdbID, v.Season, v.Episode)
		return s.getSubtitle(v, path, lang, log)
	default:
		return nil, fmt.Errorf("subtitlor: invalid type")
	}
}
