package mock

import (
	"fmt"
	"math/rand"

	polochon "github.com/odwrtw/polochon/lib"
)

// Download implements the downloader interface
func (mock *Mock) Download(*polochon.Torrent) error {
	return nil
}

// List implements the downloader interface
func (mock *Mock) List() ([]*polochon.Torrent, error) {
	id1 := fmt.Sprintf("%d", rand.Intn(1000))
	id2 := fmt.Sprintf("%d", rand.Intn(1000))
	return []*polochon.Torrent{
		{
			ID:             id1,
			UploadRate:     11,
			DownloadRate:   22,
			DownloadedSize: 290,
			UploadedSize:   11,
			FilePaths:      []string{"/this.is.a.movie.mp4"},
			IsFinished:     false,
			Name:           fmt.Sprintf("Torrent %s", id1),
			PercentDone:    float32(rand.Intn(100)),
			Ratio:          11,
			TotalSize:      500,
			Metadata: &polochon.TorrentMetadata{
				ImdbID:  id1,
				Quality: "720p",
				Type:    "movie",
			},
		},
		{
			ID:             id2,
			UploadRate:     100,
			DownloadRate:   300,
			DownloadedSize: 290,
			UploadedSize:   110,
			FilePaths: []string{
				"/this.is.a.show.s01e03.mp4",
				"bullshit.txt",
			},
			IsFinished:  false,
			Name:        fmt.Sprintf("Torrent %s", id2),
			PercentDone: float32(rand.Intn(100)),
			Ratio:       11,
			TotalSize:   500,
			Metadata: &polochon.TorrentMetadata{
				ImdbID:  id2,
				Quality: "720p",
				Type:    "episode",
				Season:  1,
				Episode: 3,
			},
		}}, nil
}

// Remove implements the downloader interface
func (mock *Mock) Remove(*polochon.Torrent) error {
	return nil
}
