package polochon

import "github.com/Sirupsen/logrus"

// Quality represents the qualities of a video
type Quality string

// Possible qualities
const (
	Quality480p  Quality = "480p"
	Quality720p          = "720p"
	Quality1080p         = "1080p"
	Quality3D            = "3D"
)

// IsAllowed checks if the quality is allowed
func (q *Quality) IsAllowed() bool {
	for _, quality := range []Quality{
		Quality480p,
		Quality720p,
		Quality1080p,
		Quality3D,
	} {
		if *q == quality {
			return true
		}
	}
	return false
}

// Video represents a generic video type
type Video interface {
	GetDetails(*logrus.Entry) error
	GetTorrents(*logrus.Entry) error
	GetSubtitle(*logrus.Entry) error
	SetFile(f *File)
	GetFile() *File
}
