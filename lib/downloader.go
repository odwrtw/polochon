package polochon

import (
	"errors"

	"github.com/sirupsen/logrus"
)

var (
	// ErrDuplicateTorrent returned when the torrent is already added
	ErrDuplicateTorrent = errors.New("Torrent already added")
)

// Downloader represent a interface for any downloader
type Downloader interface {
	Module
	Download(string, *logrus.Entry) error
	Remove(Downloadable) error
	List() ([]Downloadable, error)
}

// Downloadable is an interface for anything to be downloaded
type Downloadable interface {
	Infos() *DownloadableInfos
}

// DownloadableInfos represent infos about a Downloadable object
type DownloadableInfos struct {
	ID              string                 `json:"id"`
	Ratio           float32                `json:"ratio"`
	IsFinished      bool                   `json:"is_finished"`
	FilePaths       []string               `json:"file_paths"`
	Name            string                 `json:"name"`
	DownloadRate    int                    `json:"download_rate"`
	UploadRate      int                    `json:"upload_rate"`
	TotalSize       int                    `json:"total_size"`
	DownloadedSize  int                    `json:"downloaded_size"`
	UploadedSize    int                    `json:"uploaded_size"`
	PercentDone     float32                `json:"percent_done"`
	AdditionalInfos map[string]interface{} `json:"additional_infos"`
}
