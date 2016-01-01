package kickass

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/guessit"
	"github.com/odwrtw/polochon/lib"
)

// Custom errors
var (
	ErrMissingShowEpisodeInfos = errors.New("kickass: missing show informations")
)

// ShowEpisodeSearcher implements the Searcher interface
type ShowEpisodeSearcher struct {
	*polochon.ShowEpisode
	kickassUsers []string
}

// NewShowEpisodeSearcher returns a new ShowEpisodeSearcher
func NewShowEpisodeSearcher(se *polochon.ShowEpisode, users []string) *ShowEpisodeSearcher {
	return &ShowEpisodeSearcher{
		ShowEpisode:  se,
		kickassUsers: users,
	}
}

func (se *ShowEpisodeSearcher) validate() error {
	if se.ShowTitle == "" {
		return fmt.Errorf("kickass: missing show title")
	}

	if se.Season == 0 {
		return fmt.Errorf("kickass: missing episode season")
	}

	if se.Episode == 0 {
		return fmt.Errorf("kickass: missing episode number")
	}

	return nil
}

func (se *ShowEpisodeSearcher) searchStr() string {
	return fmt.Sprintf("%s S%02dE%02d", se.ShowTitle, se.Season, se.Episode)
}

func (se *ShowEpisodeSearcher) category() Category {
	return ShowsCategory
}

func (se *ShowEpisodeSearcher) users() []string {
	return se.kickassUsers
}

func (se *ShowEpisodeSearcher) isValidGuess(guess *guessit.Response, log *logrus.Entry) bool {
	// Check if the years match
	if guess.Season == 0 || guess.Episode == 0 {
		log.Infof("no season or episode guessed from title")
		return false
	}

	// Check if the years match
	if guess.Season != se.Season || guess.Episode != se.Episode {
		log.Infof("episode do not match: guessed S%02dE%02d, wanted S%02dE%02d", guess.Season, guess.Episode, se.Season, se.Episode)
		return false
	}

	return true
}
