package polochon

import "github.com/Sirupsen/logrus"

// Explorer is the interface explore new videos from different sources
type Explorer interface {
	Module
	AvailableMovieOptions() []string
	GetMovieList(option string, log *logrus.Entry) ([]*Movie, error)
	AvailableShowOptions() []string
	GetShowList(option string, log *logrus.Entry) ([]*Show, error)
}
