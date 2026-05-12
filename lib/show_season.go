package polochon

// ShowSeason represents a show season
type ShowSeason struct {
	ShowConfig `json:"-"`
	ShowImdbID string `json:"show_imdb_id"`
	Season     int    `json:"season"`
	BannerURL  string `json:"-"`
	FanartURL  string `json:"-"`
	PosterURL  string `json:"-"`
	// Episodes holds the indexed episodes for this season keyed by episode number.
	Episodes map[int]*ShowEpisode `json:"episodes"`
}

// NewShowSeason returns a new show season
func NewShowSeason(conf ShowConfig) *ShowSeason {
	return &ShowSeason{
		ShowConfig: conf,
	}
}
