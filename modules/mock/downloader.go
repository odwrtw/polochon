package mock

import (
	"fmt"
	"math/rand"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Download implements the downloader interface
func (mock *Mock) Download(string, *polochon.DownloadableMetadata, *logrus.Entry) error {
	return nil
}

// Torrent represents a Mock torrent
type Torrent struct{}

// Infos implements the downloader interface
func (t Torrent) Infos() *polochon.DownloadableInfos {
	id := fmt.Sprintf("%d", rand.Intn(1000))
	return &polochon.DownloadableInfos{
		ID:             id,
		UploadRate:     11,
		DownloadRate:   22,
		DownloadedSize: 290,
		UploadedSize:   11,
		FilePaths:      []string{"/tmp/yo", "/tmp/coucou"},
		IsFinished:     false,
		Name:           fmt.Sprintf("Torrent %s", id),
		PercentDone:    float32(rand.Intn(100)),
		Ratio:          11,
		TotalSize:      500,
		Metadata: &polochon.DownloadableMetadata{
			ImdbID:  id,
			Quality: "720p",
			Type:    "movie",
		},
	}
}

// List implements the downloader interface
func (mock *Mock) List() ([]polochon.Downloadable, error) {
	return []polochon.Downloadable{Torrent{}, Torrent{}}, nil
}

// Remove implements the downloader interface
func (mock *Mock) Remove(polochon.Downloadable) error {
	return nil
}
