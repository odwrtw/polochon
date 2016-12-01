package polochon

import "github.com/Sirupsen/logrus"

// ExplorerOption represents the query type of explorer
type ExplorerOption int

// Available explorer options
const (
	ExploreBySeeds ExplorerOption = iota
	ExploreByPeers
	ExploreByTitle
	ExploreByYear
	ExploreByRate
	ExploreByDownloadCount
	ExploreByLikeCount
	ExploreByDateAdded
)

// Explorer is the interface explore new videos from different sources
type Explorer interface {
	AvailableMovieOptions() []ExplorerOption
	GetMovieList(option ExplorerOption, log *logrus.Entry) ([]*Movie, error)
	AvailableShowOptions() []ExplorerOption
	GetShowList(option ExplorerOption, log *logrus.Entry) ([]*Show, error)
}
