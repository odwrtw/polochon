package tpb

import (
	"fmt"
	"strings"

	"github.com/odwrtw/guessit"
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/tpb"
	"github.com/sirupsen/logrus"
)

type movieSearcher struct {
	Movie *polochon.Movie
	Users []string
}

func (mS *movieSearcher) key() string {
	return fmt.Sprintf("%s", mS.Movie.Title)
}

func (mS *movieSearcher) users() []string {
	return mS.Users
}

func (mS *movieSearcher) setTorrents(torrents []polochon.Torrent, log *logrus.Entry) {
	mS.Movie.Torrents = torrents
}

func (mS *movieSearcher) category() tpb.TorrentCategory {
	return tpb.Video
}

func (mS *movieSearcher) videoType() string {
	return "movie"
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
	if guess.Type != mS.videoType() {
		log.Debugf("tpb: is not a %s but a %s", mS.videoType(), guess.Type)
		return false
	}
	return true
}
