package polochon

import "errors"

// Guess errors
var (
	ErrGuessingVideo    = errors.New("polochon: failed to guess video")
	ErrGuessingMetadata = errors.New("polochon: failed to guess metadata")
)

// Guesser is an interface which allows to get informations to create a video
// from a file
type Guesser interface {
	Module
	Guess(File, MovieConfig, ShowConfig) (Video, error)
	GuessMetadata(*File) (*VideoMetadata, error)
}
