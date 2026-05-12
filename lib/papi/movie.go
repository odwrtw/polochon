package papi

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
)

// Movie struct returned by papi
type Movie struct {
	*polochon.Movie

	Subtitles []*Subtitle `json:"subtitles"`
}

// SidecarFiles returns the sidecar files (fanart, thumb, nfo) as downloadable papi.File objects.
func (m *Movie) SidecarFiles() []*File {
	return []*File{
		NewFile(m.FanartFile, m),
		NewFile(m.ThumbFile, m),
		NewFile(m.NFOFile, m),
	}
}

// uri implements the Resource interface
func (m *Movie) uri() (string, error) {
	if m.Movie == nil {
		return "", ErrMissingMovie
	}

	if m.ImdbID == "" {
		return "", ErrMissingMovieID
	}

	return fmt.Sprintf("movies/%s", m.ImdbID), nil
}

// downloadURL implements the Downloadable interface
func (m *Movie) downloadURL() (string, error) {
	uri, err := m.uri()
	if err != nil {
		return "", err
	}

	if m.Path != "" {
		return fmt.Sprintf("%s/download/%s", uri, m.Filename()), nil
	}

	return fmt.Sprintf("%s/download", uri), nil
}

// GetMovies returns all the movies in the polochon library
func (c *Client) GetMovies() (*MovieCollection, error) {
	url := fmt.Sprintf("%s/%s", c.endpoint, "movies")

	index := map[string]*Movie{}
	if err := c.get(url, &index); err != nil {
		return nil, err
	}

	mc := NewMovieCollection()
	for id, m := range index {
		if m.Movie == nil {
			m.Movie = &polochon.Movie{}
		}
		m.ImdbID = id
		// Name holds the video filename from the HTTP response
		if m.Name != "" {
			m.Path = m.Name
		}

		for _, s := range m.Subtitles {
			s.Video = m.Movie
		}

		if err := mc.Add(m); err != nil {
			return nil, err
		}
	}

	return mc, nil
}

// getDetails implements the resource interface
func (m *Movie) getDetails(c *Client) error {
	return c.getMovieDetails(m)
}

// getMovieDetails updates the movie with detailed informations from polochon
func (c *Client) getMovieDetails(movie *Movie) error {
	uri, err := movie.uri()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%s", c.endpoint, uri)
	return c.get(url, movie)
}

// GetMovie returns a movie with de detailed infos from polochon
func (c *Client) GetMovie(id string) (*Movie, error) {
	movie := &Movie{Movie: &polochon.Movie{ImdbID: id}}
	if err := c.getMovieDetails(movie); err != nil {
		return nil, err
	}

	return movie, nil
}
