package mock

import (
	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Download implements the downloader interface
func (mock *Mock) Download(string, *logrus.Entry) error {
	return nil
}

// List implements the downloader interface
func (mock *Mock) List() ([]polochon.Downloadable, error) {
	return nil, nil
}

// Remove implements the downloader interface
func (mock *Mock) Remove(polochon.Downloadable) error {
	return nil
}
