package canape

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"gopkg.in/yaml.v2"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

// Module constants
const (
	moduleName = "canape"
)

// Register a new Subtitiler
func init() {
	polochon.RegisterWishlister(moduleName, NewFromRawYaml)
}

// UserConfig represents the configurations to get a user wishlist
type UserConfig struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type response struct {
	Movies []string  `json:"movies"`
	Shows  []tvShows `json:"tv_shows"`
}

type tvShows struct {
	ImdbID  string `json:"imdb_id"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
}

// Wishlist holds the canape wishlists
type Wishlist struct {
	*Params
}

// Params represents the module params
type Params struct {
	Configs []UserConfig `yaml:"wishlists"`
}

// NewFromRawYaml unmarshals the bytes as yaml as params and call the New
// function
func NewFromRawYaml(p []byte) (polochon.Wishlister, error) {
	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return nil, err
	}

	return New(params)
}

// New module
func New(params *Params) (polochon.Wishlister, error) {
	return &Wishlist{Params: params}, nil
}

// Get all the users wishlists
func (w *Wishlist) getUsersWishlists() (*polochon.Wishlist, error) {
	wl := &polochon.Wishlist{}

	for _, conf := range w.Configs {
		resp, err := w.getUserWishlists(conf.URL, conf.Token)
		if err != nil {
			return nil, err
		}

		// Add the movies
		for _, imdbID := range resp.Movies {
			if err := wl.AddMovie(&polochon.WishedMovie{ImdbID: imdbID}); err != nil {
				return nil, err
			}
		}

		// Add the shows
		for _, s := range resp.Shows {
			err := wl.AddShow(&polochon.WishedShow{
				ImdbID:  s.ImdbID,
				Season:  s.Season,
				Episode: s.Episode,
			})
			if err != nil {
				return nil, err
			}
		}
	}

	return wl, nil
}

// get a user wishlist
func (w *Wishlist) getUserWishlists(url, token string) (*response, error) {
	// Create a new request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add the auth headers
	req.Header.Add("X-Auth-Token", token)

	// Get the page
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	response := &response{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// GetMovieWishlist gets the movies wishlist
func (w *Wishlist) GetMovieWishlist(log *logrus.Entry) ([]*polochon.WishedMovie, error) {
	wl, err := w.getUsersWishlists()
	if err != nil {
		return nil, err
	}

	return wl.Movies, nil
}

// GetShowWishlist gets the show wishlist
func (w *Wishlist) GetShowWishlist(log *logrus.Entry) ([]*polochon.WishedShow, error) {
	wl, err := w.getUsersWishlists()
	if err != nil {
		return nil, err
	}

	return wl.Shows, nil
}
