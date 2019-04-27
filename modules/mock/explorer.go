package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// AvailableMovieOptions implements the explorer interface
func (mock *Mock) AvailableMovieOptions() []string {
	return nil
}

// AvailableShowOptions implements the explorer interface
func (mock *Mock) AvailableShowOptions() []string {
	return nil
}

// GetMovieList implements the explorer interface
func (mock *Mock) GetMovieList(option string, log *logrus.Entry) ([]*polochon.Movie, error) {
	return nil, nil
}

// GetShowList implements the explorer interface
func (mock *Mock) GetShowList(option string, log *logrus.Entry) ([]*polochon.Show, error) {
	return nil, nil
}
