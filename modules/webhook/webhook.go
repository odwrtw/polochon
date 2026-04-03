package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/goccy/go-yaml"

	polochon "github.com/odwrtw/polochon/lib"
)

// Make sure that the module is a notifier
var _ polochon.Notifier = (*WebHook)(nil)

func init() {
	polochon.RegisterModule(&WebHook{})
}

// WebHook errors
var (
	ErrInvalidArgument = errors.New("webhook: invalid argument type")
)

// Module constants
const (
	moduleName = "webhook"
)

// Params are the params for webhooks
type Params struct {
	Hooks []*Hook `yaml:"hooks"`
}

// Hook represents a Hook
type Hook struct {
	URLTemplate *template.Template `yaml:"-"`
	URL         string             `yaml:"url"`
}

// WebHook stores the webhook configs
type WebHook struct {
	log        *slog.Logger
	httpClient *http.Client
	hooks      []*Hook
	configured bool
}

// Init implements the module interface
func (w *WebHook) Init(p []byte, log *slog.Logger) error {
	w.log = log.With("module", moduleName)
	if w.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, &params); err != nil {
		return err
	}

	return w.InitWithParams(params)
}

// InitWithParams configures the module
func (w *WebHook) InitWithParams(params *Params) error {
	for _, h := range params.Hooks {
		url, err := template.New("url").Parse(h.URL)
		if err != nil {
			return err
		}
		h.URLTemplate = url
	}

	w.hooks = params.Hooks
	w.httpClient = http.DefaultClient
	w.configured = true

	return nil
}

// Name implements the Module interface
func (w *WebHook) Name() string {
	return moduleName
}

// Status implements the Module interface
func (w *WebHook) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// Notify sends a notification to the recipient
func (w *WebHook) Notify(ctx context.Context, i any) error {
	var videoType string
	var video polochon.Video

	switch v := i.(type) {
	case *polochon.ShowEpisode:
		videoType = "episode"
		video = v
	case *polochon.Movie:
		videoType = "movie"
		video = v
	default:
		return ErrInvalidArgument
	}

	for _, h := range w.hooks {
		err := w.notify(ctx, h, video, videoType)
		if err != nil {
			w.log.Warn(err.Error())
		}
	}

	return nil
}

func (w *WebHook) notify(ctx context.Context, hook *Hook, video polochon.Video, videoType string) error {
	var URL bytes.Buffer
	err := hook.URLTemplate.Execute(&URL, video)
	if err != nil {
		return err
	}

	b := new(bytes.Buffer)
	_ = json.NewEncoder(b).Encode(struct {
		Type string `json:"type"`
		Data any    `json:"data"`
	}{
		Type: videoType,
		Data: video,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", URL.String(), b)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	// Add a context with a timeout to the request
	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Send request
	resp, err := w.httpClient.Do(req.WithContext(timeoutCtx))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	// If status > 400, something's wrong
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s call failed with error %d", URL.String(), resp.StatusCode)
	}

	return nil
}
