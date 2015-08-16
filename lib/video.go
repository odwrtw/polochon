package polochon

import (
	"errors"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
)

// Video errors
var (
	ErrInvalidVideoType = errors.New("polochon: invalid video type")
)

// Regexp used for slugs by Movie and ShowEpisode objects
var (
	invalidSlugPattern = regexp.MustCompile(`[^a-z0-9 _-]`)
	whiteSpacePattern  = regexp.MustCompile(`\s+`)
)

// VideoType represent the types of video
type VideoType string

// Possible types of video
const (
	MovieType       VideoType = "movie"
	ShowEpisodeType           = "episode"
	ShowType                  = "show"
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

// Torrent represents a torrent file
type Torrent struct {
	Quality Quality
	URL     string
}

// Video represents a generic video type
type Video interface {
	GetDetails() error
	GetTorrents() error
	GetSubtitle() error
	Slug() string
	Notify() error
	Type() VideoType
	Store() error
	SetFile(f *File)
	GetFile() *File
	Delete() error
	SetConfig(c *VideoConfig, log *logrus.Logger)
}

func slug(text string) string {
	separator := "-"
	text = strings.ToLower(text)
	text = invalidSlugPattern.ReplaceAllString(text, "")
	text = whiteSpacePattern.ReplaceAllString(text, separator)
	text = strings.Trim(text, separator)
	return text
}
