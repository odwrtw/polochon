package pushover

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/gregdel/pushover"
	"gitlab.quimbo.fr/odwrtw/polochon/lib"
)

// Pushover errors
var (
	ErrMissingKey       = errors.New("pushover: missing key")
	ErrMissingRecipient = errors.New("pushover: missing recipient")
	ErrInvalidArgument  = errors.New("pushover: invalid argument type")
)

// Register a new notifier
func init() {
	polochon.RegisterNotifier("pushover", New)
}

// Pushover stores the notification configs
type Pushover struct {
	app       *pushover.Pushover
	recipient *pushover.Recipient
}

// New returns a new Pushover
func New(params map[string]string, log *logrus.Entry) (polochon.Notifier, error) {
	// Check params
	key, ok := params["key"]
	if !ok {
		return nil, ErrMissingKey
	}

	recipient, ok := params["recipient"]
	if !ok {
		return nil, ErrMissingRecipient
	}

	return &Pushover{
		app:       pushover.New(key),
		recipient: pushover.NewRecipient(recipient),
	}, nil
}

// Notify sends a notification to the recipient
func (p *Pushover) Notify(i interface{}) error {
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
