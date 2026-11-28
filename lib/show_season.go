package polochon

// ShowSeason represents a show season
type ShowSeason struct {
	ShowConfig `json:"-"`
	ShowImdbID string     `json:"show_imdb_id"`
	ShowTitle  string     `json:"-"`
	Season     int        `json:"season"`
	BannerURL  string     `json:"-"`
	FanartURL  string     `json:"-"`
	PosterURL  string     `json:"-"`
	Torrents   []*Torrent `json:"-"`
}

// NewShowSeason returns a new show season
func NewShowSeason(conf ShowConfig) *ShowSeason {
	return &ShowSeason{
		ShowConfig: conf,
	}
}
