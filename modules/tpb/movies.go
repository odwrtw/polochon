package tpb

import (
	"strings"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/odwrtw/whatsthis"
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
	for _, t := range torrents {
		t.ImdbID = mS.Movie.ImdbID
		t.Type = polochon.TypeMovie
	}
	mS.Movie.Torrents = torrents
}

func (mS *movieSearcher) defaultQuality() string {
	return string(polochon.Quality720p)
}

func (mS *movieSearcher) imdbID() string {
	return mS.Movie.ImdbID
}

func (mS *movieSearcher) isValidGuess(guess whatsthis.Info) bool {
	if !strings.EqualFold(guess.Title, mS.Movie.Title) {
		return false
	}

	// Check the video type
	if guess.Type != whatsthis.Movie {
		return false
	}

	return true
}
