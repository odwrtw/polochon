package polochon

// ShowSeason represents a show season
type ShowSeason struct {
	ShowConfig `json:"-"`
	ShowImdbID string `json:"show_imdb_id"`
	Season     int    `json:"season"`
	Banner     string `json:"-"`
	Fanart     string `json:"-"`
	Poster     string `json:"-"`
}

// NewShowSeason returns a new show season
func NewShowSeason(conf ShowConfig) *ShowSeason {
	return &ShowSeason{
		ShowConfig: conf,
	}
}
