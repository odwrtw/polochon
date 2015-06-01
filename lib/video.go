package polochon

import "errors"

// Video errors
var (
	ErrInvalidVideoType = errors.New("polochon: invalid video type")
	ErrInvalidQuality   = errors.New("polochon: invalid quality")
)

// VideoType represent the types of video
type VideoType string

// Possible types of video
const (
	MovieType       VideoType = "movie"
	ShowEpisodeType           = "episode"
	ShowType                  = "show"
)

// StringToVideoType converts a string to a VideoType
var stringToVideoType = map[string]VideoType{
	"movie":   MovieType,
	"episode": ShowEpisodeType,
	"show":    ShowType,
}

// Video type returns the video type from a string
func videoType(s string) (VideoType, error) {
	t, ok := stringToVideoType[s]
	if !ok {
		return "", ErrInvalidVideoType
	}

	return t, nil
}

// Quality represents the qualities of a video
type Quality string

// Possible qualities
const (
	Quality480p  Quality = "480p"
	Quality720p          = "720p"
	Quality1080p         = "1080p"
	Quality3D            = "3D"
)

var stringToQuality = map[string]Quality{
	"480p":  Quality480p,
	"720p":  Quality720p,
	"1080p": Quality1080p,
	"3D":    Quality3D,
}

// GetQuality helps find the quality from a string
func GetQuality(s string) (Quality, error) {
	q, ok := stringToQuality[s]
	if !ok {
		return "", ErrInvalidQuality
	}

	return q, nil
}

// Default qualities in order
var defaultQualities = []Quality{
	Quality3D,
	Quality1080p,
	Quality720p,
	Quality480p,
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
	Notify() error
	Type() VideoType
	Store() error
	SetFile(f *File)
	SetConfig(c *Config)
}
