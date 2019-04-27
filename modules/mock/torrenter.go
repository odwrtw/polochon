package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// GetTorrents implements the torrenter interface
func (mock *Mock) GetTorrents(interface{}, *logrus.Entry) error {
	return nil
}

// SearchTorrents implements the torrenter interface
func (mock *Mock) SearchTorrents(string) ([]*polochon.Torrent, error) {
	return nil, nil
}
