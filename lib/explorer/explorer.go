package explorer

import (
	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

// Option represents the query type of explorer
type Option int

// Available explorer options
const (
	BySeeds Option = iota
	ByPeers
	ByTitle
	ByYear
	ByRate
	ByDownloadCount
	ByLikeCount
	ByDateAdded
)

// Explorer is the interface explore new videos from different sources
type Explorer interface {
	AvailableMovieOptions() []Option
	GetMovieList(option Option, log *logrus.Entry) ([]*polochon.Movie, error)
	AvailableShowOptions() []Option
	GetShowList(option Option, log *logrus.Entry) ([]*polochon.Show, error)
}
