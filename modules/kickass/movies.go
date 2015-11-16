package kickass

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/guessit"
	"github.com/odwrtw/polochon/lib"
)

// MovieSearcher implements the searcher interface
type MovieSearcher struct {
	*polochon.Movie
	kickassUsers []string
}

// NewMovieSearcher returns a new MovieSearcher
func NewMovieSearcher(m *polochon.Movie, users []string) *MovieSearcher {
	return &MovieSearcher{
		Movie:        m,
		kickassUsers: users,
	}
}

func (ms *MovieSearcher) validate() error {
	if ms.Title == "" {
		return fmt.Errorf("kickass: missing movie title")
	}

	return nil
}

func (ms *MovieSearcher) searchStr() string {
	return ms.Title
}

func (ms *MovieSearcher) category() string {
	return "movies"
}

func (ms *MovieSearcher) users() []string {
	return ms.kickassUsers
}

func (ms *MovieSearcher) isValidGuess(guess *guessit.Response, log *logrus.Entry) bool {
	// Check if the years match
	if ms.Year != 0 && guess.Year != 0 && guess.Year != ms.Year {
		log.Infof("invalid year: guessed (%d) wanted (%d)", guess.Year, ms.Year)
		return false
	}

	return true
}
