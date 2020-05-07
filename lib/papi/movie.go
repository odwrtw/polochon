package papi

import (
	"fmt"

	polochon "github.com/odwrtw/polochon/lib"
)

// Movie struct returned by papi
type Movie struct {
	*polochon.Movie

	Subtitles []string `json:"subtitles"`
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

	return fmt.Sprintf("%s/download", uri), nil
}

// subtitleURL implements the Downloadable interface
func (m *Movie) subtitleURL(lang string) (string, error) {
	uri, err := m.uri()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/subtitles/%s/download", uri, lang), nil
}

// GetMovies returns all the movies in the polochon library
func (c *Client) GetMovies() (*MovieCollection, error) {
	url := fmt.Sprintf("%s/%s", c.endpoint, "movies")

	index := map[string]*struct {
		polochon.Movie
		Subtitles []string `json:"subtitles"`
	}{}
	if err := c.get(url, &index); err != nil {
		return nil, err
	}

	mc := NewMovieCollection()
	for id, m := range index {
		m.ImdbID = id
		movie := &Movie{
			Movie:     &m.Movie,
			Subtitles: m.Subtitles,
		}
		mc.Add(movie)
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
