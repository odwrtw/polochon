package tpb

import (
	"fmt"
	"strings"

	"github.com/odwrtw/guessit"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

type showSearcher struct {
	Episode *polochon.ShowEpisode
	Users   []string
}

func (sS *showSearcher) key() string {
	return fmt.Sprintf(
		"%s S%02dE%02d",
		sS.Episode.ShowTitle,
		sS.Episode.Season,
		sS.Episode.Episode,
	)
}

func (sS *showSearcher) users() []string {
	return sS.Users
}

func (sS *showSearcher) setTorrents(torrents []*polochon.Torrent) {
	for _, t := range torrents {
		t.ImdbID = sS.Episode.ShowImdbID
		t.Season = sS.Episode.Season
		t.Episode = sS.Episode.Episode
		t.Type = polochon.TypeEpisode
	}
	sS.Episode.Torrents = torrents
}

func (sS *showSearcher) defaultQuality() string {
	return string(polochon.Quality480p)
}

func (sS *showSearcher) imdbID() string {
	return sS.Episode.ShowImdbID
}

func (sS *showSearcher) isValidGuess(guess *guessit.Response, log *logrus.Entry) bool {
	// Check the video type
	if guess.Type != "episode" {
		log.Debugf("tpb: is not an episode but a %s", guess.Type)
		return false
	}

	if !strings.EqualFold(guess.Title, sS.Episode.ShowTitle) {
		log.Debugf("skipping bad show title %s != %s", guess.Title, sS.Episode.ShowTitle)
		return false
	}
	// Check if the data matches the episode
	if guess.Season != sS.Episode.Season || guess.Episode != sS.Episode.Episode {
		log.Debugf("skipping bad show episode/season S%dE%d != S%dE%d", guess.Season, guess.Episode, sS.Episode.Season, sS.Episode.Episode)
		return false
	}
	return true
}
