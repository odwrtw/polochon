package polochon

import (
	"errors"

	"github.com/sirupsen/logrus"
)

// Guess errors
var (
	ErrGuessingVideo    = errors.New("polochon: failed to guess video")
	ErrGuessingMetadata = errors.New("polochon: failed to guess metadata")
)

// Guesser is an interface which allows to get informations to create a video
// from a file
type Guesser interface {
	Module
	Guess(File, MovieConfig, ShowConfig, *logrus.Entry) (Video, error)
	GuessMetadata(*File, *logrus.Entry) (*VideoMetadata, error)
}
