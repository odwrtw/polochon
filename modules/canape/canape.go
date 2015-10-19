package canape

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/Sirupsen/logrus"
	"github.com/odwrtw/polochon/lib"
)

// Module constants
const (
	moduleName = "canape"
)

// Register a new Subtitiler
func init() {
	polochon.RegisterWishlister(moduleName, New)
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

type userConfig struct {
	url   string
	token string
}

// Wishlist holds the canape wishlists
type Wishlist struct {
	userConfigs []userConfig
}

// New module
func New(params map[string]interface{}) (polochon.Wishlister, error) {
	w, ok := params["wishlists"]
	if !ok {
		return nil, fmt.Errorf("canape: missing users wishlists")
	}

	m, ok := w.([]interface{})
	if !ok {
		return nil, fmt.Errorf("canape: users wishlist must be an array")
	}

	userConfigs := []userConfig{}
	for _, i := range m {
		values, ok := i.(map[interface{}]interface{})
		if !ok {
			return nil, fmt.Errorf("canape: invalid users wishlist configuration")
		}

		conf := map[string]string{}
		for k, v := range values {
			conf[k.(string)] = v.(string)
		}

		url, ok := conf["url"]
		if !ok {
			return nil, fmt.Errorf("canape: invalid users wishlist configuration, missing url")
		}

		token, ok := conf["token"]
		if !ok {
			return nil, fmt.Errorf("canape: invalid users wishlist configuration, missing token")
		}

		userConfigs = append(userConfigs, userConfig{
			url:   url,
			token: token,
		})
	}

	return &Wishlist{
		userConfigs: userConfigs,
	}, nil
}

// Get all the users wishlists
func (w *Wishlist) getUsersWishlists() (*polochon.Wishlist, error) {
	wl := &polochon.Wishlist{}

	for _, conf := range w.userConfigs {
		resp, err := w.getUserWishlists(conf.url, conf.token)
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
