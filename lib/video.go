package polochon

import (
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
)

// Regexp used for slugs by Movie and ShowEpisode objects
var (
	invalidSlugPattern = regexp.MustCompile(`[^a-z0-9 _-]`)
	whiteSpacePattern  = regexp.MustCompile(`\s+`)
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
	Slug() string
	Notify(*logrus.Entry) error
	SetFile(f *File)
	GetFile() *File
}

func slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}
