package polochon

// MovieConfig represents the configuration for a movie
type MovieConfig struct {
	Torrenters []Torrenter
	Detailers  []Detailer
	Subtitlers []Subtitler
	Explorers  []Explorer
	Searchers  []Searcher
}

// Movie represents a movie
type Movie struct {
	MovieConfig `json:"-"`
	File
	ImdbID        string    `json:"imdb_id"`
	OriginalTitle string    `json:"original_title"`
	Plot          string    `json:"plot"`
	Rating        float32   `json:"rating"`
	Runtime       int       `json:"runtime"`
	SortTitle     string    `json:"sort_title"`
	Tagline       string    `json:"tag_line"`
	Thumb         string    `json:"thumb"`
	Fanart        string    `json:"fanart"`
	Title         string    `json:"title"`
	TmdbID        int       `json:"tmdb_id"`
	Votes         int       `json:"votes"`
	Year          int       `json:"year"`
	Genres        []string  `json:"genres"`
	Torrents      []Torrent `json:"torrents"`
}

// NewMovie returns a new movie
func NewMovie(movieConfig MovieConfig) *Movie {
	return &Movie{
		MovieConfig: movieConfig,
	}
}

// NewMovieFromFile returns a new movie from a file
func NewMovieFromFile(movieConfig MovieConfig, file File) *Movie {
	return &Movie{
		MovieConfig: movieConfig,
		File:        file,
	}
}

// GetTorrenters implements the Torrentable interface
func (m *MovieConfig) GetTorrenters() []Torrenter {
	return m.Torrenters
}

// GetSubtitlers implements the Subtitlable interface
func (m *MovieConfig) GetSubtitlers() []Subtitler {
	return m.Subtitlers
}

// GetDetailers implements the Detailable interface
func (m *MovieConfig) GetDetailers() []Detailer {
	return m.Detailers
}

// GetExplorers implements the Explorer interface
func (m *MovieConfig) GetExplorers() []Explorer {
	return m.Explorers
}

// GetSearchers implements the Searcher interface
func (m *MovieConfig) GetSearchers() []Searcher {
	return m.Searchers
}

// SetFile implements the video interface
func (m *Movie) SetFile(f *File) {
	m.File = *f
}

// GetFile implements the video interface
func (m *Movie) GetFile() *File {
	return &m.File
}
