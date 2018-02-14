package pushover

import (
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"net/http"

	"gopkg.in/yaml.v2"

	"github.com/gregdel/pushover"
	"github.com/nfnt/resize"
	"github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Pushover errors
var (
	ErrMissingArgument = errors.New("pushover: missing argument")
	ErrInvalidArgument = errors.New("pushover: invalid argument type")
)

// Module constants
const (
	moduleName = "pushover"
)

// Register a new notifier
func init() {
	polochon.RegisterNotifier(moduleName, NewFromRawYaml)
}

// Params represents the module params
type Params struct {
	Key       string `yaml:"key"`
	Recipient string `yaml:"recipient"`
}

// IsValid checks if the given params are valid
func (p *Params) IsValid() bool {
	if p.Key == "" || p.Recipient == "" {
		return false
	}
	return true
}

// Pushover stores the notification configs
type Pushover struct {
	app       *pushover.Pushover
	recipient *pushover.Recipient
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Notifier, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New returns a new Pushover
func New(params *Params) (polochon.Notifier, error) {
	if !params.IsValid() {
		return nil, ErrMissingArgument
	}

	return &Pushover{
		app:       pushover.New(params.Key),
		recipient: pushover.NewRecipient(params.Recipient),
	}, nil
}

// Name implements the Module interface
func (p *Pushover) Name() string {
	return moduleName
}

// Notify sends a notification to the recipient
func (p *Pushover) Notify(i interface{}, log *logrus.Entry) error {
	switch v := i.(type) {
	case *polochon.ShowEpisode:
		return p.notifyShowEpisode(v)
	case *polochon.Movie:
		return p.notifyMovie(v)
	default:
		return ErrInvalidArgument
	}
}

// Notify sends a movie notification
func (p *Pushover) notifyMovie(movie *polochon.Movie) error {
	message := &pushover.Message{
		Title:    fmt.Sprintf("Canapé (Movie)"),
		Message:  movie.Title,
		URL:      fmt.Sprintf("imdb:///title/%s/", movie.ImdbID),
		URLTitle: "Open on imdb",
	}

	if movie.Thumb != "" {
		resp, err := http.Get(movie.Thumb)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		img, err := jpeg.Decode(resp.Body)
		if err != nil {
			return err
		}

		thumb := resize.Resize(150, 0, img, resize.Lanczos3)

		pr, pw := io.Pipe()
		go func() {
			if err := jpeg.Encode(pw, thumb, nil); err != nil {
				pw.CloseWithError(err)
				return
			}
			pw.Close()
		}()

		if err := message.AddAttachment(pr); err != nil {
			return err
		}
	}

	_, err := p.app.SendMessage(message, p.recipient)
	if err != nil {
		return err
	}

	return nil
}

// Notify sends a show episode notification
func (p *Pushover) notifyShowEpisode(show *polochon.ShowEpisode) error {
	message := &pushover.Message{
		Title:    fmt.Sprintf("Canapé (Show)"),
		Message:  fmt.Sprintf("%s - S%02dE%02d", show.ShowTitle, show.Season, show.Episode),
		URL:      fmt.Sprintf("imdb:///title/%s/", show.ShowImdbID),
		URLTitle: "Open on imdb",
	}

	_, err := p.app.SendMessage(message, p.recipient)
	if err != nil {
		return err
	}

	return nil
}
