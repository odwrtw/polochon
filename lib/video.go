package polochon

import (
	"fmt"

	"github.com/Sirupsen/logrus"
)

// Quality represents the qualities of a video
type Quality string

// Possible qualities
const (
	Quality480p  Quality = "480p"
	Quality720p          = "720p"
	Quality1080p         = "1080p"
	Quality3D            = "3D"
)

// StringToQuality returns a Quality from a string
func StringToQuality(q string) (*Quality, error) {
	for _, quality := range []Quality{
		Quality480p,
		Quality720p,
		Quality1080p,
		Quality3D,
	} {
		if string(quality) == q {
			return &quality, nil
		}
	}
	return nil, fmt.Errorf("invalid quality")
}

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
	Subtitlable
	GetDetails(*logrus.Entry) error
	GetTorrents(*logrus.Entry) error
	SetFile(f *File)
	GetFile() *File
}
