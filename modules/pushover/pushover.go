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
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a notifier
var _ polochon.Notifier = (*Pushover)(nil)

// Register a new notifier
func init() {
	polochon.RegisterModule(&Pushover{})
}

// Pushover errors
var (
	ErrMissingArgument = errors.New("pushover: missing argument")
	ErrInvalidArgument = errors.New("pushover: invalid argument type")
)

// Width of the pushover image file, the height will be calculated to keep the
// original aspect ratio of the image
const imageWidth = 720

// Module constants
const (
	moduleName = "pushover"
)

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
	app        *pushover.Pushover
	recipient  *pushover.Recipient
	configured bool
}

// Init implements the module interface
func (p *Pushover) Init(data []byte) error {
	if p.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(data, params); err != nil {
		return err
	}

	return p.InitWithParams(params)
}

// InitWithParams configures the module
func (p *Pushover) InitWithParams(params *Params) error {
	if !params.IsValid() {
		return ErrMissingArgument
	}

	p.app = pushover.New(params.Key)
	p.recipient = pushover.NewRecipient(params.Recipient)
	p.configured = true

	return nil
}

// Name implements the Module interface
func (p *Pushover) Name() string {
	return moduleName
}

// Status implements the Module interface
func (p *Pushover) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
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
		Title:    "Canapé (Movie)",
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

		thumb := resize.Resize(imageWidth, 0, img, resize.Lanczos3)

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
		Title:    "Canapé (Show)",
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
