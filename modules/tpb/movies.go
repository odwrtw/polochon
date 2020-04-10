package tpb

import (
	"strings"

	"github.com/odwrtw/guessit"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

type movieSearcher struct {
	Movie *polochon.Movie
	Users []string
}

func (mS *movieSearcher) key() string {
	return mS.Movie.Title
}

func (mS *movieSearcher) users() []string {
	return mS.Users
}

func (mS *movieSearcher) setTorrents(torrents []*polochon.Torrent) {
	mS.Movie.Torrents = torrents
}

func (mS *movieSearcher) defaultQuality() string {
	return string(polochon.Quality720p)
}

func (mS *movieSearcher) isValidGuess(guess *guessit.Response, log *logrus.Entry) bool {
	if guess.VideoCodec == "h265" {
		log.Debugf("skipping h265 codec")
		return false
	}

	if !strings.EqualFold(guess.Title, mS.Movie.Title) {
		log.Debugf("skipping bad movie title %s != %s", guess.Title, mS.Movie.Title)
		return false
	}

	// Check the video type
	if guess.Type != "movie" {
		log.Debugf("tpb: is not a movie but a %s", guess.Type)
		return false
	}

	return true
}
