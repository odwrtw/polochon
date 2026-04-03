package mock

import polochon "github.com/odwrtw/polochon/lib"

// Guess implements the guesser interface
func (mock *Mock) Guess(polochon.File, polochon.MovieConfig,
	polochon.ShowConfig) (polochon.Video, error) {
	return nil, nil
}

// GuessMetadata implements the guesser interface
func (mock *Mock) GuessMetadata(*polochon.File) (*polochon.VideoMetadata, error) {
	return nil, nil
}
