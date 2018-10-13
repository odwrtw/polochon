package webhook

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"

	"gopkg.in/yaml.v2"

	"github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// WebHook errors
var (
	ErrMissingArgument = errors.New("webhook: missing argument")
	ErrInvalidArgument = errors.New("webhook: invalid argument type")
)

// Module constants
const (
	moduleName = "webhook"
)

// Register a new notifier
func init() {
	polochon.RegisterNotifier(moduleName, NewFromRawYaml)
}

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
	httpClient *http.Client
	hooks      []*Hook
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Notifier, error) {
	params := Params{}
	if err := yaml.Unmarshal(p, &params); err != nil {
		return nil, err
	}

	for _, h := range params.Hooks {
		url, err := template.New("url").Parse(h.URL)
		if err != nil {
			return nil, err
		}
		h.URLTemplate = url
	}

	return &WebHook{
		hooks:      params.Hooks,
		httpClient: http.DefaultClient,
	}, nil
}

// Name implements the Module interface
func (w *WebHook) Name() string {
	return moduleName
}

// Notify sends a notification to the recipient
func (w *WebHook) Notify(i interface{}, log *logrus.Entry) error {
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
		err := w.notify(h, video, videoType)
		if err != nil {
			log.Warnf(err.Error())
		}
	}

	return nil
}

func (w *WebHook) notify(hook *Hook, video polochon.Video, videoType string) error {
	var URL bytes.Buffer
	err := hook.URLTemplate.Execute(&URL, video)
	if err != nil {
		return err
	}

	b := new(bytes.Buffer)
	json.NewEncoder(b).Encode(struct {
		Type string      `json:"type"`
		Data interface{} `json:"data"`
	}{
		Type: videoType,
		Data: video,
	})
	req, err := http.NewRequest("POST", URL.String(), b)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	// Send request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// If status > 400, something's wrong
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("%s call failed with error %d", URL.String(), resp.StatusCode)
	}
	return nil
}
