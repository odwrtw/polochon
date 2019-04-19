package canape

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gopkg.in/yaml.v2"

	polochon "github.com/odwrtw/polochon/lib"
	"github.com/sirupsen/logrus"
)

// Make sure that the module is a wishlister
var _ polochon.Wishlister = (*Wishlist)(nil)

// Register a new Subtitiler
func init() {
	polochon.RegisterModule(&Wishlist{})
}

// Module constants
const (
	moduleName = "canape"
)

// UserWishlist represents the configurations to get a user wishlist
type UserWishlist struct {
	URL   string `yaml:"url"`
	Token string `yaml:"token"`
}

type movieResponse struct {
	Status string  `json:"status"`
	Movies []movie `json:"data"`
}

type showResponse struct {
	Status string   `json:"status"`
	Shows  []tvShow `json:"data"`
}

type tvShow struct {
	ImdbID  string `json:"imdb_id"`
	Season  int    `json:"tracked_season"`
	Episode int    `json:"tracked_episode"`
}

type movie struct {
	ImdbID string `json:"imdb_id"`
}

// Wishlist holds the canape wishlists
type Wishlist struct {
	*Params
	configured bool
}

// Params represents the module params
type Params struct {
	Wishlists []UserWishlist `yaml:"wishlists"`
}

// Init implements the module interface
func (w *Wishlist) Init(p []byte) error {
	if w.configured {
		return nil
	}

	params := &Params{}
	if err := yaml.Unmarshal(p, params); err != nil {
		return err
	}

	return w.InitWithParams(params)
}

// InitWithParams configures the module
func (w *Wishlist) InitWithParams(params *Params) error {
	w.Params = params
	return nil
}

// Name implements the Module interface
func (w *Wishlist) Name() string {
	return moduleName
}

// Status implements the Module interface
func (w *Wishlist) Status() (polochon.ModuleStatus, error) {
	return polochon.StatusNotImplemented, nil
}

// GetMovieWishlist gets the movies wishlist
func (w *Wishlist) GetMovieWishlist(log *logrus.Entry) ([]*polochon.WishedMovie, error) {
	wl, err := w.getMovieWishlists()
	if err != nil {
		return nil, err
	}

	return wl.Movies, nil
}

// GetShowWishlist gets the show wishlist
func (w *Wishlist) GetShowWishlist(log *logrus.Entry) ([]*polochon.WishedShow, error) {
	wl, err := w.getShowWishlists()
	if err != nil {
		return nil, err
	}

	return wl.Shows, nil
}

// Get all the users movie wishlists
func (w *Wishlist) getMovieWishlists() (*polochon.Wishlist, error) {
	wl := &polochon.Wishlist{}

	for _, userWishlist := range w.Wishlists {
		movies, err := userWishlist.getMovieWishlist()
		if err != nil {
			return nil, err
		}

		// Add the movies
		for _, movie := range movies {
			if err := wl.AddMovie(&polochon.WishedMovie{ImdbID: movie.ImdbID}); err != nil {
				return nil, err
			}
		}
	}

	return wl, nil
}

// Get all the users show wishlists
func (w *Wishlist) getShowWishlists() (*polochon.Wishlist, error) {
	wl := &polochon.Wishlist{}

	for _, userWishlist := range w.Wishlists {
		showList, err := userWishlist.getShowWishlist()
		if err != nil {
			return nil, err
		}

		// Add the shows
		for _, s := range showList {
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

// Get a user's show wishlist
func (w *UserWishlist) getShowWishlist() ([]tvShow, error) {
	wishlist := &showResponse{}
	err := w.request("wishlist/shows", wishlist)
	if err != nil {
		return nil, err
	}
	return wishlist.Shows, nil
}

// Get a user's movie wishlist
func (w *UserWishlist) getMovieWishlist() ([]movie, error) {
	wishlist := &movieResponse{}
	err := w.request("wishlist/movies", wishlist)
	if err != nil {
		return nil, err
	}
	return wishlist.Movies, nil

}

func (w *UserWishlist) request(URL string, response interface{}) error {
	// Create a new request
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/%s", w.URL, URL), nil)
	if err != nil {
		return err
	}

	// Add the auth headers
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", w.Token))
	req.Header.Add("Content-type", "application/json")

	// Get the page
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("canape: invalid http code %q", resp.Status)
	}

	return json.NewDecoder(resp.Body).Decode(response)
}
