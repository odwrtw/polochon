package polochon

import (
	"github.com/sirupsen/logrus"
)

// Guesser is an interface which allows to get informations to create a video
// from a file
type Guesser interface {
	Module
	Guess(File, MovieConfig, ShowConfig, *logrus.Entry) (Video, error)
	GuessMetadata(*File) (*VideoMetadata, error)
}
