package polochon

// ShowConfig represents the configuration for a show and its show episodes
type ShowConfig struct {
	Calendar   Calendar
	Detailers  []Detailer
	Subtitlers []Subtitler
	Torrenters []Torrenter
	Explorers  []Explorer
	Searchers  []Searcher
}

// ShowEpisode represents a tvshow episode
type ShowEpisode struct {
	ShowConfig `json:"-"`
	BaseVideo

	Title         string  `json:"title"`
	ShowTitle     string  `json:"show_title"`
	Season        int     `json:"season"`
	Episode       int     `json:"episode"`
	TvdbID        int     `json:"tvdb_id"`
	Aired         string  `json:"aired"`
	Plot          string  `json:"plot"`
	Runtime       int     `json:"runtime"`
	Thumb         string  `json:"thumb"`
	Rating        float32 `json:"rating"`
	ShowImdbID    string  `json:"show_imdb_id"`
	ShowTvdbID    int     `json:"show_tvdb_id"`
	EpisodeImdbID string  `json:"imdb_id"`
	Show          *Show   `json:"-"`
}

// NewShowEpisode returns a new show episode
func NewShowEpisode(showConf ShowConfig) *ShowEpisode {
	return &ShowEpisode{
		ShowConfig: showConf,
	}
}

// NewShowEpisodeFromFile returns a new show episode from a file
func NewShowEpisodeFromFile(showConf ShowConfig, file File) *ShowEpisode {
	se := &ShowEpisode{
		ShowConfig: showConf,
	}

	se.File = file
	return se
}

// GetTorrenters implements the Torrentable interface
func (s *ShowConfig) GetTorrenters() []Torrenter {
	return s.Torrenters
}

// GetDetailers implements the Detailable interface
func (s *ShowConfig) GetDetailers() []Detailer {
	return s.Detailers
}

// GetSubtitlers implements the Subtitlable interface
func (s *ShowConfig) GetSubtitlers() []Subtitler {
	return s.Subtitlers
}

// GetExplorers implements the Explorer interface
func (s *ShowConfig) GetExplorers() []Explorer {
	return s.Explorers
}

// GetSearchers implements the Searcher interface
func (s *ShowConfig) GetSearchers() []Searcher {
	return s.Searchers
}
