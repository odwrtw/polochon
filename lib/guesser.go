package polochon

import (
	"github.com/sirupsen/logrus"
)

// Guesser is an interface which allows to get inforamtions to create a video
// from a file
type Guesser interface {
	Module
	Guess(File, MovieConfig, ShowConfig, *logrus.Entry) (Video, error)
}
