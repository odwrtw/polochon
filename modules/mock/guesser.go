package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Guess implements the guesser interface
func (mock *Mock) Guess(polochon.File, polochon.MovieConfig,
	polochon.ShowConfig, *logrus.Entry) (polochon.Video, error) {
	return nil, nil
}

// GuessMetadata implements the guesser interface
func (mock *Mock) GuessMetadata(*polochon.File, *logrus.Entry) (*polochon.VideoMetadata, error) {
	return nil, nil
}
