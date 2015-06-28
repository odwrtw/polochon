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

// Torrent represents a torrent file
type Torrent struct {
	Quality Quality
	URL     string
}

// VideoConfig represents the configuration for video object
type VideoConfig struct {
	GuesserName               string   `yaml:"guesser"`
	NotifierNames             []string `yaml:"notifiers"`
	ExcludeFileContaining     []string `yaml:"exclude_file_containing"`
	VideoExtentions           []string `yaml:"allowed_file_extensions"`
	AllowedExtentionsToDelete []string `yaml:"allowed_file_extensions_to_delete"`
	// TODO: delete it
	Guesser   Guesser    `yaml:"-"`
	Notifiers []Notifier `yaml:"-"`
}

// Video represents a generic video type
type Video interface {
	GetDetails() error
	GetTorrents() error
	GetSubtitle() error
	Notify() error
	Type() VideoType
	Store() error
	SetFile(f *File)
	SetConfig(c *Config)
}
